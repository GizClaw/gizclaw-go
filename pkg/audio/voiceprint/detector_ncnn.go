package voiceprint

import (
	"fmt"
	"io"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/audio/pcm"
	"github.com/GizClaw/gizclaw-go/pkg/vecid"
)

const (
	defaultDetectorReadDuration     = 500 * time.Millisecond
	defaultDetectorMinDuration      = 3 * time.Second
	defaultDetectorSegmentDuration  = 3 * time.Second
	defaultDetectorHopDuration      = 1500 * time.Millisecond
	defaultDetectorVecIDThreshold   = 0.65
	defaultDetectorClusterThreshold = 0.60
	defaultDetectorVecIDMinSamples  = 1
	defaultDetectorVoiceLabelPrefix = "voice"
)

type ncnnDetector struct {
	model        model
	registry     *vecid.Registry
	threshold    float32
	readBytes    int
	minBytes     int
	segmentBytes int
	hopBytes     int
	name         string
}

func normalizeDetectorConfig(cfg DetectorConfig) DetectorConfig {
	if cfg.VoiceLabelPrefix == "" {
		cfg.VoiceLabelPrefix = defaultDetectorVoiceLabelPrefix
	}
	return cfg
}

func newNCNNDetector(model model, name string, threshold float32, cfg DetectorConfig) *ncnnDetector {
	cfg = normalizeDetectorConfig(cfg)
	return &ncnnDetector{
		model:     model,
		name:      name,
		threshold: threshold,
		registry: vecid.New(vecid.Config{
			Dim:              model.Dimension(),
			Threshold:        threshold,
			ClusterThreshold: defaultDetectorClusterThreshold,
			MinSamples:       defaultDetectorVecIDMinSamples,
			Prefix:           cfg.VoiceLabelPrefix,
		}, nil),
	}
}

func (d *ncnnDetector) Detect(format pcm.Format, r io.Reader, fn DetectCallback) (DetectResult, error) {
	return d.detect(format, r, fn, false)
}

func (d *ncnnDetector) DetectAndUpdate(format pcm.Format, r io.Reader, fn DetectCallback) (DetectResult, error) {
	return d.detect(format, r, fn, true)
}

func (d *ncnnDetector) detect(format pcm.Format, r io.Reader, fn DetectCallback, update bool) (DetectResult, error) {
	if r == nil {
		return DetectResult{}, fmt.Errorf("voiceprint: detector reader is nil")
	}

	readBytes := d.readChunkBytes(format)
	minBytes := d.detectMinBytes(format, readBytes)
	buf := make([]byte, readBytes)
	pending := make([]byte, 0, minBytes)
	var last DetectResult
	stopped := false

	for {
		chunk, eof, readErr := readDetectorChunk(r, buf)
		if readErr != nil {
			return last, readErr
		}
		if len(chunk) > 0 {
			pending = append(pending, chunk...)
		}

		if len(pending) >= minBytes || (eof && len(pending) > 0) {
			emb, err := d.aggregateEmbedding(pending, format)
			if err != nil {
				return last, err
			}
			if len(emb) > 0 {
				label, confidence := d.identifyEmbedding(emb)
				last = DetectResult{
					Label:      label,
					Confidence: confidence,
					Embedding:  copyEmbedding(emb),
					Bytes:      int64(len(pending)),
				}
				if fn != nil {
					stop, err := fn.OnDetect(last)
					if err != nil {
						return last, err
					}
					if stop {
						stopped = true
						break
					}
				}
			}
		}

		if eof {
			break
		}
	}

	if len(last.Embedding) > 0 {
		if !update {
			return last, nil
		}
		d.registry.Identify(last.Embedding)
		d.registry.Recluster()
		label, confidence := d.identifyEmbedding(last.Embedding)
		last.Label = label
		last.Confidence = confidence
		return last, nil
	}
	if stopped {
		return last, nil
	}
	return last, io.EOF
}

func (d *ncnnDetector) Reset() {
	if d.registry != nil {
		d.registry.Reset()
	}
}

func (d *ncnnDetector) Close() error {
	if d == nil || d.model == nil {
		return nil
	}
	return d.model.Close()
}

