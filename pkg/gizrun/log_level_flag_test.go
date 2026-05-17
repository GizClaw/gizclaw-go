package gizrun

import (
	"log/slog"
	"testing"
)

func TestLogLevelFlag(t *testing.T) {
	var level slog.LevelVar
	flag := logLevelFlag{target: &level}

	if got := flag.String(); got != slog.LevelInfo.String() {
		t.Fatalf("String() = %q, want %q", got, slog.LevelInfo.String())
	}
	if err := flag.Set("DEBUG"); err != nil {
		t.Fatal(err)
	}
	if got := level.Level(); got != slog.LevelDebug {
		t.Fatalf("level = %v, want %v", got, slog.LevelDebug)
	}
	if got := flag.String(); got != slog.LevelDebug.String() {
		t.Fatalf("String() = %q, want %q", got, slog.LevelDebug.String())
	}
	if err := flag.Set("invalid"); err == nil {
		t.Fatal("Set(invalid) err = nil, want error")
	}
}

func TestLogLevelFlagNilTarget(t *testing.T) {
	if got := (logLevelFlag{}).String(); got != slog.LevelInfo.String() {
		t.Fatalf("String() = %q, want %q", got, slog.LevelInfo.String())
	}
}
