//go:build gizclaw_e2e

package admin_test

import (
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminAIProviderCatalogUserStory(t *testing.T) {
	h := clitest.NewSetupHarness(t, "510-admin-ai-provider-catalog")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	openAIList := h.RunCLI("admin", "openai-tenants", "list", "--context", "admin-a")
	openAIList.MustSucceed(t)
	assertOutputContains(t, openAIList.Stdout, `"name":"ui-seed-openai-tenant"`, `"credential_name":"ui-seed-openai-credential"`)

	openAIGet := h.RunCLI("admin", "openai-tenants", "get", "ui-seed-openai-tenant", "--context", "admin-a")
	openAIGet.MustSucceed(t)
	assertOutputContains(t, openAIGet.Stdout, `"kind":"compatible"`, `"api_mode":"chat_completions"`)

	geminiList := h.RunCLI("admin", "gemini-tenants", "list", "--context", "admin-a")
	geminiList.MustSucceed(t)
	assertOutputContains(t, geminiList.Stdout, `"name":"ui-seed-gemini-tenant"`, `"project_id":"ui-seed-project"`)

	geminiGet := h.RunCLI("admin", "gemini-tenants", "get", "ui-seed-gemini-tenant", "--context", "admin-a")
	geminiGet.MustSucceed(t)
	assertOutputContains(t, geminiGet.Stdout, `"credential_name":"ui-seed-gemini-credential"`, `"location":"global"`)

	dashScopeList := h.RunCLI("admin", "dashscope-tenants", "list", "--context", "admin-a")
	dashScopeList.MustSucceed(t)
	assertOutputContains(t, dashScopeList.Stdout, `"name":"ui-seed-dashscope-tenant"`, `"credential_name":"ui-seed-dashscope-credential"`)

	dashScopeGet := h.RunCLI("admin", "dashscope-tenants", "get", "ui-seed-dashscope-tenant", "--context", "admin-a")
	dashScopeGet.MustSucceed(t)
	assertOutputContains(t, dashScopeGet.Stdout, `"base_url":"https://dashscope.example.invalid/compatible-mode/v1"`)

	modelsList := h.RunCLI("admin", "models", "list", "--provider-kind", "openai-tenant", "--provider-name", "ui-seed-openai-tenant", "--context", "admin-a")
	modelsList.MustSucceed(t)
	assertOutputContains(t, modelsList.Stdout, `"id":"ui-seed-openai-chat"`, `"upstream_model":"gpt-4o-mini"`)

	rpcModelsList := h.RunCLI("admin", "models", "list", "--provider-kind", "openai-tenant", "--provider-name", "e2e-rpc-provider", "--context", "admin-a")
	rpcModelsList.MustSucceed(t)
	assertOutputContains(t, rpcModelsList.Stdout, `"id":"e2e-rpc-model"`, `"id":"e2e-rpc-model-079"`)

	modelGet := h.RunCLI("admin", "models", "get", "ui-seed-openai-chat", "--context", "admin-a")
	modelGet.MustSucceed(t)
	assertOutputContains(t, modelGet.Stdout, `"kind":"llm"`, `"name":"Seeded OpenAI Chat"`)

	rpcModelGet := h.RunCLI("admin", "models", "get", "e2e-rpc-model", "--context", "admin-a")
	rpcModelGet.MustSucceed(t)
	assertOutputContains(t, rpcModelGet.Stdout, `"upstream_model":"e2e-rpc-core-model"`)

	viewsList := h.RunCLI("admin", "acl", "views", "list", "--context", "admin-a")
	viewsList.MustSucceed(t)
	assertOutputContains(t, viewsList.Stdout, `"name":"under-12"`)

	viewGet := h.RunCLI("admin", "acl", "views", "get", "under-12", "--context", "admin-a")
	viewGet.MustSucceed(t)
	assertOutputContains(t, viewGet.Stdout, `"description":"Seeded child-safe content view for UI tests"`)
}

func assertOutputContains(t *testing.T, output string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(output, value) {
			t.Fatalf("output missing %s:\n%s", value, output)
		}
	}
}
