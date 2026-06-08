package clientpublicreadsequence_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestSetNameUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "300-client-public-read-sequence")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("device-a").MustSucceed(t)
	h.RegisterContext("device-a", "--name", "before-name", "--sn", "device-a-sn").MustSucceed(t)

	setName := h.RunCLI("peer", "set-name", "after-name", "--context", "device-a")
	setName.MustSucceed(t)
	if !strings.Contains(setName.Stdout, `"name":"after-name"`) {
		t.Fatalf("set-name missing updated name:\n%s", setName.Stdout)
	}
	if !strings.Contains(setName.Stdout, `"sn":"device-a-sn"`) {
		t.Fatalf("set-name should preserve existing fields:\n%s", setName.Stdout)
	}

	h.CreateContext("device-b").MustSucceed(t)
	autoRegistered := h.RunCLI("peer", "set-name", "auto-name", "--context", "device-b")
	autoRegistered.MustSucceed(t)
	if !strings.Contains(autoRegistered.Stdout, `"name":"auto-name"`) {
		t.Fatalf("set-name should register missing gear with name:\n%s", autoRegistered.Stdout)
	}
}
