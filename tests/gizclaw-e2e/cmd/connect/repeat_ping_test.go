//go:build gizclaw_e2e

package connect_test

import (
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestRepeatPingUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "600-repeat-ping")

	h.CreateContext("client-a").MustSucceed(t)
	for range 10 {
		if _, err := h.RunCLIUntilSuccess("connect", "ping", "--context", "client-a"); err != nil {
			t.Fatal(err)
		}
	}
}
