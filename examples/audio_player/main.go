package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GizClaw/gizclaw-go/pkg/audio/codec/opus"
	"github.com/GizClaw/gizclaw-go/pkg/audio/codecconv"
	"github.com/GizClaw/gizclaw-go/pkg/audio/pcm"
	"github.com/GizClaw/gizclaw-go/pkg/audio/portaudio"
)

const defaultRate = "16k"

type config struct {
	path            string
	listDevices     bool
	deviceID        int
	framesPerBuffer uint
	rate            string
}

var openFileFn = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse args failed: %v\n", err)
		os.Exit(2)
	}
	if err := run(cfg, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		os.Exit(1)
	}
}

func parseConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("audio_player", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := config{
		deviceID: portaudio.DefaultDeviceID,
		rate:     defaultRate,
	}
	fs.BoolVar(&cfg.listDevices, "list-devices", false, "List available PortAudio output devices")
	fs.IntVar(&cfg.deviceID, "device-id", cfg.deviceID, "Output device ID (-1 uses default)")
	fs.UintVar(&cfg.framesPerBuffer, "frames-per-buffer", 0, "Override playback frames per buffer")
	fs.StringVar(&cfg.rate, "rate", cfg.rate, "Playback sample rate: 16k, 24k, or 48k")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	rest := fs.Args()
	switch {
	case cfg.listDevices && len(rest) == 0:
		return cfg, cfg.validate()
	case len(rest) == 1:
		cfg.path = rest[0]
		return cfg, cfg.validate()
	default:
		return config{}, errors.New("expected a single .ogg path, or use -list-devices")
	}
}

func (cfg config) validate() error {
	if _, _, err := playbackFormat(cfg.rate); err != nil {
		return err
	}
	if cfg.framesPerBuffer > 0 && cfg.framesPerBuffer < 16 {
		return fmt.Errorf("frames-per-buffer must be >= 16, got %d", cfg.framesPerBuffer)
	}
	if cfg.listDevices {
		return nil
	}
	if strings.TrimSpace(cfg.path) == "" {
		return errors.New("path is required")
	}
	return nil
}

func run(cfg config, stdout, stderr io.Writer) error {
	if cfg.listDevices {
		return listDevices(stdout)
	}
	return playFile(cfg, stderr)
}

func listDevices(w io.Writer) error {
	devices, err := portaudio.ListDevices()
	if err != nil {
		return err
	}
	defaultOut, err := portaudio.DefaultOutputDevice()
	if err != nil && !errors.Is(err, portaudio.ErrDeviceNotFound) {
		return err
	}
	defaultID := portaudio.DefaultDeviceID
	if defaultOut != nil {
		defaultID = defaultOut.ID
	}

	for _, device := range devices {
		if device.MaxOutputChannels <= 0 {
			continue
		}
		marker := " "
		if device.ID == defaultID {
			marker = "*"
		}
		if _, err := fmt.Fprintf(
			w,
			"%s id=%d name=%q host=%q out=%d default_rate=%.0f latency=%.2fms\n",
			marker,
			device.ID,
			device.Name,
			device.HostAPI,
			device.MaxOutputChannels,
			device.DefaultSampleRate,
			device.DefaultOutputLatencyMs,
		); err != nil {
			return err
		}
	}
	return nil
}

func playFile(cfg config, stderr io.Writer) error {
	outputRate, format, err := playbackFormat(cfg.rate)
	if err != nil {
		return err
	}

	if !portaudio.NativeRuntimeSupported() {
		return fmt.Errorf("portaudio backend %q is unavailable on this runtime", portaudio.BackendName())
	}

	f, err := openFileFn(cfg.path)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	stream, err := portaudio.OpenPlayback(format, portaudio.PlaybackOptions{
		HasDeviceID:     cfg.deviceID != portaudio.DefaultDeviceID,
		DeviceID:        cfg.deviceID,
		FramesPerBuffer: uint32(cfg.framesPerBuffer),
	})
	if err != nil {
		return fmt.Errorf("open playback stream: %w", err)
	}
	defer func() {
		_ = stream.Close()
	}()

	deviceLabel := "default"
	if cfg.deviceID != portaudio.DefaultDeviceID {
		deviceLabel = fmt.Sprintf("%d", cfg.deviceID)
	}
	_, _ = fmt.Fprintf(stderr, "backend=%s device=%s format=%s path=%s\n", portaudio.BackendName(), deviceLabel, format, cfg.path)

	if _, err := codecconv.OggToPCM(stream, f, outputRate); err != nil {
		return fmt.Errorf("decode and play ogg: %w", err)
	}
	return nil
}

func playbackFormat(rate string) (opus.OpusSampleRate, pcm.Format, error) {
	switch strings.ToLower(strings.TrimSpace(rate)) {
	case "16k", "16000":
		return opus.SampleRate16K, pcm.L16Mono16K, nil
	case "24k", "24000":
		return opus.SampleRate24K, pcm.L16Mono24K, nil
	case "48k", "48000":
		return opus.SampleRate48K, pcm.L16Mono48K, nil
	default:
		return 0, 0, fmt.Errorf("unsupported rate %q (use 16k, 24k, or 48k)", rate)
	}
}
