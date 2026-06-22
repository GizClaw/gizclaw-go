package chatroom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/genx"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/agenthost"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func TestFactoryCreatesChatRoomAgent(t *testing.T) {
	params := validWorkspaceParameters(t)
	agent, err := (Factory{}).NewAgent(context.Background(), agenthost.Spec{
		Workspace: apitypes.Workspace{Name: "demo", Parameters: &params},
		Workflow:  validWorkflow(),
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if agent == nil {
		t.Fatal("NewAgent() = nil")
	}
}

func TestFactoryRejectsInvalidSpec(t *testing.T) {
	for name, tc := range map[string]struct {
		spec    agenthost.Spec
		wantErr string
	}{
		"missing chatroom spec": {
			spec: agenthost.Spec{
				Workflow: apitypes.WorkflowDocument{
					Spec: apitypes.WorkflowSpec{Driver: apitypes.WorkflowDriverChatroom},
				},
			},
			wantErr: "spec.chatroom is required",
		},
		"wrong workspace parameters": {
			spec: agenthost.Spec{
				Workflow:  validWorkflow(),
				Workspace: apitypes.Workspace{Parameters: rawWorkspaceParameters(t, `{"agent_type":"flowcraft"}`)},
			},
			wantErr: "unsupported agent_type",
		},
		"bad workspace mode": {
			spec: agenthost.Spec{
				Workflow:  validWorkflow(),
				Workspace: apitypes.Workspace{Parameters: rawWorkspaceParameters(t, `{"agent_type":"chatroom","mode":"bad"}`)},
			},
			wantErr: "unsupported mode",
		},
		"bad workspace input": {
			spec: agenthost.Spec{
				Workflow:  validWorkflow(),
				Workspace: apitypes.Workspace{Parameters: rawWorkspaceParameters(t, `{"agent_type":"chatroom","input":"bad"}`)},
			},
			wantErr: "unsupported input",
		},
		"transcript enabled without transformer": {
			spec: agenthost.Spec{
				Workflow: validWorkflowWithTranscript("asr", true),
			},
			wantErr: "transformer is required",
		},
		"transcript enabled without asr model": {
			spec: agenthost.Spec{
				Workflow: validWorkflowWithTranscript("", true),
			},
			wantErr: "transcript.asr_model is required",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := (Factory{}).NewAgent(context.Background(), tc.spec)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("NewAgent() error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestAgentTransformDrainsInputWithoutOutput(t *testing.T) {
	agent, err := (Factory{}).NewAgent(context.Background(), agenthost.Spec{
		Workflow: validWorkflow(),
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	input := &recordingStream{
		chunks: []*genx.MessageChunk{
			{Role: genx.RoleUser, Part: genx.Text("hello")},
			genx.NewTextEndOfStream(),
		},
		doneErr: genx.ErrDone,
	}
	output, err := agent.Transform(context.Background(), "demo", input)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if chunk, err := output.Next(); !errors.Is(err, genx.ErrDone) || chunk != nil {
		t.Fatalf("output.Next() = %#v, %v; want ErrDone without chunks", chunk, err)
	}
	if !input.waitClosed(100 * time.Millisecond) {
		t.Fatal("input stream was not closed")
	}
	if input.nexts != 3 {
		t.Fatalf("input Next calls = %d, want 3", input.nexts)
	}
}

func TestAgentTransformRejectsNilInput(t *testing.T) {
	agent, err := (Factory{}).NewAgent(context.Background(), agenthost.Spec{
		Workflow: validWorkflow(),
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if _, err := agent.Transform(context.Background(), "demo", nil); err == nil || !strings.Contains(err.Error(), "input stream is required") {
		t.Fatalf("Transform(nil) error = %v, want input stream error", err)
	}
}

func TestAgentTransformPropagatesInputError(t *testing.T) {
	agent, err := (Factory{}).NewAgent(context.Background(), agenthost.Spec{
		Workflow: validWorkflow(),
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	want := errors.New("input failed")
	output, err := agent.Transform(context.Background(), "demo", &recordingStream{doneErr: want})
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if _, err := output.Next(); !errors.Is(err, want) {
		t.Fatalf("output.Next() error = %v, want %v", err, want)
	}
}

func TestWorkspaceTranscriptOverrideDisablesWorkflowTranscript(t *testing.T) {
	params := rawWorkspaceParameters(t, `{"agent_type":"chatroom","transcript":{"enabled":false}}`)
	agent, err := (Factory{}).NewAgent(context.Background(), agenthost.Spec{
		Workflow:  validWorkflowWithTranscript("asr", true),
		Workspace: apitypes.Workspace{Parameters: params},
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if agent == nil {
		t.Fatal("NewAgent() = nil")
	}
}

func TestWorkspaceTranscriptOverrideModel(t *testing.T) {
	enabled := true
	model := "workspace-asr"
	var params apitypes.WorkspaceParameters
	if err := params.FromChatRoomWorkspaceParameters(apitypes.ChatRoomWorkspaceParameters{
		Transcript: &apitypes.ChatRoomWorkspaceTranscriptParameters{Enabled: &enabled, AsrModel: &model},
	}); err != nil {
		t.Fatalf("FromChatRoomWorkspaceParameters() error = %v", err)
	}
	transformer := &scriptedASRTransformer{text: "hello"}
	agent, err := (Factory{Transformer: transformer}).NewAgent(context.Background(), agenthost.Spec{
		Workflow:  validWorkflowWithTranscript("workflow-asr", true),
		Workspace: apitypes.Workspace{Parameters: &params},
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	output, err := agent.Transform(context.Background(), "demo", &recordingStream{
		chunks: []*genx.MessageChunk{
			{Part: &genx.Blob{MIMEType: "audio/opus", Data: []byte{1}}, Ctrl: &genx.StreamCtrl{EndOfStream: true}},
		},
		doneErr: genx.ErrDone,
	})
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	_ = drainOutput(t, output)
	if transformer.pattern != "model/workspace-asr" {
		t.Fatalf("ASR pattern = %q, want model/workspace-asr", transformer.pattern)
	}
}

func TestAgentTransformTranscriptIgnoresTextOnlyInput(t *testing.T) {
	transformer := &scriptedASRTransformer{text: "unused"}
	agent, err := (Factory{Transformer: transformer}).NewAgent(context.Background(), agenthost.Spec{
		Workflow: validWorkflowWithTranscript("asr", true),
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	input := &recordingStream{
		chunks: []*genx.MessageChunk{
			{Role: genx.RoleUser, Part: genx.Text("hello")},
			genx.NewTextEndOfStream(),
		},
		doneErr: genx.ErrDone,
	}
	output, err := agent.Transform(context.Background(), "demo", input)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if chunk, err := output.Next(); !isStreamDone(err) || chunk != nil {
		t.Fatalf("output.Next() = %#v, %v; want done", chunk, err)
	}
	if transformer.pattern != "" {
		t.Fatalf("ASR pattern = %q, want no ASR call", transformer.pattern)
	}
	if !input.waitClosed(100 * time.Millisecond) {
		t.Fatal("input stream was not closed")
	}
}

func TestAgentTransformReportsASRStartError(t *testing.T) {
	want := errors.New("asr unavailable")
	agent, err := (Factory{Transformer: errorTransformer{err: want}}).NewAgent(context.Background(), agenthost.Spec{
		Workflow: validWorkflowWithTranscript("asr", true),
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	output, err := agent.Transform(context.Background(), "demo", &recordingStream{
		chunks: []*genx.MessageChunk{
			{Part: &genx.Blob{MIMEType: "audio/opus", Data: []byte{1}}},
		},
		doneErr: genx.ErrDone,
	})
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if _, err := output.Next(); !errors.Is(err, want) {
		t.Fatalf("output.Next() error = %v, want %v", err, want)
	}
}

func TestAgentTransformReportsAudioInputError(t *testing.T) {
	want := errors.New("input failed")
	agent, err := (Factory{Transformer: &scriptedASRTransformer{text: "unused"}}).NewAgent(context.Background(), agenthost.Spec{
		Workflow: validWorkflowWithTranscript("asr", true),
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	output, err := agent.Transform(context.Background(), "demo", &recordingStream{
		chunks: []*genx.MessageChunk{
			{Part: &genx.Blob{MIMEType: "audio/opus", Data: []byte{1}}},
		},
		doneErr: want,
	})
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if _, err := output.Next(); !errors.Is(err, want) {
		t.Fatalf("output.Next() error = %v, want %v", err, want)
	}
}

func TestAgentTransformTranscribesAudioInput(t *testing.T) {
	transformer := &scriptedASRTransformer{text: "hello"}
	agent, err := (Factory{Transformer: transformer}).NewAgent(context.Background(), agenthost.Spec{
		Workflow: validWorkflowWithTranscript("asr", true),
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	input := &recordingStream{
		chunks: []*genx.MessageChunk{
			{Role: genx.RoleUser, Part: &genx.Blob{MIMEType: "audio/opus", Data: []byte{1, 2, 3}}, Ctrl: &genx.StreamCtrl{StreamID: "turn-a", Label: "input"}},
			{Role: genx.RoleUser, Part: &genx.Blob{MIMEType: "audio/opus"}, Ctrl: &genx.StreamCtrl{StreamID: "turn-a", Label: "input", EndOfStream: true}},
		},
		doneErr: genx.ErrDone,
	}
	output, err := agent.Transform(context.Background(), "demo", input)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	chunks := drainOutput(t, output)
	if transformer.pattern != "model/asr" {
		t.Fatalf("ASR pattern = %q, want model/asr", transformer.pattern)
	}
	if len(transformer.audio) != 1 || string(transformer.audio[0]) != string([]byte{1, 2, 3}) {
		t.Fatalf("ASR audio = %#v", transformer.audio)
	}
	if len(chunks) != 2 {
		t.Fatalf("output chunks = %#v, want transcript text and EOS", chunks)
	}
	if chunks[0].Role != genx.RoleUser || chunks[0].Name != transcriptLabel || chunks[0].Ctrl == nil || chunks[0].Ctrl.Label != transcriptLabel || chunks[0].Ctrl.StreamID != "turn-a" || chunks[0].Part != genx.Text("hello") {
		t.Fatalf("transcript chunk = %#v", chunks[0])
	}
	if !chunks[1].IsEndOfStream() || chunks[1].Role != genx.RoleUser || chunks[1].Ctrl == nil || chunks[1].Ctrl.StreamID != "turn-a" {
		t.Fatalf("transcript EOS = %#v", chunks[1])
	}
	if !input.waitClosed(100 * time.Millisecond) {
		t.Fatal("input stream was not closed")
	}
}

func TestChunkHelpers(t *testing.T) {
	if isAudioChunk(nil) {
		t.Fatal("isAudioChunk(nil) = true")
	}
	if isAudioChunk(&genx.MessageChunk{Part: genx.Text("hello")}) {
		t.Fatal("isAudioChunk(text) = true")
	}
	if !isAudioChunk(&genx.MessageChunk{Part: &genx.Blob{MIMEType: " Audio/OGG ; codecs=opus "}}) {
		t.Fatal("isAudioChunk(audio/ogg) = false")
	}
	chunk := textChunk("", "hello", false)
	if chunk.Ctrl == nil || chunk.Ctrl.StreamID != defaultInputStreamID {
		t.Fatalf("textChunk default stream = %#v", chunk)
	}
	if got := baseMIME(" Audio/OGG ; codecs=opus "); got != "audio/ogg" {
		t.Fatalf("baseMIME = %q, want audio/ogg", got)
	}
}

func validWorkflow() apitypes.WorkflowDocument {
	return apitypes.WorkflowDocument{
		Metadata: apitypes.WorkflowMetadata{Name: "chatroom"},
		Spec: apitypes.WorkflowSpec{
			Driver: apitypes.WorkflowDriverChatroom,
			Chatroom: &apitypes.ChatRoomWorkflowSpec{
				History: apitypes.ChatRoomWorkflowHistorySpec{},
			},
		},
	}
}

func validWorkflowWithTranscript(asrModel string, enabled bool) apitypes.WorkflowDocument {
	workflow := validWorkflow()
	if asrModel == "" {
		workflow.Spec.Chatroom.Transcript = &apitypes.ChatRoomWorkflowTranscriptSpec{Enabled: &enabled}
	} else {
		workflow.Spec.Chatroom.Transcript = &apitypes.ChatRoomWorkflowTranscriptSpec{Enabled: &enabled, AsrModel: &asrModel}
	}
	return workflow
}

func validWorkspaceParameters(t *testing.T) apitypes.WorkspaceParameters {
	t.Helper()
	mode := apitypes.ChatRoomModeDirect
	input := apitypes.WorkspaceInputModePushToTalk
	var params apitypes.WorkspaceParameters
	if err := params.FromChatRoomWorkspaceParameters(apitypes.ChatRoomWorkspaceParameters{
		Mode:  &mode,
		Input: &input,
	}); err != nil {
		t.Fatalf("FromChatRoomWorkspaceParameters() error = %v", err)
	}
	return params
}

func rawWorkspaceParameters(t *testing.T, raw string) *apitypes.WorkspaceParameters {
	t.Helper()
	var params apitypes.WorkspaceParameters
	if err := params.UnmarshalJSON([]byte(raw)); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	return &params
}

func drainOutput(t *testing.T, stream genx.Stream) []*genx.MessageChunk {
	t.Helper()
	defer stream.Close()
	var chunks []*genx.MessageChunk
	for {
		chunk, err := stream.Next()
		if isStreamDone(err) {
			return chunks
		}
		if err != nil {
			t.Fatalf("output.Next() error = %v", err)
		}
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}
}

type recordingStream struct {
	mu      sync.Mutex
	chunks  []*genx.MessageChunk
	idx     int
	doneErr error
	nexts   int
	closed  bool
}

func (s *recordingStream) Next() (*genx.MessageChunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nexts++
	if s.idx < len(s.chunks) {
		chunk := s.chunks[s.idx]
		s.idx++
		return chunk, nil
	}
	if s.doneErr != nil {
		return nil, s.doneErr
	}
	return nil, genx.ErrDone
}

func (s *recordingStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *recordingStream) CloseWithError(err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if !errors.Is(err, genx.ErrDone) {
		s.doneErr = err
	}
	return nil
}

func (s *recordingStream) waitClosed(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()
		if closed {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

type scriptedASRTransformer struct {
	mu      sync.Mutex
	pattern string
	text    string
	audio   [][]byte
}

func (t *scriptedASRTransformer) Transform(_ context.Context, pattern string, input genx.Stream) (genx.Stream, error) {
	t.mu.Lock()
	t.pattern = pattern
	t.mu.Unlock()
	output := genx.NewStreamBuilder((&genx.ModelContextBuilder{}).Build(), 4)
	go func() {
		defer input.Close()
		for {
			chunk, err := input.Next()
			if err != nil {
				if errors.Is(err, io.EOF) || isStreamDone(err) {
					break
				}
				_ = output.Abort(fmt.Errorf("fake ASR input: %w", err))
				return
			}
			if chunk == nil {
				continue
			}
			if blob, ok := chunk.Part.(*genx.Blob); ok && len(blob.Data) > 0 {
				t.mu.Lock()
				t.audio = append(t.audio, append([]byte(nil), blob.Data...))
				t.mu.Unlock()
			}
			if chunk.IsEndOfStream() {
				break
			}
		}
		_ = output.Add(
			&genx.MessageChunk{Part: genx.Text(t.text)},
			&genx.MessageChunk{Part: genx.Text(""), Ctrl: &genx.StreamCtrl{EndOfStream: true}},
		)
		_ = output.Done(genx.Usage{})
	}()
	return output.Stream(), nil
}

type errorTransformer struct {
	err error
}

func (t errorTransformer) Transform(context.Context, string, genx.Stream) (genx.Stream, error) {
	return nil, t.err
}
