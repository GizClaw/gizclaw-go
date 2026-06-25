//go:build gizclaw_e2e

package admin_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminWorkspacesUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "508-admin-workspaces")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	list := h.RunCLI("admin", "workspaces", "list", "--context", "admin-a")
	list.MustSucceed(t)
	if !strings.Contains(list.Stdout, `"name":"ui-seed-workspace"`) {
		t.Fatalf("workspaces list missing created item:\n%s", list.Stdout)
	}
	for _, want := range []string{`"name":"e2e-rpc-workspace"`, `"name":"e2e-rpc-workspace-119"`} {
		if !strings.Contains(list.Stdout, want) {
			t.Fatalf("workspaces list missing %q:\n%s", want, list.Stdout)
		}
	}

	get := h.RunCLI("admin", "workspaces", "get", "ui-seed-workspace", "--context", "admin-a")
	get.MustSucceed(t)
	if !strings.Contains(get.Stdout, `"workflow_name":"ui-seed-workflow"`) {
		t.Fatalf("workspaces get missing workflow name:\n%s", get.Stdout)
	}

	rpcGet := h.RunCLI("admin", "workspaces", "get", "e2e-rpc-workspace", "--context", "admin-a")
	rpcGet.MustSucceed(t)
	if !strings.Contains(rpcGet.Stdout, `"workflow_name":"e2e-rpc-workflow"`) {
		t.Fatalf("workspaces get missing RPC workflow name:\n%s", rpcGet.Stdout)
	}
}
