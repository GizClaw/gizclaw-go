//go:build gizclaw_e2e

package connect_test

import (
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestServerUnavailableUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "702-server-unavailable")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("client-a").MustSucceed(t)
	if _, err := h.RunCLIUntilSuccess("connect", "ping", "--context", "client-a"); err != nil {
		t.Fatal(err)
	}

	h.StopServer()
	result := h.RunCLI("connect", "ping", "--context", "client-a")
	if result.Err == nil {
		t.Fatalf("expected ping to fail while server is unavailable:\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}
