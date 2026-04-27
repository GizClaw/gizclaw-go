package adminconfigorfirmwareflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/integration/cmd"
)

func TestAdminConfigOrFirmwareFlowUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "503-admin-config-or-firmware-flow")
	h.StartServerFromFixture("server_config.yaml")

	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "admin_default", "--sn", "admin-sn").MustSucceed(t)
	h.CreateContext("device-a").MustSucceed(t)
	h.RegisterContext("device-a", "admin_default", "--sn", "device-sn", "--depot", "demo", "--firmware-semver", "1.0.0").MustSucceed(t)
	devicePubKey := h.ContextPublicKey("device-a")

	listBefore := h.RunCLI("admin", "firmware", "list", "--context", "admin-a")
	listBefore.MustSucceed(t)

	infoPath := filepath.Join(h.SandboxDir, "depot-info.json")
	if err := os.WriteFile(infoPath, []byte(`{"files":[{"path":"image.bin"}]}`), 0o644); err != nil {
		t.Fatalf("write depot info: %v", err)
	}

	put := h.RunCLI("admin", "firmware", "put-info", "demo", "--context", "admin-a", "--file", infoPath)
	put.MustSucceed(t)
	if !strings.Contains(put.Stdout, `"name":"demo"`) {
		t.Fatalf("expected put-info output to include depot name:\n%s", put.Stdout)
	}

	get := h.RunCLI("admin", "firmware", "get", "demo", "--context", "admin-a")
	get.MustSucceed(t)
	for _, fragment := range []string{`"name":"demo"`, `"path":"image.bin"`} {
		if !strings.Contains(get.Stdout, fragment) {
			t.Fatalf("expected firmware get output to include %q:\n%s", fragment, get.Stdout)
		}
	}

	listAfter := h.RunCLI("admin", "firmware", "list", "--context", "admin-a")
	listAfter.MustSucceed(t)
	if !strings.Contains(listAfter.Stdout, `"name":"demo"`) {
		t.Fatalf("expected firmware list output to include depot:\n%s", listAfter.Stdout)
	}

	setChannel := h.RunCLI("admin", "gears", "set-firmware-channel", devicePubKey, "stable", "--context", "admin-a")
	setChannel.MustSucceed(t)
	if !strings.Contains(setChannel.Stdout, `"channel":"stable"`) {
		t.Fatalf("expected set-firmware-channel output to include stable channel:\n%s", setChannel.Stdout)
	}

	configPath := filepath.Join(h.SandboxDir, "gear-config.json")
	if err := os.WriteFile(configPath, []byte(`{"firmware":{"channel":"beta"}}`), 0o644); err != nil {
		t.Fatalf("write gear config: %v", err)
	}
	putConfig := h.RunCLI("admin", "gears", "put-config", devicePubKey, "--file", configPath, "--context", "admin-a")
	putConfig.MustSucceed(t)
	if !strings.Contains(putConfig.Stdout, `"channel":"beta"`) {
		t.Fatalf("expected put-config output to include beta channel:\n%s", putConfig.Stdout)
	}
}
