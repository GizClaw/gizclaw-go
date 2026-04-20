package voiceprint

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/audio/pcm"
	"github.com/GizClaw/gizclaw-go/pkg/vecid"
)

type frontendOnlyDetectorModel struct {
	frontendCalls int
	inferCalls    int
}

func (m *frontendOnlyDetectorModel) Extract(audio []byte) ([]float32, error) {
	panic("Extract should not be called when frontend/infer is available")
}

func (m *frontendOnlyDetectorModel) frontend(audio []byte) ([][]float32, error) {
	m.frontendCalls++
	return [][]float32{{1, 0}}, nil
}

func (m *frontendOnlyDetectorModel) infer(features [][]float32) ([]float32, error) {
	m.inferCalls++
	return []float32{1, 0}, nil
}

func (m *frontendOnlyDetectorModel) Dimension() int { return 2 }
func (m *frontendOnlyDetectorModel) Close() error   { return nil }

func TestNCNNDetectorUsesFrontendPathWhenAvailable(t *testing.T) {
	model := &frontendOnlyDetectorModel{}
	reg := vecid.New(vecid.Config{
		Dim:        model.Dimension(),
		Threshold:  0.65,
		MinSamples: 1,
		Prefix:     "voice",
	}, nil)
	reg.Identify([]float32{1, 0})
	reg.Recluster()

	d := &ncnnDetector{
		model:        model,
		registry:     reg,
		threshold:    0.65,
		readBytes:    2,
		minBytes:     2,
		segmentBytes: 2,
		hopBytes:     2,
		name:         "test",
	}

	result, err := d.Detect(pcm.L16Mono16K, bytes.NewReader([]byte{1, 0}), ConfidentGt(0.5))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(result.Embedding) != 2 || result.Embedding[0] != 1 || result.Embedding[1] != 0 {
		t.Fatalf("unexpected embedding: %v", result.Embedding)
	}
	if model.frontendCalls == 0 || model.inferCalls == 0 {
		t.Fatalf("frontend/infer should both be called, got frontend=%d infer=%d", model.frontendCalls, model.inferCalls)
	}
}

func TestNCNNDetectorDetectDoesNotUpdateRegistry(t *testing.T) {
	model := makeStubDetectorModel()
	reg := vecid.New(vecid.Config{
		Dim:        model.Dimension(),
		Threshold:  0.65,
		MinSamples: 1,
		Prefix:     "voice",
	}, nil)
	reg.Identify([]float32{1, 0})
	reg.Recluster()

	d := &ncnnDetector{
		model:        model,
		registry:     reg,
		threshold:    0.65,
		readBytes:    2,
		minBytes:     2,
		segmentBytes: 2,
		hopBytes:     2,
		name:         "test",
	}

	before := d.registry.Len()
	result, err := d.Detect(pcm.L16Mono16K, bytes.NewReader([]byte{1, 0}), ConfidentGt(0.5))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if after := d.registry.Len(); after != before {
		t.Fatalf("registry len = %d, want %d", after, before)
	}
	if result.Label != "voice:001" {
		t.Fatalf("label = %q, want %q", result.Label, "voice:001")
	}
}

func TestNewNCNNDetectorUsesVoiceLabelPrefix(t *testing.T) {
	d := newNCNNDetector(makeStubDetectorModel(), "test", 0.65, DetectorConfig{
		VoiceLabelPrefix: "speaker",
	})
	d.registry.Identify([]float32{1, 0})
	d.registry.Recluster()

	buckets := d.registry.Buckets()
	if len(buckets) != 1 {
		t.Fatalf("bucket count = %d, want 1", len(buckets))
	}
	if buckets[0].ID != "speaker:001" {
		t.Fatalf("bucket id = %q, want %q", buckets[0].ID, "speaker:001")
	}
}

func TestNormalizeDetectorConfigDefault(t *testing.T) {
	cfg := normalizeDetectorConfig(DetectorConfig{})
	if cfg.VoiceLabelPrefix != defaultDetectorVoiceLabelPrefix {
		t.Fatalf("VoiceLabelPrefix = %q, want %q", cfg.VoiceLabelPrefix, defaultDetectorVoiceLabelPrefix)
	}
}

