//go:build gizclaw_e2e

package connect_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
)

func TestMultiClientSequentialIsolationUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "401-multi-client-sequential-isolation")

	h.CreateContext("alpha").MustSucceed(t)
	h.CreateContext("beta").MustSucceed(t)
	h.UseContext("alpha").MustSucceed(t)

	if _, err := h.RunCLIUntilSuccess("connect", "ping"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.RunCLIUntilSuccess("connect", "ping", "--context", "beta"); err != nil {
		t.Fatal(err)
	}

	list := h.ListContexts()
	list.MustSucceed(t)
	if !strings.Contains(list.Stdout, "* alpha") {
		t.Fatalf("expected alpha to remain current after explicit beta command:\n%s", list.Stdout)
	}
	if _, err := h.RunCLIUntilSuccess("connect", "ping"); err != nil {
		t.Fatal(err)
	}
}
