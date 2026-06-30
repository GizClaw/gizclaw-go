//go:build gizclaw_e2e

package root_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestLegacySingleDashLongFlagsUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "root")

	result := h.RunCLI("admin", "-listen=127.0.0.1:0", "--help")
	result.MustSucceed(t)
	if !strings.Contains(result.Stdout, "--listen") {
		t.Fatalf("admin help missing normalized --listen flag:\n%s", result.Stdout)
	}
}
