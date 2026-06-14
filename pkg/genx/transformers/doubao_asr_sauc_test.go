package transformers

import (
	"context"
	"errors"
	"io"
	"iter"
	"slices"
	"testing"

	doubaospeech "github.com/GizClaw/doubao-speech-go"
	"github.com/GizClaw/gizclaw-go/pkg/genx"
)

type fakeDoubaoASRSend struct {
	data   []byte
	isLast bool
}

type fakeDoubaoASRSession struct {
	sends  []fakeDoubaoASRSend
	result chan *doubaospeech.ASRV2Result
}

func newFakeDoubaoASRSession() *fakeDoubaoASRSession {
	return &fakeDoubaoASRSession{result: make(chan *doubaospeech.ASRV2Result, 1)}
}

func (s *fakeDoubaoASRSession) SendAudio(_ context.Context, data []byte, isLast bool) error {
	s.sends = append(s.sends, fakeDoubaoASRSend{data: slices.Clone(data), isLast: isLast})
	if isLast {
		s.result <- &doubaospeech.ASRV2Result{
			Text:    "recognized text",
			IsFinal: true,
			Utterances: []doubaospeech.ASRV2Utterance{
				{Text: "recognized text", EndTime: 100},
			},
		}
		close(s.result)
	}
	return nil
}

func (s *fakeDoubaoASRSession) Recv() iter.Seq2[*doubaospeech.ASRV2Result, error] {
	return func(yield func(*doubaospeech.ASRV2Result, error) bool) {
		for result := range s.result {
			if !yield(result, nil) {
				return
			}
		}
	}
}

func (s *fakeDoubaoASRSession) Close() error {
	return nil
}

func TestDoubaoASRSAUCSendsLastNonEmptyAudioFrame(t *testing.T) {
	session := newFakeDoubaoASRSession()
	transformer := NewDoubaoASRSAUC(nil)
	transformer.newSession = func(context.Context) (doubaoASRSession, error) {
		return session, nil
	}

	input := newBufferStream(4)
	output, err := transformer.Transform(context.Background(), "asr", input)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if err := input.Push(&genx.MessageChunk{Part: &genx.Blob{MIMEType: "audio/mp3", Data: []byte("first")}}); err != nil {
		t.Fatalf("push first audio = %v", err)
	}
	if err := input.Push(&genx.MessageChunk{Part: &genx.Blob{MIMEType: "audio/mp3", Data: []byte("second")}}); err != nil {
		t.Fatalf("push second audio = %v", err)
	}
	if err := input.Push(genx.NewEndOfStream("audio/mp3")); err != nil {
		t.Fatalf("push eos = %v", err)
	}
	if err := input.Close(); err != nil {
		t.Fatalf("close input = %v", err)
	}

	chunk, err := output.Next()
	if err != nil {
		t.Fatalf("output first chunk = %v", err)
	}
	if got := chunk.Part.(genx.Text); got != "recognized text" {
		t.Fatalf("output text = %q, want recognized text", got)
	}
	chunk, err = output.Next()
	if err != nil {
		t.Fatalf("output eos chunk = %v", err)
	}
	if chunk == nil || !chunk.IsEndOfStream() {
		t.Fatalf("output eos chunk = %#v", chunk)
	}
	if _, err := output.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("output final error = %v, want EOF", err)
	}

	if len(session.sends) != 2 {
		t.Fatalf("SendAudio calls = %#v, want two non-empty frames", session.sends)
	}
	if got := string(session.sends[0].data); got != "first" || session.sends[0].isLast {
		t.Fatalf("first SendAudio = data %q last %t, want first/false", got, session.sends[0].isLast)
	}
	if got := string(session.sends[1].data); got != "second" || !session.sends[1].isLast {
		t.Fatalf("second SendAudio = data %q last %t, want second/true", got, session.sends[1].isLast)
	}
}
