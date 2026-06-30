//go:build gizclaw_e2e

package service_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestServeRejectsDirectStartUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "705-serve-vs-service-managed-workspace")
	help := h.RunCLI("serve", "--help")
	help.MustSucceed(t)
	for _, want := range []string{"Direct foreground server starts are disabled", "--force", "gizclaw service"} {
		if !strings.Contains(help.Stdout, want) {
			t.Fatalf("serve help missing %q:\n%s", want, help.Stdout)
		}
	}

	result := h.RunCLI("serve", h.ServerWorkspace)
	if result.Err == nil {
		t.Fatalf("serve should fail without --force for direct start")
	}
	combined := result.Stderr + result.Stdout
	if !strings.Contains(combined, "direct serve is disabled") || !strings.Contains(combined, "--force") {
		t.Fatalf("unexpected serve error:\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}
