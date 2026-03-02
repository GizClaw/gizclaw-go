// Package resampler provides audio resampling using a pure Go implementation.
//
// It supports:
//   - Sample rate conversion (e.g., 44100Hz to 48000Hz)
//   - Channel conversion (mono to stereo or stereo to mono)
//   - Streaming interface via io.Reader
//
// The package uses high-quality resampling by default and handles 16-bit signed
// integer audio samples.
package resampler