func TestNCNNDetectorHelpersAndLifecycle(t *testing.T) {
	model := makeStubDetectorModel()
	d := newNCNNDetector(model, "test", 0.65, DetectorConfig{})

	if got := d.readChunkBytes(pcm.L16Mono16K); got <= 0 || got%2 != 0 {
		t.Fatalf("readChunkBytes = %d, want positive even", got)
	}
	d.readBytes = 3
	if got := d.readChunkBytes(pcm.L16Mono16K); got != 2 {
		t.Fatalf("readChunkBytes(custom) = %d, want 2", got)
	}

	d.minBytes = 3
	if got := d.detectMinBytes(pcm.L16Mono16K, 4); got != 2 {
		t.Fatalf("detectMinBytes(custom) = %d, want 2", got)
	}
	d.minBytes = 0

	d.segmentBytes = 3
	if got := d.detectSegmentBytes(); got != 2 {
		t.Fatalf("detectSegmentBytes(custom) = %d, want 2", got)
	}

	d.hopBytes = 3
	if got := d.detectHopBytes(8); got != 2 {
		t.Fatalf("detectHopBytes(custom) = %d, want 2", got)
	}

	d.registry.Identify([]float32{1, 0})
	d.registry.Recluster()
	if d.registry.Len() == 0 {
		t.Fatal("expected registry to contain samples")
	}
	d.Reset()
	if d.registry.Len() != 0 {
		t.Fatalf("registry len after Reset = %d, want 0", d.registry.Len())
	}
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNCNNDetectorDetectErrorsAndEOF(t *testing.T) {
	d := newNCNNDetector(makeStubDetectorModel(), "test", 0.65, DetectorConfig{})
	if _, err := d.Detect(pcm.L16Mono16K, nil, nil); err == nil {
		t.Fatal("expected nil reader error")
	}

	result, err := d.Detect(pcm.L16Mono16K, bytes.NewReader(nil), nil)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Detect(empty) err = %v, want EOF", err)
	}
	if len(result.Embedding) != 0 {
		t.Fatalf("Detect(empty) result = %+v, want empty", result)
	}
}

func TestAverageEmbeddingsAndEvenByteCount(t *testing.T) {
	if got := averageEmbeddings(nil); got != nil {
		t.Fatalf("averageEmbeddings(nil) = %v, want nil", got)
	}

	got := averageEmbeddings([][]float32{{1, 0}, {1, 0}})
	if len(got) != 2 || got[0] != 1 || got[1] != 0 {
		t.Fatalf("averageEmbeddings = %v, want [1 0]", got)
	}

	if n := evenByteCount(-1); n != 2 {
		t.Fatalf("evenByteCount(-1) = %d, want 2", n)
	}
	if n := evenByteCount(3); n != 2 {
		t.Fatalf("evenByteCount(3) = %d, want 2", n)
	}
}

func TestReadDetectorChunk(t *testing.T) {
	buf := make([]byte, 4)

	chunk, eof, err := readDetectorChunk(bytes.NewReader([]byte{1, 2, 3, 4}), buf)
	if err != nil || eof || !bytes.Equal(chunk, []byte{1, 2, 3, 4}) {
		t.Fatalf("readDetectorChunk(full) = chunk=%v eof=%v err=%v", chunk, eof, err)
	}

	chunk, eof, err = readDetectorChunk(bytes.NewReader([]byte{1, 2}), buf)
	if err != nil || !eof || !bytes.Equal(chunk, []byte{1, 2}) {
		t.Fatalf("readDetectorChunk(short) = chunk=%v eof=%v err=%v", chunk, eof, err)
	}

	chunk, eof, err = readDetectorChunk(bytes.NewReader(nil), buf)
	if err != nil || !eof || len(chunk) != 0 {
		t.Fatalf("readDetectorChunk(empty) = chunk=%v eof=%v err=%v", chunk, eof, err)
	}
}

func TestAggregateEmbeddingWithFrontendSegments(t *testing.T) {
	d := &ncnnDetector{
		model:        makeStubDetectorModel(),
		registry:     vecid.New(vecid.Config{Dim: 2, Threshold: 0.65, MinSamples: 1, Prefix: "voice"}, nil),
		threshold:    0.65,
		segmentBytes: 2,
		hopBytes:     2,
		name:         "test",
	}

	emb, err := d.aggregateEmbeddingWithFrontend(d.model, []byte{1, 0, 2, 0, 1, 0})
	if err != nil {
		t.Fatalf("aggregateEmbeddingWithFrontend: %v", err)
	}
	if len(emb) != 2 {
		t.Fatalf("embedding len = %d, want 2", len(emb))
	}
}
