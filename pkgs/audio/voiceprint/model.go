package voiceprint

// model handles detector-side frontend extraction and model inference.
type model interface {
	// Extract computes one embedding vector from raw PCM16 audio bytes.
	Extract(audio []byte) ([]float32, error)

	// frontend converts PCM16 audio into frontend features.
	frontend(audio []byte) ([][]float32, error)

	// infer converts frontend features into one embedding vector.
	infer(features [][]float32) ([]float32, error)

	// Dimension returns the embedding size produced by Extract.
	Dimension() int

	// Close releases any model resources.
	Close() error
}
