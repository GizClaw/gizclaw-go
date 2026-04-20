package voiceprint

import (
	"bytes"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/audio/pcm"
	"github.com/GizClaw/gizclaw-go/pkg/vecid"
)

func TestERes2NetDetectorDetectAndUpdateCommitsEmbedding(t *testing.T) {
	model := makeStubDetectorModel()
	reg := vecid.New(vecid.Config{
		Dim:        model.Dimension(),
		Threshold:  0.65,
		MinSamples: 1,
		Prefix:     "voice",
	}, nil)
	reg.Identify([]float32{1, 0})
	reg.Recluster()

	d := &eres2netDetector{
		ncnnDetector: &ncnnDetector{
			model:        model,
			registry:     reg,
			threshold:    0.65,
			readBytes:    2,
			minBytes:     2,
			segmentBytes: 2,
			hopBytes:     2,
			name:         "eres2net",
		},
	}

	result, err := d.DetectAndUpdate(pcm.L16Mono16K, bytes.NewReader([]byte{1, 0, 1, 0}), ConfidentGt(0.5))
	if err != nil {
		t.Fatalf("DetectAndUpdate: %v", err)
	}
	if len(result.Embedding) != 2 || result.Embedding[0] != 1 || result.Embedding[1] != 0 {
		t.Fatalf("unexpected embedding: %v", result.Embedding)
	}
	if got := d.registry.Len(); got != 2 {
		t.Fatalf("registry len = %d, want 2", got)
	}
}
