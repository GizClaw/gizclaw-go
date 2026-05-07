package servevsservicemanagedworkspace_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestServeRejectsDirectStartUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "705-serve-vs-service-managed-workspace")
	help := h.RunCLI("serve", "--help")
	help.MustSucceed(t)
	for _, want := range []string{"Direct server starts are disabled", "--force", "gizclaw service"} {
		if !strings.Contains(help.Stdout, want) {
			t.Fatalf("serve help missing %q:\n%s", want, help.Stdout)
		}
	}

	for _, args := range [][]string{
		{"serve", h.ServerWorkspace},
		{"serve", "-f", h.ServerWorkspace},
	} {
		result := h.RunCLI(args...)
		if result.Err == nil {
			t.Fatalf("%q should fail for direct start", strings.Join(args, " "))
		}
		combined := result.Stderr + result.Stdout
		if !strings.Contains(combined, "direct serve is disabled") || !strings.Contains(combined, "gizclaw service start") {
			t.Fatalf("unexpected serve error for %q:\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), result.Stdout, result.Stderr)
		}
	}
}
