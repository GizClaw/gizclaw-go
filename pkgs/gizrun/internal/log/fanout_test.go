package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestFanoutHandlerWritesEnabledHandlers(t *testing.T) {
	var first bytes.Buffer
	var second bytes.Buffer
	handler := NewFanoutHandler(
		slog.NewTextHandler(&first, &slog.HandlerOptions{Level: slog.LevelWarn}),
		slog.NewTextHandler(&second, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)
	logger := slog.New(handler)
	logger.Info("hello", "name", "gizclaw")
	if first.Len() != 0 {
		t.Fatalf("warn handler received info: %s", first.String())
	}
	if got := second.String(); !strings.Contains(got, "hello") || !strings.Contains(got, "name=gizclaw") {
		t.Fatalf("info handler output = %q", got)
	}
}

func TestFanoutHandlerWithAttrsAndGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := NewFanoutHandler(slog.NewTextHandler(&buf, nil)).WithAttrs([]slog.Attr{
		slog.String("service", "openai"),
	}).WithGroup("request")
	if err := handler.Handle(context.Background(), slog.NewRecord(time.Time{}, slog.LevelInfo, "served", 0)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "service=openai") {
		t.Fatalf("output missing attr: %q", out)
	}
}
