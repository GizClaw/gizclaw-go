package voiceprint

import (
	"fmt"

	"github.com/GizClaw/gizclaw-go/pkg/agent/ncnn"
)

type eres2netDetector struct {
	*ncnnDetector
}

func eres2netFbankConfig() fbankConfig {
	return fbankConfig{
		SampleRate:   16000,
		NumMels:      80,
		FrameLength:  400,
		FrameShift:   160,
		FFTSize:      0,
		PreEmphasis:  0.97,
		EnergyFloor:  1e-10,
		LowFreq:      20,
		HighFreq:     -400,
		RemoveDC:     true,
		Center:       false,
		PoveyWindow:  true,
		UseLog10:     false,
		TopDB:        0,
		MeanNorm:     true,
		StdNorm:      true,
		NormalizePCM: true,
	}
}

func NewERes2Net(cfg DetectorConfig) (Detector, error) {
	fbankCfg := eres2netFbankConfig()
	net, err := ncnn.LoadModel(ncnn.ModelSpeakerERes2Net)
	if err != nil {
		return nil, fmt.Errorf("voiceprint: load eres2net ncnn model: %w", err)
	}
	model := &ncnnModel{
		net:           net,
		dim:           512,
		fbankCfg:      fbankCfg,
		inputName:     "in0",
		outputName:    "out0",
		segmentFrames: defaultSegmentFrames,
		hopFrames:     defaultHopFrames,
	}
	if err := validateFbankConfig(model.fbankCfg); err != nil {
		panic(fmt.Sprintf("voiceprint: invalid fbank config: %v", err))
	}
	return &eres2netDetector{
		ncnnDetector: newNCNNDetector(model, "eres2net", defaultDetectorVecIDThreshold, cfg),
	}, nil
}
