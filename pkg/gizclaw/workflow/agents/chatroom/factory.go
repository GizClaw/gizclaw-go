package chatroom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/GizClaw/gizclaw-go/pkg/genx"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/agenthost"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

const Type = "chatroom"

const (
	defaultInputStreamID = "audio"
	transcriptLabel      = "transcript"
)

type Factory struct {
	Transformer genx.Transformer
}

type config struct {
	transformer       genx.Transformer
	transcriptEnabled bool
	asrModel          string
}

func (f Factory) NewAgent(_ context.Context, spec agenthost.Spec) (agenthost.Agent, error) {
	if spec.Workflow.Spec.Chatroom == nil {
		return nil, fmt.Errorf("chatroom: workflow spec.chatroom is required")
	}
	cfg := config{transformer: f.Transformer}
	if spec.Workflow.Spec.Chatroom.Transcript != nil {
		cfg.transcriptEnabled = boolValue(spec.Workflow.Spec.Chatroom.Transcript.Enabled)
		cfg.asrModel = stringValue(spec.Workflow.Spec.Chatroom.Transcript.AsrModel)
	}
	if spec.Workspace.Parameters != nil {
		typed, err := spec.Workspace.Parameters.AsChatRoomWorkspaceParameters()
		if err != nil {
			return nil, fmt.Errorf("chatroom: decode workspace parameters: %w", err)
		}
		if !typed.AgentType.Valid() {
			return nil, fmt.Errorf("chatroom: unsupported agent_type %q", typed.AgentType)
		}
		if typed.Mode != nil && !typed.Mode.Valid() {
			return nil, fmt.Errorf("chatroom: unsupported mode %q", *typed.Mode)
		}
		if typed.Input != nil && !typed.Input.Valid() {
			return nil, fmt.Errorf("chatroom: unsupported input %q", *typed.Input)
		}
		mergeWorkspaceTranscriptConfig(&cfg, typed)
	}
	if cfg.transcriptEnabled {
		if cfg.asrModel == "" {
			return nil, fmt.Errorf("chatroom: transcript.asr_model is required when transcript is enabled")
		}
		if cfg.transformer == nil {
			return nil, fmt.Errorf("chatroom: transformer is required when transcript is enabled")
		}
	}
	return agenthost.NewTransformerAgent(agent{cfg: cfg}), nil
}

type agent struct {
	cfg config
}

func (a agent) Transform(ctx context.Context, _ string, input genx.Stream) (genx.Stream, error) {
	if input == nil {
		return nil, fmt.Errorf("chatroom: input stream is required")
	}
	builder := genx.NewStreamBuilder((&genx.ModelContextBuilder{}).Build(), 1)
	if a.cfg.transcriptEnabled {
		go a.transcribeInput(ctx, input, builder)
	} else {
		go drainInput(ctx, input, builder)
	}
	return builder.Stream(), nil
}

func mergeWorkspaceTranscriptConfig(cfg *config, params apitypes.ChatRoomWorkspaceParameters) {
	if cfg == nil || params.Transcript == nil {
		return
	}
	if params.Transcript.Enabled != nil {
		cfg.transcriptEnabled = *params.Transcript.Enabled
	}
	if model := strings.TrimSpace(stringValue(params.Transcript.AsrModel)); model != "" {
		cfg.asrModel = model
	}
}

func drainInput(ctx context.Context, input genx.Stream, builder *genx.StreamBuilder) {
	defer input.Close()
	for {
		if err := ctx.Err(); err != nil {
			_ = builder.Abort(err)
			return
		}
		_, err := input.Next()
		switch {
		case err == nil:
			continue
		case isStreamDone(err):
			_ = builder.Done(genx.Usage{})
			return
		default:
			_ = builder.Abort(err)
			return
		}
	}
}

