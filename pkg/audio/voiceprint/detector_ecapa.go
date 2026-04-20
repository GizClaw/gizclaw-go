package voiceprint

import (
	"fmt"

	"github.com/GizClaw/gizclaw-go/pkg/agent/ncnn"
)

type ecapaDetector struct {
	*ncnnDetector
}

func NewECAPA(cfg DetectorConfig) (Detector, error) {
	fbankCfg := fbankConfig{
		SampleRate:   16000,
		NumMels:      80,
		FrameLength:  400,
		FrameShift:   160,
		FFTSize:      400,
		PreEmphasis:  0,
		EnergyFloor:  1e-10,
		LowFreq:      0,
		HighFreq:     8000,
		RemoveDC:     false,
		Center:       true,
		PoveyWindow:  false,
		UseLog10:     true,
		TopDB:        80,
		MeanNorm:     true,
		StdNorm:      false,
		NormalizePCM: true,
	}
	net, err := ncnn.LoadModel(ncnn.ModelSpeakerECAPA)
	if err != nil {
		return nil, fmt.Errorf("voiceprint: load ecapa ncnn model: %w", err)
	}
	model := &ncnnModel{
		net:           net,
		dim:           192,
		fbankCfg:      fbankCfg,
		inputName:     "in0",
		outputName:    "out0",
		segmentFrames: 0,
		hopFrames:     0,
	}
	if err := validateFbankConfig(model.fbankCfg); err != nil {
		panic(fmt.Sprintf("voiceprint: invalid fbank config: %v", err))
	}
	return &ecapaDetector{
		ncnnDetector: newNCNNDetector(model, "ecapa", defaultDetectorVecIDThreshold, cfg),
	}, nil
}
