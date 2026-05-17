package gizrun

import (
	"context"
	"testing"
)

func TestTagHTTP(t *testing.T) {
	ctx := TagHTTP(context.Background(), HTTPMethod, "POST", HTTPRoute, "/v1/chat")
	ctx = TagHTTP(ctx, HTTPStatusCode, "200")

	ns, ok := HTTPLabels(ctx)
	if !ok {
		t.Fatal("HTTPLabels ok = false, want true")
	}
	for key, want := range map[string]string{
		HTTPMethod:     "POST",
		HTTPRoute:      "/v1/chat",
		HTTPStatusCode: "200",
	} {
		if got, ok := ns.Value(key); !ok || got != want {
			t.Fatalf("HTTP namespace value %q = (%q, %v), want (%q, true)", key, got, ok, want)
		}
	}
}

func TestTagGenx(t *testing.T) {
	ctx := TagGenx(context.Background(), GenxProvider, "openai", GenxModel, "gpt-test")
	ctx = TagGenx(ctx, GenxTokenType, TokenPrompt)

	ns, ok := GenxLabels(ctx)
	if !ok {
		t.Fatal("GenxLabels ok = false, want true")
	}
	for key, want := range map[string]string{
		GenxProvider:  "openai",
		GenxModel:     "gpt-test",
		GenxTokenType: TokenPrompt,
	} {
		if got, ok := ns.Value(key); !ok || got != want {
			t.Fatalf("Genx namespace value %q = (%q, %v), want (%q, true)", key, got, ok, want)
		}
	}
}

func TestTagLogSink(t *testing.T) {
	ctx := TagLogSink(context.Background(), "status", "ok")

	ns, ok := LogSinkLabels(ctx)
	if !ok {
		t.Fatal("LogSinkLabels ok = false, want true")
	}
	if got, ok := ns.Value("status"); !ok || got != "ok" {
		t.Fatalf("logsink namespace status = (%q, %v), want (%q, true)", got, ok, "ok")
	}
}

func TestTag(t *testing.T) {
	ctx := Tag(context.Background(), "custom", "key", "value")

	ns, ok := Labels(ctx, "custom")
	if !ok {
		t.Fatal("Labels(custom) ok = false, want true")
	}
	if got, ok := ns.Value("key"); !ok || got != "value" {
		t.Fatalf("custom namespace key = (%q, %v), want (%q, true)", got, ok, "value")
	}
}
