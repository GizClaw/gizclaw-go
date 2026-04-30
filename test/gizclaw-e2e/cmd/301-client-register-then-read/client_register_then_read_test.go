package clientregisterthenread_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestClientRegisterThenReadUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "301-client-register-then-read")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	h.CreateContext("device-a").MustSucceed(t)
	h.RegisterContext(
		"device-a",
		"--name", "device-a",
		"--sn", "device-a-sn",
		"--manufacturer", "Acme",
		"--model", "Model-A",
		"--depot", "demo",
		"--firmware-semver", "1.0.0",
	).MustSucceed(t)

	devicePubKey := h.ContextPublicKey("device-a")

	info := h.RunCLI("admin", "gears", "info", devicePubKey, "--context", "admin-a")
	info.MustSucceed(t)
	for _, fragment := range []string{`"sn":"device-a-sn"`, `"manufacturer":"Acme"`, `"model":"Model-A"`} {
		if !strings.Contains(info.Stdout, fragment) {
			t.Fatalf("admin info output missing %q:\n%s", fragment, info.Stdout)
		}
	}
}
