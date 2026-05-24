package adminfirmwares_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminFirmwaresUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "511-admin-firmwares")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	firmwarePath := filepath.Join(h.SandboxDir, "firmware.json")
	if err := os.WriteFile(firmwarePath, []byte(`{
		"name": "devkit",
		"description": "Devkit firmware line",
		"slots": {
			"rollback": {"version": "0.9.0"},
			"stable": {"version": "1.0.0", "artifacts": [{"name": "main", "kind": "app", "url": "https://firmware.example/devkit/1.0.0/app.bin"}]},
			"beta": {"version": "1.1.0"},
			"develop": {"version": "1.2.0"},
			"pending": {"version": "1.3.0", "artifacts": [{"name": "assets", "kind": "data", "url": "https://firmware.example/devkit/1.3.0/assets.bin"}]}
		}
	}`), 0o644); err != nil {
		t.Fatalf("write firmware file: %v", err)
	}

	put := h.RunCLI("admin", "firmwares", "put", "devkit", "-f", firmwarePath, "--context", "admin-a")
	put.MustSucceed(t)
	assertContains(t, put.Stdout, `"name":"devkit"`, `"version":"1.0.0"`)

	list := h.RunCLI("admin", "firmwares", "list", "--context", "admin-a")
	list.MustSucceed(t)
	assertContains(t, list.Stdout, `"name":"devkit"`, `"description":"Devkit firmware line"`)

	get := h.RunCLI("admin", "firmwares", "get", "devkit", "--context", "admin-a")
	get.MustSucceed(t)
	assertContains(t, get.Stdout, `"kind":"app"`, `"kind":"data"`)

	release := h.RunCLI("admin", "firmwares", "release", "devkit", "--context", "admin-a")
	release.MustSucceed(t)
	assertContains(t, release.Stdout, `"stable":{"version":"1.1.0"`, `"rollback":{"artifacts":[{"kind":"app"`, `"version":"1.0.0"`)

	rollback := h.RunCLI("admin", "firmwares", "rollback", "devkit", "--context", "admin-a")
	rollback.MustSucceed(t)
	assertContains(t, rollback.Stdout, `"stable":{"artifacts":[{"kind":"app"`, `"version":"1.0.0"`)

	resource := h.RunCLI("admin", "show", "Firmware", "devkit", "--context", "admin-a")
	resource.MustSucceed(t)
	assertContains(t, resource.Stdout, `"kind":"Firmware"`, `"name":"devkit"`)

	delete := h.RunCLI("admin", "firmwares", "delete", "devkit", "--context", "admin-a")
	delete.MustSucceed(t)
	assertContains(t, delete.Stdout, `"name":"devkit"`)
}

func assertContains(t *testing.T, output string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(output, value) {
			t.Fatalf("output missing %s:\n%s", value, output)
		}
	}
}
