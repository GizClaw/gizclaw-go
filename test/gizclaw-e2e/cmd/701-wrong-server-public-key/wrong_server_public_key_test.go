package wrongserverpublickey_test

import (
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestWrongServerPublicKeyUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "701-wrong-server-public-key")
	h.StartServerFromFixture("server_config.yaml")

	var wrongKey giznet.PublicKey
	if err := wrongKey.UnmarshalText([]byte(h.ServerPublicKey)); err != nil {
		t.Fatalf("parse server public key: %v", err)
	}
	wrongKey[0] ^= 0xff
	h.CreateContextWith("broken", h.ServerAddr, wrongKey.String()).MustSucceed(t)

	result := h.RunCLI("ping", "--context", "broken")
	if result.Err == nil {
		t.Fatalf("expected ping to fail with wrong server public key:\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}
