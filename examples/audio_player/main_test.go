package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	cfg, err := parseConfig([]string{"-rate", "24k", "-device-id", "3", "/tmp/demo.ogg"})
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.path != "/tmp/demo.ogg" || cfg.rate != "24k" || cfg.deviceID != 3 {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	cfg, err = parseConfig([]string{"-list-devices"})
	if err != nil {
		t.Fatalf("parseConfig list-devices: %v", err)
	}
	if !cfg.listDevices {
		t.Fatalf("expected list-devices config: %+v", cfg)
	}

	if _, err := parseConfig(nil); err == nil {
		t.Fatal("expected missing arg error")
	}
}

func TestConfigValidateErrors(t *testing.T) {
	cases := []config{
		{rate: "11k", path: "/tmp/demo.ogg"},
		{rate: "16k", framesPerBuffer: 1, path: "/tmp/demo.ogg"},
		{rate: "16k"},
	}
	for _, tc := range cases {
		if err := tc.validate(); err == nil {
			t.Fatalf("expected validate error for %+v", tc)
		}
	}
}

func TestPlaybackFormat(t *testing.T) {
	if _, _, err := playbackFormat("16k"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := playbackFormat("48000"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := playbackFormat("bogus"); err == nil {
		t.Fatal("expected unsupported rate error")
	}
}

func TestRunOpenError(t *testing.T) {
	origOpen := openFileFn
	t.Cleanup(func() {
		openFileFn = origOpen
	})

	openFileFn = func(path string) (io.ReadCloser, error) {
		return nil, errors.New("boom")
	}

	var stderr bytes.Buffer
	err := run(config{path: "/tmp/demo.ogg", rate: "16k"}, io.Discard, &stderr)
	if err == nil || !strings.Contains(err.Error(), "open input file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
