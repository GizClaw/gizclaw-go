//go:build gizclaw_e2e

package connect_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestRepeatCommandAfterPartialStateUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "603-repeat-command-after-partial-state")

	h.CreateContext("device-a").MustSucceed(t)
	first := h.RegisterContext("device-a", "--sn", "repeat-partial-device-a")
	first.MustSucceed(t)

	second := h.RegisterContext("device-a", "--sn", "repeat-partial-device-a-retry")
	second.MustSucceed(t)
	if !strings.Contains(second.Stdout, `"sn":"repeat-partial-device-a-retry"`) {
		t.Fatalf("second register should update auto-registered device info:\n%s", second.Stdout)
	}

	if _, err := h.RunCLIUntilSuccess("connect", "ping", "--context", "device-a"); err != nil {
		t.Fatal(err)
	}
}
