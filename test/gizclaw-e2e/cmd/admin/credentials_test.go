//go:build gizclaw_e2e

package admin_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminCredentialsUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "504-admin-credentials")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	list := h.RunCLI("admin", "credentials", "list", "--context", "admin-a")
	list.MustSucceed(t)
	for _, want := range []string{`"name":"ui-seed-openai-credential"`, `"name":"ui-seed-credential"`, `"name":"e2e-rpc-credential"`, `"name":"e2e-rpc-credential-049"`} {
		if !strings.Contains(list.Stdout, want) {
			t.Fatalf("credentials list missing %q:\n%s", want, list.Stdout)
		}
	}

	filtered := h.RunCLI("admin", "credentials", "list", "--provider", "openai", "--context", "admin-a")
	filtered.MustSucceed(t)
	if !strings.Contains(filtered.Stdout, `"name":"ui-seed-openai-credential"`) || strings.Contains(filtered.Stdout, `"name":"ui-seed-credential"`) {
		t.Fatalf("credentials filtered list returned unexpected items:\n%s", filtered.Stdout)
	}

	get := h.RunCLI("admin", "credentials", "get", "ui-seed-openai-credential", "--context", "admin-a")
	get.MustSucceed(t)
	if !strings.Contains(get.Stdout, `"provider":"openai"`) {
		t.Fatalf("credentials get missing provider:\n%s", get.Stdout)
	}

	rpcGet := h.RunCLI("admin", "credentials", "get", "e2e-rpc-credential", "--context", "admin-a")
	rpcGet.MustSucceed(t)
	if !strings.Contains(rpcGet.Stdout, `"api_key":"sk-e2e-rpc"`) {
		t.Fatalf("credentials get missing RPC fixture api key:\n%s", rpcGet.Stdout)
	}
}
