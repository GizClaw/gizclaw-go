package adminlistgears_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminListGearsUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "501-admin-list-gears")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("admin-a").MustSucceed(t)
	h.CreateContext("device-a").MustSucceed(t)
	h.CreateContext("device-b").MustSucceed(t)

	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)
	h.RegisterContext("device-a", "--sn", "device-a-sn").MustSucceed(t)
	h.RegisterContext("device-b", "--sn", "device-b-sn").MustSucceed(t)

	list := h.RunCLI("admin", "gears", "list", "--context", "admin-a")
	list.MustSucceed(t)

	for _, publicKey := range []string{
		h.ContextPublicKey("admin-a"),
		h.ContextPublicKey("device-a"),
		h.ContextPublicKey("device-b"),
	} {
		if !strings.Contains(list.Stdout, publicKey) {
			t.Fatalf("expected admin gear list to include %q:\n%s", publicKey, list.Stdout)
		}
	}
}
