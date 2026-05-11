package resourcemanager

import (
	"context"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func TestGetRejectsUnknownKind(t *testing.T) {
	manager := New(Services{})

	_, err := manager.Get(context.Background(), apitypes.ResourceKind("Unknown"), "example")
	assertResourceError(t, err, 400, "UNKNOWN_RESOURCE_KIND")
}

func TestGetRejectsEmptyName(t *testing.T) {
	manager := New(Services{})

	_, err := manager.Get(context.Background(), apitypes.ResourceKindCredential, "")
	assertResourceError(t, err, 400, "INVALID_RESOURCE")
}

func TestGetRejectsResourceList(t *testing.T) {
	manager := New(Services{})

	_, err := manager.Get(context.Background(), apitypes.ResourceKindResourceList, "bundle")
	assertResourceError(t, err, 400, "UNSUPPORTED_RESOURCE_GET")
}

func TestGetRejectsMissingService(t *testing.T) {
	manager := New(Services{})

	_, err := manager.Get(context.Background(), apitypes.ResourceKindCredential, "example")
	assertResourceError(t, err, 500, "RESOURCE_SERVICE_NOT_CONFIGURED")
}

func TestGetReturnsNotFoundByKind(t *testing.T) {
	tests := []struct {
		name     string
		kind     apitypes.ResourceKind
		manager  *Manager
		wantCode string
	}{
		{name: "credential", kind: apitypes.ResourceKindCredential, manager: New(Services{Credentials: newFakeCredentials()}), wantCode: "RESOURCE_NOT_FOUND"},
		{name: "peer config", kind: apitypes.ResourceKindPeerConfig, manager: New(Services{Peers: newFakePeers()}), wantCode: "GEAR_NOT_FOUND"},
		{name: "model", kind: apitypes.ResourceKindModel, manager: newModelManager(), wantCode: "RESOURCE_NOT_FOUND"},
		{name: "minimax tenant", kind: apitypes.ResourceKindMiniMaxTenant, manager: New(Services{MiniMax: newFakeMiniMax()}), wantCode: "RESOURCE_NOT_FOUND"},
		{name: "voice", kind: apitypes.ResourceKindVoice, manager: New(Services{MiniMax: newFakeMiniMax()}), wantCode: "RESOURCE_NOT_FOUND"},
		{name: "workspace", kind: apitypes.ResourceKindWorkspace, manager: New(Services{Workspaces: newFakeWorkspaces()}), wantCode: "RESOURCE_NOT_FOUND"},
		{name: "workflow", kind: apitypes.ResourceKindWorkflow, manager: New(Services{Workflows: newFakeWorkflows()}), wantCode: "RESOURCE_NOT_FOUND"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.manager.Get(context.Background(), tc.kind, "missing")
			assertResourceError(t, err, 404, tc.wantCode)
		})
	}
}

func TestPutRejectsUnknownKind(t *testing.T) {
	manager := New(Services{})

	_, err := manager.Put(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Unknown",
		"metadata": {"name": "example"},
		"spec": {}
	}`))
	assertResourceError(t, err, 400, "UNKNOWN_RESOURCE_KIND")
}

func TestPutRejectsNilManager(t *testing.T) {
	var manager *Manager

	_, err := manager.Put(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Credential",
		"metadata": {"name": "example"},
		"spec": {
			"provider": "minimax",
			"method": "api_key",
			"body": {"api_key": "secret"}
		}
	}`))
	assertResourceError(t, err, 500, "RESOURCE_MANAGER_NOT_CONFIGURED")
}

func TestPutRejectsMissingServicesByKind(t *testing.T) {
	manager := New(Services{})
	tests := []struct {
		name     string
		resource string
	}{
		{name: "credential", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Credential","metadata":{"name":"name"},"spec":{"provider":"minimax","method":"api_key","body":{"api_key":"secret"}}}`},
		{name: "peer config", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"PeerConfig","metadata":{"name":"gear"},"spec":{}}`},
		{name: "model", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Model","metadata":{"name":"model"},"spec":{"kind":"llm","provider":{"kind":"openai-compatible","name":"main"},"source":"manual"}}`},
		{name: "minimax tenant", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`},
		{name: "voice", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`},
		{name: "workspace", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workflow_name":"workflow"}}`},
		{name: "workflow", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workflow","metadata":{"name":"workflow"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"FlowcraftWorkflow","metadata":{"name":"workflow"},"spec":{}}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.Put(context.Background(), mustResource(t, tc.resource))
			assertResourceError(t, err, 500, "RESOURCE_SERVICE_NOT_CONFIGURED")
		})
	}
}

