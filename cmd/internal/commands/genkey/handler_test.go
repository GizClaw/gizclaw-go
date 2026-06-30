package genkey

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/gizrun"
)

func TestHandlerExecute(t *testing.T) {
	var out bytes.Buffer
	handler := Handler{Out: &out}

	if err := handler.Execute(context.Background(), gizrun.CommandLine{Args: []string{"gen-key"}}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	value := strings.TrimSpace(out.String())
	var key giznet.Key
	if err := key.UnmarshalText([]byte(value)); err != nil {
		t.Fatalf("handler output is not a GizClaw key: %v, output=%q", err, value)
	}
	if _, err := giznet.NewKeyPair(key); err != nil {
		t.Fatalf("handler output cannot derive a key pair: %v", err)
	}
}

func TestHandlerDefaultOut(t *testing.T) {
	if got := (Handler{}).out(); got != os.Stdout {
		t.Fatalf("default out = %v, want stdout", got)
	}
}

func TestNewCmdExecute(t *testing.T) {
	var out bytes.Buffer
	cmd := NewCmd()
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	assertKeyOutput(t, out.String())
}

func TestNewCmdRejectsArgs(t *testing.T) {
	cmd := NewCmd()
	cmd.SetArgs([]string{"extra"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("Execute succeeded, want argument error")
	}
}

func assertKeyOutput(t *testing.T, output string) {
	t.Helper()
	value := strings.TrimSpace(output)
	var key giznet.Key
	if err := key.UnmarshalText([]byte(value)); err != nil {
		t.Fatalf("output is not a GizClaw key: %v, output=%q", err, value)
	}
	if _, err := giznet.NewKeyPair(key); err != nil {
		t.Fatalf("output cannot derive a key pair: %v", err)
	}
}