func (a agent) transcribeInput(ctx context.Context, input genx.Stream, output *genx.StreamBuilder) {
	defer input.Close()
	var asrInput *genx.StreamBuilder
	var asr genx.Stream
	var readDone chan error
	streamID := &lockedString{value: defaultInputStreamID}
	startASR := func() error {
		if readDone != nil {
			return nil
		}
		asrInput = genx.NewStreamBuilder((&genx.ModelContextBuilder{}).Build(), 64)
		var err error
		asr, err = a.cfg.transformer.Transform(ctx, "model/"+a.cfg.asrModel, asrInput.Stream())
		if err != nil {
			return fmt.Errorf("chatroom: start ASR: %w", err)
		}
		readDone = make(chan error, 1)
		go func() {
			defer asr.Close()
			readDone <- readTranscript(ctx, asr, output, streamID)
		}()
		return nil
	}

	audioSeen := false
	for {
		if err := ctx.Err(); err != nil {
			if asrInput != nil {
				_ = asrInput.Abort(err)
			}
			_ = output.Abort(err)
			return
		}
		chunk, err := input.Next()
		if err != nil {
			if !isStreamDone(err) {
				if asrInput != nil {
					_ = asrInput.Abort(err)
				}
				_ = output.Abort(err)
				return
			}
			if !audioSeen {
				_ = output.Done(genx.Usage{})
				return
			}
			if err := asrInput.Done(genx.Usage{}); err != nil {
				_ = output.Abort(err)
				return
			}
			if err := <-readDone; err != nil {
				_ = output.Abort(err)
				return
			}
			_ = output.Done(genx.Usage{})
			return
		}
		if chunk == nil {
			continue
		}
		if chunk.Ctrl != nil && strings.TrimSpace(chunk.Ctrl.StreamID) != "" {
			streamID.Set(strings.TrimSpace(chunk.Ctrl.StreamID))
		}
		if !isAudioChunk(chunk) {
			continue
		}
		audioSeen = true
		if err := startASR(); err != nil {
			_ = output.Abort(err)
			return
		}
		next := chunk.Clone()
		if next.Ctrl == nil {
			next.Ctrl = &genx.StreamCtrl{}
		}
		if strings.TrimSpace(next.Ctrl.StreamID) == "" {
			next.Ctrl.StreamID = streamID.Get()
		}
		if err := asrInput.Add(next); err != nil {
			_ = output.Abort(err)
			return
		}
		if chunk.IsEndOfStream() {
			if err := asrInput.Done(genx.Usage{}); err != nil {
				_ = output.Abort(err)
				return
			}
			if err := <-readDone; err != nil {
				_ = output.Abort(err)
				return
			}
			_ = output.Done(genx.Usage{})
			return
		}
	}
}

func readTranscript(ctx context.Context, asr genx.Stream, output *genx.StreamBuilder, streamID *lockedString) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		chunk, err := asr.Next()
		if err != nil {
			if isStreamDone(err) {
				return nil
			}
			return fmt.Errorf("chatroom: read ASR: %w", err)
		}
		if chunk == nil {
			continue
		}
		text, hasText := chunk.Part.(genx.Text)
		if hasText && text != "" {
			if err := output.Add(textChunk(streamID.Get(), string(text), false)); err != nil {
				return err
			}
		}
		if chunk.IsEndOfStream() {
			if err := output.Add(textChunk(streamID.Get(), "", true)); err != nil {
				return err
			}
		}
	}
}

func textChunk(streamID, text string, eos bool) *genx.MessageChunk {
	if strings.TrimSpace(streamID) == "" {
		streamID = defaultInputStreamID
	}
	return &genx.MessageChunk{
		Role: genx.RoleUser,
		Name: transcriptLabel,
		Part: genx.Text(text),
		Ctrl: &genx.StreamCtrl{StreamID: streamID, Label: transcriptLabel, EndOfStream: eos},
	}
}

func isAudioChunk(chunk *genx.MessageChunk) bool {
	if chunk == nil {
		return false
	}
	blob, ok := chunk.Part.(*genx.Blob)
	return ok && strings.HasPrefix(baseMIME(blob.MIMEType), "audio/")
}

func baseMIME(mimeType string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if i := strings.IndexByte(mimeType, ';'); i >= 0 {
		mimeType = strings.TrimSpace(mimeType[:i])
	}
	return mimeType
}

func isStreamDone(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, genx.ErrDone)
}

type lockedString struct {
	mu    sync.RWMutex
	value string
}

func (s *lockedString) Set(value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
}

func (s *lockedString) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func boolValue(v *bool) bool {
	return v != nil && *v
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
