package adminworkspacetemplates_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminWorkspaceTemplatesUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "507-admin-workspace-templates")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	seedWorkspaceTemplates(t, h)

	list := h.RunCLI("admin", "workspace-templates", "list", "--context", "admin-a")
	list.MustSucceed(t)
	for _, want := range []string{`"name":"demo-assistant"`, `"name":"ops-assistant"`} {
		if !strings.Contains(list.Stdout, want) {
			t.Fatalf("workspace templates list missing %q:\n%s", want, list.Stdout)
		}
	}

	get := h.RunCLI("admin", "workspace-templates", "get", "demo-assistant", "--context", "admin-a")
	get.MustSucceed(t)
	if !strings.Contains(get.Stdout, `"kind":"SingleAgentGraphWorkflowTemplate"`) {
		t.Fatalf("workspace templates get missing kind:\n%s", get.Stdout)
	}
}

func seedWorkspaceTemplates(t *testing.T, h *clitest.Harness) {
	t.Helper()

	c := h.ConnectClientFromContext("admin-a")
	defer c.Close()
	api, err := c.ServerAdminClient()
	if err != nil {
		t.Fatalf("create admin client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, doc := range []apitypes.WorkflowTemplateDocument{
		templateDocument(t, "demo-assistant", "primary"),
		templateDocument(t, "ops-assistant", "secondary"),
	} {
		resp, err := api.CreateWorkspaceTemplateWithResponse(ctx, doc)
		if err != nil {
			t.Fatalf("seed workspace template: %v", err)
		}
		if resp.JSON200 == nil {
			t.Fatalf("seed workspace template got status %d: %s", resp.StatusCode(), strings.TrimSpace(string(resp.Body)))
		}
	}
}

func templateDocument(t *testing.T, name, description string) apitypes.WorkflowTemplateDocument {
	t.Helper()

	var doc apitypes.WorkflowTemplateDocument
	if err := doc.FromSingleAgentGraphWorkflowTemplate(apitypes.SingleAgentGraphWorkflowTemplate{
		ApiVersion: apitypes.WorkflowTemplateAPIVersionGizclawFlowcraftv1alpha1,
		Kind:       apitypes.SingleAgentGraphWorkflowTemplateKindSingleAgentGraphWorkflowTemplate,
		Metadata: apitypes.TemplateMetadata{
			Name:        name,
			Description: ptr(description),
		},
		Spec: apitypes.SingleAgentGraphWorkflowSpec{
			"workspace_layout": map[string]interface{}{},
			"runtime":          map[string]interface{}{},
			"agents":           []interface{}{},
			"entry_agent":      "",
		},
	}); err != nil {
		t.Fatalf("build template document: %v", err)
	}
	return doc
}

func ptr(value string) *string {
	return &value
}
