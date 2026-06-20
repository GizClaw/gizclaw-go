package transformers

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"iter"
	"testing"

	"github.com/GizClaw/doubao-speech-go"
	"github.com/GizClaw/gizclaw-go/pkg/audio/codec/ogg"
	"github.com/GizClaw/gizclaw-go/pkg/genx"
)

func TestDoubaoASTTranslateStreamsTranslationAndAudio(t *testing.T) {
	input := newBufferStream(4)
	tr := NewDoubaoASTTranslate(doubaospeech.NewClient("app-id"),
		WithDoubaoASTTranslateMode(doubaospeech.ASTTranslateModeS2S),
		WithDoubaoASTTranslateSourceLanguage("zh"),
		WithDoubaoASTTranslateTargetLanguage("ja"),
	)
	fake := &fakeASTTranslateSession{
		events: []*doubaospeech.ASTTranslateEvent{
			{Type: doubaospeech.ASTEventSourceSubtitleStart},
			{Type: doubaospeech.ASTEventSourceSubtitleResponse, Text: "你好"},
			{Type: doubaospeech.ASTEventSourceSubtitleEnd, Text: "你好"},
			{Type: doubaospeech.ASTEventTranslationSubtitleStart},
			{Type: doubaospeech.ASTEventTranslationSubtitleResponse, Text: "こんにちは"},
			{Type: doubaospeech.ASTEventTranslationSubtitleEnd, Text: "こんにちは"},
			{Type: doubaospeech.ASTEventTTSSentenceStart},
			{Type: doubaospeech.ASTEventTTSResponse, Audio: buildASTTranslateOggPackets(t, astTranslateOpusHeadPacket(48000, 1), astTranslateOpusTagsPacket("test"), []byte{1, 2, 3})},
			{Type: doubaospeech.ASTEventTTSSentenceEnd},
			{Type: doubaospeech.ASTEventSessionFinished},
		},
	}
	tr.newSession = func(_ context.Context, cfg doubaospeech.ASTTranslateConfig) (doubaoASTTranslateSession, error) {
		if cfg.Mode != doubaospeech.ASTTranslateModeS2S || cfg.SourceLanguage != "zh" || cfg.TargetLanguage != "ja" {
			t.Fatalf("cfg = %+v, want s2s zh->ja", cfg)
		}
		return fake, nil
	}
	out, err := tr.Transform(context.Background(), "", input)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if err := input.Push(&genx.MessageChunk{Ctrl: &genx.StreamCtrl{StreamID: "turn-1", BeginOfStream: true}}); err != nil {
		t.Fatalf("Push(BOS): %v", err)
	}
	if err := input.Push(&genx.MessageChunk{Part: &genx.Blob{MIMEType: "audio/pcm", Data: []byte{1, 2, 3, 4}}, Ctrl: &genx.StreamCtrl{StreamID: "turn-1"}}); err != nil {
		t.Fatalf("Push(audio): %v", err)
	}
	if err := input.Push(&genx.MessageChunk{Part: &genx.Blob{MIMEType: "audio/pcm"}, Ctrl: &genx.StreamCtrl{StreamID: "turn-1", EndOfStream: true}}); err != nil {
		t.Fatalf("Push(EOS): %v", err)
	}
	if err := input.Close(); err != nil {
		t.Fatalf("Close(input): %v", err)
	}

	chunks := readAllASTTranslateChunks(t, out)
	if len(fake.sentAudio) != 1 || len(fake.sentAudio[0]) != 4 {
		t.Fatalf("sentAudio = %v", fake.sentAudio)
	}
	if !fake.finished {
		t.Fatalf("session was not finished")
	}
	assertASTTranslateTextChunk(t, chunks, genx.RoleUser, doubaoASTTranslateTranscriptLabel, "turn-1", "你好")
	assertASTTranslateTextChunk(t, chunks, genx.RoleModel, doubaoASTTranslateAssistantLabel, "turn-1", "こんにちは")
	assertASTTranslateAudioChunk(t, chunks, "turn-1", []byte{1, 2, 3})
	assertASTTranslateEOS(t, chunks, genx.RoleUser, doubaoASTTranslateTranscriptLabel, "turn-1")
	assertASTTranslateEOS(t, chunks, genx.RoleModel, doubaoASTTranslateAssistantLabel, "turn-1")
}

