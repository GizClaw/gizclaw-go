package voiceprint

import (
	"io"

	"github.com/GizClaw/gizclaw-go/pkg/audio/pcm"
)

// DetectResult is the high-level candidate result produced while scanning
// a PCM stream.
type DetectResult struct {
	Label      string
	Confidence float32
	Embedding  []float32
	Bytes      int64
}

// DetectorConfig configures detector-side label assignment behavior.
type DetectorConfig struct {
	// VoiceLabelPrefix controls the generated vecid label prefix
	// (for example "voice" -> "voice:001").
	VoiceLabelPrefix string
}

// DetectCallback decides whether the current candidate result is strong enough
// to stop reading more audio from the current stream.
type DetectCallback interface {
	OnDetect(DetectResult) (stop bool, err error)
}

// ConfidentGt stops once the detector confidence is strictly greater than
// the configured threshold.
type ConfidentGt float32

// OnDetect implements DetectCallback.
func (v ConfidentGt) OnDetect(result DetectResult) (bool, error) {
	return result.Confidence > float32(v), nil
}

// Detector scans a PCM stream, produces candidate labels incrementally,
// and lets the caller decide when to stop reading more audio.
type Detector interface {
	// Detect classifies against the current registry state without mutating it.
	Detect(format pcm.Format, r io.Reader, fn DetectCallback) (DetectResult, error)
	// DetectAndUpdate classifies and then writes the final embedding back into
	// the registry, followed by a recluster pass.
	DetectAndUpdate(format pcm.Format, r io.Reader, fn DetectCallback) (DetectResult, error)
	Reset()
}
