package adminworkflows_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminWorkflowsUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "507-admin-workflows")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "--sn", "admin-sn").MustSucceed(t)

	seedWorkflows(t, h)

	list := h.RunCLI("admin", "workflows", "list", "--context", "admin-a")
	list.MustSucceed(t)
	for _, want := range []string{`"name":"demo-assistant"`, `"name":"ops-assistant"`} {
		if !strings.Contains(list.Stdout, want) {
			t.Fatalf("workflows list missing %q:\n%s", want, list.Stdout)
		}
	}

	get := h.RunCLI("admin", "workflows", "get", "demo-assistant", "--context", "admin-a")
	get.MustSucceed(t)
	if !strings.Contains(get.Stdout, `"driver":"flowcraft"`) {
		t.Fatalf("workflows get missing driver:\n%s", get.Stdout)
	}
}

func seedWorkflows(t *testing.T, h *clitest.Harness) {
	t.Helper()

	c := h.ConnectClientFromContext("admin-a")
	defer c.Close()
	api, err := c.ServerAdminClient()
	if err != nil {
		t.Fatalf("create admin client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, doc := range []apitypes.WorkflowDocument{
		workflowDocument(t, "demo-assistant", "primary"),
		workflowDocument(t, "ops-assistant", "secondary"),
	} {
		resp, err := api.CreateWorkflowWithResponse(ctx, doc)
		if err != nil {
			t.Fatalf("seed workflow: %v", err)
		}
		if resp.JSON200 == nil {
			t.Fatalf("seed workflow got status %d: %s", resp.StatusCode(), strings.TrimSpace(string(resp.Body)))
		}
	}
}

func workflowDocument(t *testing.T, name, description string) apitypes.WorkflowDocument {
	t.Helper()

	spec := apitypes.FlowcraftWorkflowSpec{
		"workspace_layout": map[string]interface{}{},
		"runtime":          map[string]interface{}{},
		"agents":           []interface{}{},
		"entry_agent":      "",
	}
	return apitypes.WorkflowDocument{
		Metadata: apitypes.WorkflowMetadata{
			Name:        name,
			Description: ptr(description),
		},
		Spec: apitypes.WorkflowSpec{
			Driver:    apitypes.WorkflowDriverFlowcraft,
			Flowcraft: &spec,
		},
	}
}

func ptr(value string) *string {
	return &value
}
