package adminworkspaces_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminWorkspacesUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "508-admin-workspaces")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	seedWorkspace(t, h)

	list := h.RunCLI("admin", "workspaces", "list", "--context", "admin-a")
	list.MustSucceed(t)
	if !strings.Contains(list.Stdout, `"name":"workspace-a"`) {
		t.Fatalf("workspaces list missing created item:\n%s", list.Stdout)
	}

	get := h.RunCLI("admin", "workspaces", "get", "workspace-a", "--context", "admin-a")
	get.MustSucceed(t)
	if !strings.Contains(get.Stdout, `"workspace_template_name":"demo-assistant"`) {
		t.Fatalf("workspaces get missing template name:\n%s", get.Stdout)
	}
}

func seedWorkspace(t *testing.T, h *clitest.Harness) {
	t.Helper()

	c := h.ConnectClientFromContext("admin-a")
	defer c.Close()
	api, err := c.ServerAdminClient()
	if err != nil {
		t.Fatalf("create admin client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	templateResp, err := api.CreateWorkspaceTemplateWithResponse(ctx, templateDocument(t, "demo-assistant"))
	if err != nil {
		t.Fatalf("seed workspace template: %v", err)
	}
	if templateResp.JSON200 == nil {
		t.Fatalf("seed workspace template got status %d: %s", templateResp.StatusCode(), strings.TrimSpace(string(templateResp.Body)))
	}
	params := map[string]interface{}{"city": "shanghai"}
	workspaceResp, err := api.CreateWorkspaceWithResponse(ctx, adminservice.WorkspaceUpsert{
		Name:                  "workspace-a",
		WorkspaceTemplateName: "demo-assistant",
		Parameters:            &params,
	})
	if err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	if workspaceResp.JSON200 == nil {
		t.Fatalf("seed workspace got status %d: %s", workspaceResp.StatusCode(), strings.TrimSpace(string(workspaceResp.Body)))
	}
}

func templateDocument(t *testing.T, name string) apitypes.WorkflowTemplateDocument {
	t.Helper()

	var doc apitypes.WorkflowTemplateDocument
	if err := doc.FromSingleAgentGraphWorkflowTemplate(apitypes.SingleAgentGraphWorkflowTemplate{
		ApiVersion: apitypes.WorkflowTemplateAPIVersionGizclawFlowcraftv1alpha1,
		Kind:       apitypes.SingleAgentGraphWorkflowTemplateKindSingleAgentGraphWorkflowTemplate,
		Metadata: apitypes.TemplateMetadata{
			Name: name,
		},
		Spec: apitypes.SingleAgentGraphWorkflowSpec{},
	}); err != nil {
		t.Fatalf("build template document: %v", err)
	}
	return doc
}
