package voiceprint

import (
	"bytes"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/audio/pcm"
)

func TestPrepareDetectorAudio(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out, err := prepareDetectorAudio(nil, pcm.L16Mono16K)
		if err != nil {
			t.Fatalf("prepareDetectorAudio(empty): %v", err)
		}
		if out != nil {
			t.Fatalf("prepareDetectorAudio(empty) = %v, want nil", out)
		}
	})

	t.Run("odd_length", func(t *testing.T) {
		if _, err := prepareDetectorAudio([]byte{1}, pcm.L16Mono16K); err == nil {
			t.Fatal("expected odd-length pcm error")
		}
	})

	t.Run("passthrough_16k", func(t *testing.T) {
		audio := []byte{1, 0, 2, 0}
		out, err := prepareDetectorAudio(audio, pcm.L16Mono16K)
		if err != nil {
			t.Fatalf("prepareDetectorAudio(16k): %v", err)
		}
		if !bytes.Equal(out, audio) {
			t.Fatalf("prepareDetectorAudio(16k) = %v, want %v", out, audio)
		}
	})

	t.Run("resample_24k", func(t *testing.T) {
		audio := bytes.Repeat([]byte{1, 0}, 480)
		out, err := prepareDetectorAudio(audio, pcm.L16Mono24K)
		if err != nil {
			t.Fatalf("prepareDetectorAudio(24k): %v", err)
		}
		if len(out) == 0 || len(out)%2 != 0 {
			t.Fatalf("resampled output len = %d, want positive even length", len(out))
		}
	})
}

func TestCopyEmbedding(t *testing.T) {
	if out := copyEmbedding(nil); out != nil {
		t.Fatalf("copyEmbedding(nil) = %v, want nil", out)
	}

	src := []float32{1, 2, 3}
	out := copyEmbedding(src)
	if len(out) != len(src) {
		t.Fatalf("copyEmbedding len = %d, want %d", len(out), len(src))
	}
	out[0] = 99
	if src[0] != 1 {
		t.Fatal("copyEmbedding should make a defensive copy")
	}
}

func TestCosineSimilarity(t *testing.T) {
	if got := cosineSimilarity(nil, []float32{1}); got != 0 {
		t.Fatalf("cosineSimilarity(nil, x) = %f, want 0", got)
	}
	if got := cosineSimilarity([]float32{1, 0}, []float32{1, 0}); got != 1 {
		t.Fatalf("cosineSimilarity(same) = %f, want 1", got)
	}
	if got := cosineSimilarity([]float32{1, 0}, []float32{0, 1}); got != 0 {
		t.Fatalf("cosineSimilarity(orthogonal) = %f, want 0", got)
	}
}
