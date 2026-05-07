package invalidworkspaceconfig_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestInvalidWorkspaceConfigUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "703-invalid-workspace-config")
	h.PrepareServerWorkspaceFromFixture("server_config.yaml")

	result := h.RunCLI("serve", h.ServerWorkspace)
	if result.Err == nil {
		t.Fatalf("expected serve to fail for invalid config:\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
	combined := result.Stderr + result.Stdout
	if !strings.Contains(combined, "direct serve is disabled") || !strings.Contains(combined, "gizclaw service start") {
		t.Fatalf("expected service-start guidance, got:\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}
