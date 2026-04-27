package servevsservicemanagedworkspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestServeRejectsServiceManagedWorkspaceUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "705-serve-vs-service-managed-workspace")
	help := h.RunCLI("serve", "--help")
	help.MustSucceed(t)
	for _, want := range []string{"foreground", "--force", "service-managed"} {
		if !strings.Contains(help.Stdout, want) {
			t.Fatalf("serve help missing %q:\n%s", want, help.Stdout)
		}
	}

	if err := os.WriteFile(filepath.Join(h.ServerWorkspace, "service.json"), []byte(`{
		"managed": true,
		"service_name": "com.gizclaw.serve",
		"service_type": "system-service",
		"workspace": "`+h.ServerWorkspace+`"
	}`), 0o644); err != nil {
		t.Fatalf("write service marker: %v", err)
	}

	for _, args := range [][]string{
		{"serve", h.ServerWorkspace},
		{"serve", "-f", h.ServerWorkspace},
	} {
		result := h.RunCLI(args...)
		if result.Err == nil {
			t.Fatalf("%q should fail for service-managed workspace", strings.Join(args, " "))
		}
		if !strings.Contains(result.Stderr+result.Stdout, "managed by gizclaw service") {
			t.Fatalf("unexpected serve error for %q:\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), result.Stdout, result.Stderr)
		}
	}
}