type fakeASTTranslateSession struct {
	events    []*doubaospeech.ASTTranslateEvent
	sentAudio [][]byte
	finished  bool
	closed    bool
}

func (s *fakeASTTranslateSession) SendAudio(_ context.Context, audio []byte) error {
	cp := append([]byte(nil), audio...)
	s.sentAudio = append(s.sentAudio, cp)
	return nil
}

func (s *fakeASTTranslateSession) Finish(context.Context) error {
	s.finished = true
	return nil
}

func (s *fakeASTTranslateSession) Recv() iter.Seq2[*doubaospeech.ASTTranslateEvent, error] {
	return func(yield func(*doubaospeech.ASTTranslateEvent, error) bool) {
		for _, event := range s.events {
			if !yield(event, nil) {
				return
			}
		}
	}
}

func (s *fakeASTTranslateSession) Close() error {
	s.closed = true
	return nil
}

func readAllASTTranslateChunks(t *testing.T, stream genx.Stream) []*genx.MessageChunk {
	t.Helper()
	var chunks []*genx.MessageChunk
	for {
		chunk, err := stream.Next()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, genx.ErrDone) {
				return chunks
			}
			t.Fatalf("Next() error = %v", err)
		}
		chunks = append(chunks, chunk)
	}
}

func assertASTTranslateTextChunk(t *testing.T, chunks []*genx.MessageChunk, role genx.Role, label, streamID, text string) {
	t.Helper()
	for _, chunk := range chunks {
		if chunk.Role != role || chunk.Ctrl == nil || chunk.Ctrl.Label != label || chunk.Ctrl.StreamID != streamID {
			continue
		}
		if got, ok := chunk.Part.(genx.Text); ok && string(got) == text {
			return
		}
	}
	t.Fatalf("missing text chunk role=%s label=%s stream=%s text=%q in %#v", role, label, streamID, text, chunks)
}

func assertASTTranslateAudioChunk(t *testing.T, chunks []*genx.MessageChunk, streamID string, audio []byte) {
	t.Helper()
	for _, chunk := range chunks {
		if chunk.Role != genx.RoleModel || chunk.Ctrl == nil || chunk.Ctrl.Label != doubaoASTTranslateAssistantLabel || chunk.Ctrl.StreamID != streamID {
			continue
		}
		blob, ok := chunk.Part.(*genx.Blob)
		if ok && string(blob.Data) == string(audio) {
			return
		}
	}
	t.Fatalf("missing audio chunk stream=%s bytes=%v in %#v", streamID, audio, chunks)
}

func assertASTTranslateEOS(t *testing.T, chunks []*genx.MessageChunk, role genx.Role, label, streamID string) {
	t.Helper()
	for _, chunk := range chunks {
		if chunk.Role == role && chunk.Ctrl != nil && chunk.Ctrl.Label == label && chunk.Ctrl.StreamID == streamID && chunk.Ctrl.EndOfStream {
			return
		}
	}
	t.Fatalf("missing EOS role=%s label=%s stream=%s in %#v", role, label, streamID, chunks)
}

func astTranslateOpusHeadPacket(sampleRate, channels int) []byte {
	packet := make([]byte, 19)
	copy(packet[:8], "OpusHead")
	packet[8] = 1
	packet[9] = byte(channels)
	binary.LittleEndian.PutUint32(packet[12:16], uint32(sampleRate))
	return packet
}

func astTranslateOpusTagsPacket(vendor string) []byte {
	vendorBytes := []byte(vendor)
	packet := make([]byte, 8+4+len(vendorBytes)+4)
	copy(packet[:8], "OpusTags")
	binary.LittleEndian.PutUint32(packet[8:12], uint32(len(vendorBytes)))
	copy(packet[12:12+len(vendorBytes)], vendorBytes)
	return packet
}

func buildASTTranslateOggPackets(t *testing.T, packets ...[]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	sw, err := ogg.NewStreamWriter(&out, 77)
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	for i, packet := range packets {
		if _, err := sw.WritePacket(packet, uint64(i), i == len(packets)-1); err != nil {
			t.Fatalf("WritePacket %d: %v", i, err)
		}
	}
	return out.Bytes()
}
