//go:build gizclaw_e2e

package genkey_test

import (
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestGenerateKeyUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "gen-key")

	result := h.RunCLI("gen-key")
	result.MustSucceed(t)
	value := strings.TrimSpace(result.Stdout)
	var private giznet.Key
	if err := private.UnmarshalText([]byte(value)); err != nil {
		t.Fatalf("gen-key output is not a GizClaw key: %v, output=%q", err, value)
	}
	if _, err := giznet.NewKeyPair(private); err != nil {
		t.Fatalf("gen-key output cannot derive a key pair: %v", err)
	}

	extra := h.RunCLI("gen-key", "extra")
	if extra.Err == nil {
		t.Fatalf("gen-key with extra args should fail:\nstdout:\n%s", extra.Stdout)
	}
}
