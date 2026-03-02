// Package fbank computes log mel filterbank features from PCM audio.
//
// This is the standard front-end for speaker recognition models like
// 3D-Speaker ERes2Net. The output is a [T, numMels] float32 matrix
// suitable for direct input to ncnn inference.
//
// Default parameters match the Kaldi/3D-Speaker convention:
//
//	SampleRate:  16000
//	WindowSize:  400 (25 ms)
//	HopSize:     160 (10 ms)
//	FFTSize:     512
//	NumMels:     80
//	LowFreq:     20
//	HighFreq:  7600
//	PreEmphasis: 0.97
package fbank

import (
	"fmt"
	"math"
)

// Config controls mel filterbank extraction parameters.
type Config struct {
	SampleRate  int     // audio sample rate in Hz (default 16000)
	WindowSize  int     // window length in samples (default 400 = 25ms)
	HopSize     int     // hop length in samples (default 160 = 10ms)
	FFTSize     int     // FFT size (default 512)
	NumMels     int     // number of mel bins (default 80)
	LowFreq     float64 // lowest mel frequency (default 20)
	HighFreq    float64 // highest mel frequency (default 7600)
	PreEmphasis float64 // pre-emphasis coefficient (default 0.97)
}

// DefaultConfig returns the standard config for 3D-Speaker ERes2Net.
func DefaultConfig() Config {
	return Config{
		SampleRate:  16000,
		WindowSize:  400,
		HopSize:     160,
		FFTSize:     512,
		NumMels:     80,
		LowFreq:     20,
		HighFreq:    7600,
		PreEmphasis: 0.97,
	}
}

// Extractor computes mel filterbank features from PCM samples.
type Extractor struct {
	cfg     Config
	window  []float64 // Hamming window
	melBank [][]float64
}

// New creates a new fbank Extractor with the given config.
func New(cfg Config) (*Extractor, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	e := &Extractor{cfg: cfg}
	e.window = hammingWindow(cfg.WindowSize)
	e.melBank = melFilterBank(cfg.NumMels, cfg.FFTSize, cfg.SampleRate, cfg.LowFreq, cfg.HighFreq)
	return e, nil
}

// MustNew creates a new extractor and panics on invalid configuration.
func MustNew(cfg Config) *Extractor {
	e, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return e
}

func validateConfig(cfg Config) error {
	if cfg.SampleRate <= 0 {
		return fmt.Errorf("fbank: invalid SampleRate %d", cfg.SampleRate)
	}
	if cfg.WindowSize <= 0 {
		return fmt.Errorf("fbank: invalid WindowSize %d", cfg.WindowSize)
	}
	if cfg.HopSize <= 0 {
		return fmt.Errorf("fbank: invalid HopSize %d", cfg.HopSize)
	}
	if cfg.FFTSize <= 0 {
		return fmt.Errorf("fbank: invalid FFTSize %d", cfg.FFTSize)
	}
	if cfg.FFTSize < cfg.WindowSize {
		return fmt.Errorf("fbank: FFTSize(%d) must be >= WindowSize(%d)", cfg.FFTSize, cfg.WindowSize)
	}
	if cfg.NumMels <= 0 {
		return fmt.Errorf("fbank: invalid NumMels %d", cfg.NumMels)
	}
	if cfg.LowFreq < 0 {
		return fmt.Errorf("fbank: invalid LowFreq %f", cfg.LowFreq)
	}
	if cfg.HighFreq <= cfg.LowFreq {
		return fmt.Errorf("fbank: invalid freq range low=%f high=%f", cfg.LowFreq, cfg.HighFreq)
	}

	return nil
}

// Extract computes log mel filterbank features from PCM float32 samples.
// Input: pcm is normalized float32 audio samples (range [-1, 1]).
// Output: [T][numMels] float32 matrix where T = (len(pcm) - windowSize) / hopSize + 1.
func (e *Extractor) Extract(pcm []float32) [][]float32 {
	cfg := e.cfg
	n := len(pcm)
	if n < cfg.WindowSize {
		return nil
	}

	numFrames := (n-cfg.WindowSize)/cfg.HopSize + 1
	nfft := cfg.FFTSize
	halfFFT := nfft/2 + 1

	// Pre-allocate output.
	features := make([][]float32, numFrames)

	// Working buffers.
	frame := make([]float64, nfft)
	real := make([]float64, nfft)
	imag := make([]float64, nfft)

	for t := range numFrames {
		start := t * cfg.HopSize

		// Pre-emphasis + windowing.
		for i := 0; i < cfg.WindowSize; i++ {
			s := float64(pcm[start+i])
			if i > 0 {
				s -= cfg.PreEmphasis * float64(pcm[start+i-1])
			}
			frame[i] = s * e.window[i]
		}
		// Zero-pad.
		for i := cfg.WindowSize; i < nfft; i++ {
			frame[i] = 0
		}

		// FFT.
		copy(real, frame)
		for i := range imag {
			imag[i] = 0
		}
		FFT(real, imag)

		// Power spectrum.
		power := make([]float64, halfFFT)
		for i := range halfFFT {
			power[i] = real[i]*real[i] + imag[i]*imag[i]
		}

		// Mel filterbank.
		mel := make([]float32, cfg.NumMels)
		for m := range cfg.NumMels {
			sum := 0.0
			for k, w := range e.melBank[m] {
				sum += w * power[k]
			}
			// Log with floor to avoid -inf.
			if sum < 1e-10 {
				sum = 1e-10
			}
			mel[m] = float32(math.Log(sum))
		}
		features[t] = mel
	}

	return features
}

// ExtractFromInt16 is a convenience wrapper that converts int16 PCM to float32
// and then extracts features.
// Input: pcm is raw int16 samples (little-endian bytes, 2 bytes per sample).
func (e *Extractor) ExtractFromInt16(pcm []byte) [][]float32 {
	n := len(pcm) / 2
	samples := make([]float32, n)
	for i := range n {
		s := int16(pcm[i*2]) | int16(pcm[i*2+1])<<8
		samples[i] = float32(s) / 32768.0
	}
	return e.Extract(samples)
}

// CMVN applies Cepstral Mean and Variance Normalization in-place.
// For each mel dimension, subtracts the mean and divides by the standard
// deviation across all frames. This removes channel and environment effects,
// significantly improving speaker verification accuracy.
func CMVN(features [][]float32) {
	if len(features) == 0 {
		return
	}
	numMels := len(features[0])
	T := float64(len(features))

	for m := range numMels {
		// Compute mean.
		sum := float64(0)
		for _, f := range features {
			sum += float64(f[m])
		}
		mean := sum / T

		// Compute variance.
		varSum := float64(0)
		for _, f := range features {
			d := float64(f[m]) - mean
			varSum += d * d
		}
		std := math.Sqrt(varSum / T)
		if std < 1e-10 {
			std = 1e-10
		}

		// Normalize.
		for _, f := range features {
			f[m] = float32((float64(f[m]) - mean) / std)
		}
	}
}

// Flatten converts [T][numMels] to a flat [T*numMels] slice suitable for
// creating an ncnn Mat2D(numMels, T, data).
func Flatten(features [][]float32) []float32 {
	if len(features) == 0 {
		return nil
	}
	cols := len(features[0])
	flat := make([]float32, len(features)*cols)
	for t, row := range features {
		copy(flat[t*cols:], row)
	}
	return flat
}
