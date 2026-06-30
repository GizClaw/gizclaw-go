//go:build gizclaw_e2e

package context_test

import (
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestCurrentContextDefaultUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "101-context-current-default")

	h.CreateContext("valid").MustSucceed(t)
	h.WaitForPing("valid")

	wrongKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate wrong server key: %v", err)
	}
	h.CreateContextWith("invalid", h.ServerAddr, wrongKey.Private.String()).MustSucceed(t)

	h.UseContext("valid").MustSucceed(t)
	validPing, err := h.RunCLIUntilSuccess("connect", "ping")
	if err != nil {
		t.Fatal(err)
	}
	assertDefaultPingOutput(t, validPing.Stdout)

	h.UseContext("invalid").MustSucceed(t)
	invalidPing := h.RunCLI("connect", "ping")
	if invalidPing.Err == nil {
		t.Fatalf("ping without --context should fail for invalid current context:\nstdout:\n%s\nstderr:\n%s", invalidPing.Stdout, invalidPing.Stderr)
	}
	if !strings.Contains(invalidPing.Stderr, "Error:") {
		t.Fatalf("expected user-facing error output:\n%s", invalidPing.Stderr)
	}

	h.UseContext("valid").MustSucceed(t)
	validPingAgain, err := h.RunCLIUntilSuccess("connect", "ping")
	if err != nil {
		t.Fatal(err)
	}
	assertDefaultPingOutput(t, validPingAgain.Stdout)
}

func assertDefaultPingOutput(t *testing.T, stdout string) {
	t.Helper()

	for _, fragment := range []string{"Server Time:", "RTT:", "Clock Diff:"} {
		if !strings.Contains(stdout, fragment) {
			t.Fatalf("ping output missing %q:\n%s", fragment, stdout)
		}
	}
}
