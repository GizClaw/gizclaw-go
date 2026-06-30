//go:build gizclaw_e2e

package serve_test

import (
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestServerIdentityPersistenceUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "201-server-identity-persistence")
	h.StartServerFromFixture("server_config.yaml")

	firstPubKey := h.ServerPublicKey
	h.StopServer()
	h.RestartServer()

	if h.ServerPublicKey != firstPubKey {
		t.Fatalf("server public key changed after restart: got %q want %q", h.ServerPublicKey, firstPubKey)
	}

	h.CreateContext("client-a").MustSucceed(t)
	if _, err := h.RunCLIUntilSuccess("connect", "ping", "--context", "client-a"); err != nil {
		t.Fatal(err)
	}
}
