//go:build gizclaw_e2e

package root_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestRootHelpUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "root")

	result := h.RunCLI("--help")
	result.MustSucceed(t)
	for _, want := range []string{"serve", "service", "context", "gen-key", "migrate", "connect", "admin", "play"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("root help missing %q:\n%s", want, result.Stdout)
		}
	}
}