func TestPutRejectsUnsupportedVersionByKind(t *testing.T) {
	manager := New(Services{
		Credentials: newFakeCredentials(),
		Peers:       newFakePeers(),
		Models:      newModelManager().services.Models,
		MiniMax:     newFakeMiniMax(),
		Workspaces:  newFakeWorkspaces(),
		Workflows:   newFakeWorkflows(),
	})
	tests := []struct {
		name     string
		resource string
	}{
		{name: "credential", resource: `{"apiVersion":"unsupported","kind":"Credential","metadata":{"name":"name"},"spec":{"provider":"minimax","method":"api_key","body":{"api_key":"secret"}}}`},
		{name: "peer config", resource: `{"apiVersion":"unsupported","kind":"PeerConfig","metadata":{"name":"gear"},"spec":{}}`},
		{name: "model", resource: `{"apiVersion":"unsupported","kind":"Model","metadata":{"name":"model"},"spec":{"kind":"llm","provider":{"kind":"openai-compatible","name":"main"},"source":"manual"}}`},
		{name: "minimax tenant", resource: `{"apiVersion":"unsupported","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`},
		{name: "resource list", resource: `{"apiVersion":"unsupported","kind":"ResourceList","metadata":{"name":"bundle"},"spec":{"items":[]}}`},
		{name: "voice", resource: `{"apiVersion":"unsupported","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`},
		{name: "workspace", resource: `{"apiVersion":"unsupported","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workflow_name":"workflow"}}`},
		{name: "workflow", resource: `{"apiVersion":"unsupported","kind":"Workflow","metadata":{"name":"workflow"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"FlowcraftWorkflow","metadata":{"name":"workflow"},"spec":{}}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.Put(context.Background(), mustResource(t, tc.resource))
			assertResourceError(t, err, 400, "UNSUPPORTED_RESOURCE_VERSION")
		})
	}
}

func TestDeleteRejectsUnsupportedInputs(t *testing.T) {
	var nilManager *Manager
	_, err := nilManager.Delete(context.Background(), apitypes.ResourceKindCredential, "example")
	assertResourceError(t, err, 500, "RESOURCE_MANAGER_NOT_CONFIGURED")

	manager := New(Services{})
	_, err = manager.Delete(context.Background(), apitypes.ResourceKindCredential, "")
	assertResourceError(t, err, 400, "INVALID_RESOURCE")

	_, err = manager.Delete(context.Background(), apitypes.ResourceKind("Unknown"), "example")
	assertResourceError(t, err, 400, "UNKNOWN_RESOURCE_KIND")

	_, err = manager.Delete(context.Background(), apitypes.ResourceKindResourceList, "bundle")
	assertResourceError(t, err, 400, "UNSUPPORTED_RESOURCE_DELETE")

	_, err = manager.Delete(context.Background(), apitypes.ResourceKindPeerConfig, "gear")
	assertResourceError(t, err, 400, "UNSUPPORTED_RESOURCE_DELETE")

	_, err = manager.Delete(context.Background(), apitypes.ResourceKindCredential, "example")
	assertResourceError(t, err, 500, "RESOURCE_SERVICE_NOT_CONFIGURED")
}

func TestDeleteRejectsMissingServicesByKind(t *testing.T) {
	manager := New(Services{})
	tests := []struct {
		name string
		kind apitypes.ResourceKind
	}{
		{name: "credential", kind: apitypes.ResourceKindCredential},
		{name: "model", kind: apitypes.ResourceKindModel},
		{name: "minimax tenant", kind: apitypes.ResourceKindMiniMaxTenant},
		{name: "voice", kind: apitypes.ResourceKindVoice},
		{name: "workspace", kind: apitypes.ResourceKindWorkspace},
		{name: "workflow", kind: apitypes.ResourceKindWorkflow},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.Delete(context.Background(), tc.kind, "example")
			assertResourceError(t, err, 500, "RESOURCE_SERVICE_NOT_CONFIGURED")
		})
	}
}

func TestDeleteReturnsNotFoundByKind(t *testing.T) {
	tests := []struct {
		name    string
		kind    apitypes.ResourceKind
		manager *Manager
	}{
		{name: "credential", kind: apitypes.ResourceKindCredential, manager: New(Services{Credentials: newFakeCredentials()})},
		{name: "model", kind: apitypes.ResourceKindModel, manager: newModelManager()},
		{name: "minimax tenant", kind: apitypes.ResourceKindMiniMaxTenant, manager: New(Services{MiniMax: newFakeMiniMax()})},
		{name: "voice", kind: apitypes.ResourceKindVoice, manager: New(Services{MiniMax: newFakeMiniMax()})},
		{name: "workspace", kind: apitypes.ResourceKindWorkspace, manager: New(Services{Workspaces: newFakeWorkspaces()})},
		{name: "workflow", kind: apitypes.ResourceKindWorkflow, manager: New(Services{Workflows: newFakeWorkflows()})},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.manager.Delete(context.Background(), tc.kind, "missing")
			assertResourceError(t, err, 404, "RESOURCE_NOT_FOUND")
		})
	}
}

func TestDeleteRemovesResourcesByKind(t *testing.T) {
	credentials := newFakeCredentials()
	models := newModelManager().services.Models
	minimax := newFakeMiniMax()
	workspaces := newFakeWorkspaces()
	workflows := newFakeWorkflows()
	manager := New(Services{
		Credentials: credentials,
		Models:      models,
		MiniMax:     minimax,
		Workspaces:  workspaces,
		Workflows:   workflows,
	})

	for _, resource := range []apitypes.Resource{
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Credential","metadata":{"name":"credential"},"spec":{"provider":"minimax","method":"api_key","body":{"api_key":"secret"}}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Model","metadata":{"name":"model"},"spec":{"kind":"llm","provider":{"kind":"openai-compatible","name":"main"},"source":"manual"}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workflow","metadata":{"name":"workflow"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"FlowcraftWorkflow","metadata":{"name":"workflow"},"spec":{}}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workflow_name":"workflow"}}`),
	} {
		if _, err := manager.Put(context.Background(), resource); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	}

	tests := []struct {
		kind apitypes.ResourceKind
		name string
	}{
		{apitypes.ResourceKindCredential, "credential"},
		{apitypes.ResourceKindModel, "model"},
		{apitypes.ResourceKindMiniMaxTenant, "tenant"},
		{apitypes.ResourceKindVoice, "voice"},
		{apitypes.ResourceKindWorkspace, "workspace"},
		{apitypes.ResourceKindWorkflow, "workflow"},
	}
	for _, tc := range tests {
		t.Run(string(tc.kind), func(t *testing.T) {
			if _, err := manager.Delete(context.Background(), tc.kind, tc.name); err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			_, err := manager.Get(context.Background(), tc.kind, tc.name)
			assertResourceError(t, err, 404, "RESOURCE_NOT_FOUND")
		})
	}
}

