//go:build gizclaw_e2e

package connect_test

import (
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestWrongServerPublicKeyUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "701-wrong-server-public-key")

	wrongKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate wrong server key: %v", err)
	}
	h.CreateContextWith("broken", h.ServerAddr, wrongKey.Private.String()).MustSucceed(t)

	result := h.RunCLI("connect", "ping", "--context", "broken")
	if result.Err == nil {
		t.Fatalf("expected ping to fail with wrong server public key:\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}
