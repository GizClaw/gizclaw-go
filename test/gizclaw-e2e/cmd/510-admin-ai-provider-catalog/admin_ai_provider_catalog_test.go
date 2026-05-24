package adminaiprovidercatalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminAIProviderCatalogUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "510-admin-ai-provider-catalog")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	resourcePath := filepath.Join(h.SandboxDir, "ai-provider-catalog.json")
	if err := os.WriteFile(resourcePath, []byte(`{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "ResourceList",
		"metadata": {"name": "ai-provider-catalog"},
		"spec": {
			"items": [
				{
					"apiVersion": "gizclaw.admin/v1alpha1",
					"kind": "Credential",
					"metadata": {"name": "openai-cli-key"},
					"spec": {
						"provider": "openai",
						"method": "api_key",
						"body": {"api_key": "sk-openai"}
					}
				},
				{
					"apiVersion": "gizclaw.admin/v1alpha1",
					"kind": "Credential",
					"metadata": {"name": "gemini-cli-key"},
					"spec": {
						"provider": "gemini",
						"method": "api_key",
						"body": {"api_key": "sk-gemini"}
					}
				},
				{
					"apiVersion": "gizclaw.admin/v1alpha1",
					"kind": "Credential",
					"metadata": {"name": "dashscope-cli-key"},
					"spec": {
						"provider": "dashscope",
						"method": "api_key",
						"body": {"api_key": "sk-dashscope"}
					}
				},
				{
					"apiVersion": "gizclaw.admin/v1alpha1",
					"kind": "OpenAITenant",
					"metadata": {"name": "openai-cli"},
					"spec": {
						"kind": "compatible",
						"credential_name": "openai-cli-key",
						"base_url": "https://api.openai.example/v1",
						"api_mode": "chat_completions",
						"description": "CLI seeded OpenAI-compatible tenant"
					}
				},
				{
					"apiVersion": "gizclaw.admin/v1alpha1",
					"kind": "GeminiTenant",
					"metadata": {"name": "gemini-cli"},
					"spec": {
						"credential_name": "gemini-cli-key",
						"project_id": "project-cli",
						"location": "global",
						"base_url": "https://gemini.example",
						"description": "CLI seeded Gemini tenant"
					}
				},
				{
					"apiVersion": "gizclaw.admin/v1alpha1",
					"kind": "DashScopeTenant",
					"metadata": {"name": "dashscope-cli"},
					"spec": {
						"credential_name": "dashscope-cli-key",
						"base_url": "https://dashscope.example",
						"description": "CLI seeded DashScope tenant"
					}
				},
				{
					"apiVersion": "gizclaw.admin/v1alpha1",
					"kind": "Model",
					"metadata": {"name": "openai-cli-chat"},
					"spec": {
						"kind": "llm",
						"source": "manual",
						"provider": {
							"kind": "openai-tenant",
							"name": "openai-cli"
						},
						"name": "OpenAI CLI Chat",
						"description": "CLI seeded chat model",
						"provider_data": {
							"openai-tenant": {
								"upstream_model": "gpt-cli",
								"support_json_output": true,
								"support_tool_calls": true
							}
						}
					}
				},
				{
					"apiVersion": "gizclaw.admin/v1alpha1",
					"kind": "ACLView",
					"metadata": {"name": "under-12"},
					"spec": {
						"description": "CLI seeded child-safe view"
					}
				}
			]
		}
	}`), 0o644); err != nil {
		t.Fatalf("write resource list file: %v", err)
	}

	apply := h.RunCLI("admin", "apply", "-f", resourcePath, "--context", "admin-a")
	apply.MustSucceed(t)
	assertOutputContains(t, apply.Stdout, `"kind":"OpenAITenant"`, `"kind":"GeminiTenant"`, `"kind":"DashScopeTenant"`, `"kind":"Model"`, `"kind":"ACLView"`)

	openAIList := h.RunCLI("admin", "openai-tenants", "list", "--context", "admin-a")
	openAIList.MustSucceed(t)
	assertOutputContains(t, openAIList.Stdout, `"name":"openai-cli"`, `"credential_name":"openai-cli-key"`)

	openAIGet := h.RunCLI("admin", "openai-tenants", "get", "openai-cli", "--context", "admin-a")
	openAIGet.MustSucceed(t)
	assertOutputContains(t, openAIGet.Stdout, `"kind":"compatible"`, `"api_mode":"chat_completions"`)

	geminiList := h.RunCLI("admin", "gemini-tenants", "list", "--context", "admin-a")
	geminiList.MustSucceed(t)
	assertOutputContains(t, geminiList.Stdout, `"name":"gemini-cli"`, `"project_id":"project-cli"`)

	geminiGet := h.RunCLI("admin", "gemini-tenants", "get", "gemini-cli", "--context", "admin-a")
	geminiGet.MustSucceed(t)
	assertOutputContains(t, geminiGet.Stdout, `"credential_name":"gemini-cli-key"`, `"location":"global"`)

	dashScopeList := h.RunCLI("admin", "dashscope-tenants", "list", "--context", "admin-a")
	dashScopeList.MustSucceed(t)
	assertOutputContains(t, dashScopeList.Stdout, `"name":"dashscope-cli"`, `"credential_name":"dashscope-cli-key"`)

	dashScopeGet := h.RunCLI("admin", "dashscope-tenants", "get", "dashscope-cli", "--context", "admin-a")
	dashScopeGet.MustSucceed(t)
	assertOutputContains(t, dashScopeGet.Stdout, `"base_url":"https://dashscope.example"`)

	modelsList := h.RunCLI("admin", "models", "list", "--provider-kind", "openai-tenant", "--provider-name", "openai-cli", "--context", "admin-a")
	modelsList.MustSucceed(t)
	assertOutputContains(t, modelsList.Stdout, `"id":"openai-cli-chat"`, `"upstream_model":"gpt-cli"`)

	modelGet := h.RunCLI("admin", "models", "get", "openai-cli-chat", "--context", "admin-a")
	modelGet.MustSucceed(t)
	assertOutputContains(t, modelGet.Stdout, `"kind":"llm"`, `"name":"OpenAI CLI Chat"`)

	viewsList := h.RunCLI("admin", "acl", "views", "list", "--context", "admin-a")
	viewsList.MustSucceed(t)
	assertOutputContains(t, viewsList.Stdout, `"name":"under-12"`)

	viewGet := h.RunCLI("admin", "acl", "views", "get", "under-12", "--context", "admin-a")
	viewGet.MustSucceed(t)
	assertOutputContains(t, viewGet.Stdout, `"description":"CLI seeded child-safe view"`)
}

func assertOutputContains(t *testing.T, output string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(output, value) {
			t.Fatalf("output missing %s:\n%s", value, output)
		}
	}
}