func TestApplyRejectsNilManager(t *testing.T) {
	var manager *Manager

	_, err := manager.Apply(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "Credential",
		"metadata": {"name": "example"},
		"spec": {
			"provider": "minimax",
			"method": "api_key",
			"body": {"api_key": "secret"}
		}
	}`))
	assertResourceError(t, err, 500, "RESOURCE_MANAGER_NOT_CONFIGURED")
}

func TestApplyRejectsMissingServicesByKind(t *testing.T) {
	manager := New(Services{})
	tests := []struct {
		name     string
		resource string
	}{
		{name: "credential", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Credential","metadata":{"name":"name"},"spec":{"provider":"minimax","method":"api_key","body":{"api_key":"secret"}}}`},
		{name: "peer config", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"PeerConfig","metadata":{"name":"gear"},"spec":{}}`},
		{name: "model", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Model","metadata":{"name":"model"},"spec":{"kind":"llm","provider":{"kind":"openai-compatible","name":"main"},"source":"manual"}}`},
		{name: "minimax tenant", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`},
		{name: "voice", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`},
		{name: "workspace", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workflow_name":"workflow"}}`},
		{name: "workflow", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workflow","metadata":{"name":"workflow"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"FlowcraftWorkflow","metadata":{"name":"workflow"},"spec":{}}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.Apply(context.Background(), mustResource(t, tc.resource))
			assertResourceError(t, err, 500, "RESOURCE_SERVICE_NOT_CONFIGURED")
		})
	}
}

func TestApplyRejectsUnsupportedVersionByKind(t *testing.T) {
	manager := New(Services{
		Credentials: newFakeCredentials(),
		Peers:       newFakePeers(),
		Models:      newModelManager().services.Models,
		MiniMax:     newFakeMiniMax(),
		Workspaces:  newFakeWorkspaces(),
		Workflows:   newFakeWorkflows(),
	})
	tests := []struct {
		name     string
		resource string
	}{
		{name: "credential", resource: `{"apiVersion":"unsupported","kind":"Credential","metadata":{"name":"name"},"spec":{"provider":"minimax","method":"api_key","body":{"api_key":"secret"}}}`},
		{name: "peer config", resource: `{"apiVersion":"unsupported","kind":"PeerConfig","metadata":{"name":"gear"},"spec":{}}`},
		{name: "model", resource: `{"apiVersion":"unsupported","kind":"Model","metadata":{"name":"model"},"spec":{"kind":"llm","provider":{"kind":"openai-compatible","name":"main"},"source":"manual"}}`},
		{name: "minimax tenant", resource: `{"apiVersion":"unsupported","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`},
		{name: "resource list", resource: `{"apiVersion":"unsupported","kind":"ResourceList","metadata":{"name":"bundle"},"spec":{"items":[]}}`},
		{name: "voice", resource: `{"apiVersion":"unsupported","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`},
		{name: "workspace", resource: `{"apiVersion":"unsupported","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workflow_name":"workflow"}}`},
		{name: "workflow", resource: `{"apiVersion":"unsupported","kind":"Workflow","metadata":{"name":"workflow"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"FlowcraftWorkflow","metadata":{"name":"workflow"},"spec":{}}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manager.Apply(context.Background(), mustResource(t, tc.resource))
			assertResourceError(t, err, 400, "UNSUPPORTED_RESOURCE_VERSION")
		})
	}
}

func assertResourceError(t *testing.T, err error, statusCode int, code string) {
	t.Helper()
	resourceErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error = %T %v, want *Error", err, err)
	}
	if resourceErr.StatusCode != statusCode {
		t.Fatalf("StatusCode = %d, want %d", resourceErr.StatusCode, statusCode)
	}
	if resourceErr.Code != code {
		t.Fatalf("Code = %q, want %q", resourceErr.Code, code)
	}
}
