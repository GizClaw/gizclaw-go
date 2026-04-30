package adminresources_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminResourcesUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "509-admin-resources")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	resourcePath := filepath.Join(h.SandboxDir, "credential-resource.json")
	if err := os.WriteFile(resourcePath, []byte(`{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Credential",
		"metadata": {"name": "minimax-main"},
		"spec": {
			"provider": "minimax",
			"method": "api_key",
			"body": {"api_key": "secret"}
		}
	}`), 0o644); err != nil {
		t.Fatalf("write resource file: %v", err)
	}

	apply := h.RunCLI("admin", "apply", "-f", resourcePath, "--context", "admin-a")
	apply.MustSucceed(t)
	if !strings.Contains(apply.Stdout, `"action":"created"`) || !strings.Contains(apply.Stdout, `"name":"minimax-main"`) {
		t.Fatalf("admin apply create output unexpected:\n%s", apply.Stdout)
	}

	missing := h.RunCLI("admin", "show", "Credential", "missing", "--context", "admin-a")
	if missing.Err == nil {
		t.Fatal("admin show missing resource should fail")
	}
	if !strings.Contains(missing.Stderr, "RESOURCE_NOT_FOUND") {
		t.Fatalf("admin show missing stderr = %s", missing.Stderr)
	}

	show := h.RunCLI("admin", "show", "Credential", "minimax-main", "--context", "admin-a")
	show.MustSucceed(t)
	if !strings.Contains(show.Stdout, `"kind":"Credential"`) || !strings.Contains(show.Stdout, `"name":"minimax-main"`) {
		t.Fatalf("admin show output unexpected:\n%s", show.Stdout)
	}

	if err := os.WriteFile(resourcePath, []byte(`{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Credential",
		"metadata": {"name": "minimax-main"},
		"spec": {
			"provider": "minimax",
			"method": "api_key",
			"description": "updated credential",
			"body": {"api_key": "secret"}
		}
	}`), 0o644); err != nil {
		t.Fatalf("write updated resource file: %v", err)
	}
	update := h.RunCLI("admin", "apply", "-f", resourcePath, "--context", "admin-a")
	update.MustSucceed(t)
	if !strings.Contains(update.Stdout, `"action":"updated"`) {
		t.Fatalf("admin apply update output unexpected:\n%s", update.Stdout)
	}

	deleted := h.RunCLI("admin", "delete", "Credential", "minimax-main", "--context", "admin-a")
	deleted.MustSucceed(t)
	if !strings.Contains(deleted.Stdout, `"kind":"Credential"`) || !strings.Contains(deleted.Stdout, `"name":"minimax-main"`) {
		t.Fatalf("admin delete output unexpected:\n%s", deleted.Stdout)
	}

	resourceList := h.RunCLI("admin", "show", "ResourceList", "bundle", "--context", "admin-a")
	if resourceList.Err == nil {
		t.Fatal("admin show ResourceList should fail before server lookup")
	}
	if !strings.Contains(resourceList.Stderr, `resource kind "ResourceList" cannot be addressed by name`) {
		t.Fatalf("admin show ResourceList stderr = %s", resourceList.Stderr)
	}
}