func (d *ncnnDetector) readChunkBytes(format pcm.Format) int {
	if d.readBytes > 0 {
		return evenByteCount(d.readBytes)
	}
	return evenByteCount(int(format.BytesInDuration(defaultDetectorReadDuration)))
}

func (d *ncnnDetector) detectMinBytes(format pcm.Format, readBytes int) int {
	if d.minBytes > 0 {
		return evenByteCount(d.minBytes)
	}
	n := evenByteCount(int(format.BytesInDuration(defaultDetectorMinDuration)))
	if n < readBytes {
		return readBytes
	}
	return n
}

func (d *ncnnDetector) detectSegmentBytes() int {
	if d.segmentBytes > 0 {
		return evenByteCount(d.segmentBytes)
	}
	return evenByteCount(int(pcm.L16Mono16K.BytesInDuration(defaultDetectorSegmentDuration)))
}

func (d *ncnnDetector) detectHopBytes(segmentBytes int) int {
	if d.hopBytes > 0 {
		return evenByteCount(d.hopBytes)
	}
	n := evenByteCount(int(pcm.L16Mono16K.BytesInDuration(defaultDetectorHopDuration)))
	if n <= 0 {
		return segmentBytes
	}
	return n
}

func (d *ncnnDetector) aggregateEmbedding(audio []byte, format pcm.Format) ([]float32, error) {
	audio, err := prepareDetectorAudio(audio, format)
	if err != nil {
		return nil, err
	}
	if len(audio) == 0 {
		return nil, nil
	}
	return d.aggregateEmbeddingWithFrontend(d.model, audio)
}

func (d *ncnnDetector) aggregateEmbeddingWithFrontend(model model, audio []byte) ([]float32, error) {
	segmentBytes := d.detectSegmentBytes()
	hopBytes := d.detectHopBytes(segmentBytes)
	if len(audio) <= segmentBytes {
		features, err := model.frontend(audio)
		if err != nil {
			return nil, err
		}
		return model.infer(features)
	}

	var (
		embeddings [][]float32
		lastStart  int
	)
	for start := 0; start+segmentBytes <= len(audio); start += hopBytes {
		features, err := model.frontend(audio[start : start+segmentBytes])
		if err != nil {
			continue
		}
		emb, err := model.infer(features)
		if err != nil {
			continue
		}
		l2Normalize(emb)
		embeddings = append(embeddings, emb)
		lastStart = start
	}

	if tail := len(audio) - segmentBytes; tail > lastStart {
		features, err := model.frontend(audio[tail:])
		if err == nil {
			emb, err := model.infer(features)
			if err == nil {
				l2Normalize(emb)
				embeddings = append(embeddings, emb)
			}
		}
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("voiceprint: %s detector produced no segment embeddings", d.name)
	}
	return averageEmbeddings(embeddings), nil
}

func (d *ncnnDetector) identifyEmbedding(emb []float32) (string, float32) {
	buckets := d.registry.Buckets()
	if len(buckets) == 0 {
		return "", 0
	}
	bestID := ""
	bestSim := float32(-1)
	for _, bucket := range buckets {
		sim := cosineSimilarity(emb, bucket.Centroid)
		if sim > bestSim {
			bestSim = sim
			bestID = bucket.ID
		}
	}
	if bestSim >= d.threshold {
		return bestID, bestSim
	}
	return "", 0
}

func averageEmbeddings(embeddings [][]float32) []float32 {
	if len(embeddings) == 0 {
		return nil
	}
	avg := make([]float32, len(embeddings[0]))
	for _, emb := range embeddings {
		for i, v := range emb {
			avg[i] += v
		}
	}
	n := float32(len(embeddings))
	for i := range avg {
		avg[i] /= n
	}
	l2Normalize(avg)
	return avg
}

func evenByteCount(n int) int {
	if n <= 0 {
		return 2
	}
	if n%2 != 0 {
		n--
	}
	if n < 2 {
		return 2
	}
	return n
}

func readDetectorChunk(r io.Reader, buf []byte) ([]byte, bool, error) {
	n, err := io.ReadFull(r, buf)
	switch err {
	case nil:
		return buf[:n], false, nil
	case io.EOF:
		return nil, true, nil
	case io.ErrUnexpectedEOF:
		return buf[:n], true, nil
	default:
		return nil, false, err
	}
}
