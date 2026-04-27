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
		{name: "gear config", kind: apitypes.ResourceKindGearConfig, manager: New(Services{Gears: newFakeGears()}), wantCode: "GEAR_NOT_FOUND"},
		{name: "minimax tenant", kind: apitypes.ResourceKindMiniMaxTenant, manager: New(Services{MiniMax: newFakeMiniMax()}), wantCode: "RESOURCE_NOT_FOUND"},
		{name: "voice", kind: apitypes.ResourceKindVoice, manager: New(Services{MiniMax: newFakeMiniMax()}), wantCode: "RESOURCE_NOT_FOUND"},
		{name: "workspace", kind: apitypes.ResourceKindWorkspace, manager: New(Services{Workspaces: newFakeWorkspaces()}), wantCode: "RESOURCE_NOT_FOUND"},
		{name: "workspace template", kind: apitypes.ResourceKindWorkspaceTemplate, manager: New(Services{WorkspaceTemplates: newFakeWorkspaceTemplates()}), wantCode: "RESOURCE_NOT_FOUND"},
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
		{name: "gear config", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"GearConfig","metadata":{"name":"gear"},"spec":{}}`},
		{name: "minimax tenant", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`},
		{name: "voice", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`},
		{name: "workspace", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workspace_template_name":"template"}}`},
		{name: "workspace template", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"WorkspaceTemplate","metadata":{"name":"template"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"SingleAgentGraphWorkflowTemplate","metadata":{"name":"template"},"spec":{}}}`},
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
		Credentials:        newFakeCredentials(),
		Gears:              newFakeGears(),
		MiniMax:            newFakeMiniMax(),
		Workspaces:         newFakeWorkspaces(),
		WorkspaceTemplates: newFakeWorkspaceTemplates(),
	})
	tests := []struct {
		name     string
		resource string
	}{
		{name: "credential", resource: `{"apiVersion":"unsupported","kind":"Credential","metadata":{"name":"name"},"spec":{"provider":"minimax","method":"api_key","body":{"api_key":"secret"}}}`},
		{name: "gear config", resource: `{"apiVersion":"unsupported","kind":"GearConfig","metadata":{"name":"gear"},"spec":{}}`},
		{name: "minimax tenant", resource: `{"apiVersion":"unsupported","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`},
		{name: "resource list", resource: `{"apiVersion":"unsupported","kind":"ResourceList","metadata":{"name":"bundle"},"spec":{"items":[]}}`},
		{name: "voice", resource: `{"apiVersion":"unsupported","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`},
		{name: "workspace", resource: `{"apiVersion":"unsupported","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workspace_template_name":"template"}}`},
		{name: "workspace template", resource: `{"apiVersion":"unsupported","kind":"WorkspaceTemplate","metadata":{"name":"template"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"SingleAgentGraphWorkflowTemplate","metadata":{"name":"template"},"spec":{}}}`},
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

	_, err = manager.Delete(context.Background(), apitypes.ResourceKindGearConfig, "gear")
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
		{name: "minimax tenant", kind: apitypes.ResourceKindMiniMaxTenant},
		{name: "voice", kind: apitypes.ResourceKindVoice},
		{name: "workspace", kind: apitypes.ResourceKindWorkspace},
		{name: "workspace template", kind: apitypes.ResourceKindWorkspaceTemplate},
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
		{name: "minimax tenant", kind: apitypes.ResourceKindMiniMaxTenant, manager: New(Services{MiniMax: newFakeMiniMax()})},
		{name: "voice", kind: apitypes.ResourceKindVoice, manager: New(Services{MiniMax: newFakeMiniMax()})},
		{name: "workspace", kind: apitypes.ResourceKindWorkspace, manager: New(Services{Workspaces: newFakeWorkspaces()})},
		{name: "workspace template", kind: apitypes.ResourceKindWorkspaceTemplate, manager: New(Services{WorkspaceTemplates: newFakeWorkspaceTemplates()})},
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
	minimax := newFakeMiniMax()
	workspaces := newFakeWorkspaces()
	templates := newFakeWorkspaceTemplates()
	manager := New(Services{
		Credentials:        credentials,
		MiniMax:            minimax,
		Workspaces:         workspaces,
		WorkspaceTemplates: templates,
	})

	for _, resource := range []apitypes.Resource{
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Credential","metadata":{"name":"credential"},"spec":{"provider":"minimax","method":"api_key","body":{"api_key":"secret"}}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"WorkspaceTemplate","metadata":{"name":"template"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"SingleAgentGraphWorkflowTemplate","metadata":{"name":"template"},"spec":{}}}`),
		mustResource(t, `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workspace_template_name":"template"}}`),
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
		{apitypes.ResourceKindMiniMaxTenant, "tenant"},
		{apitypes.ResourceKindVoice, "voice"},
		{apitypes.ResourceKindWorkspace, "workspace"},
		{apitypes.ResourceKindWorkspaceTemplate, "template"},
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
		{name: "gear config", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"GearConfig","metadata":{"name":"gear"},"spec":{}}`},
		{name: "minimax tenant", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`},
		{name: "voice", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`},
		{name: "workspace", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workspace_template_name":"template"}}`},
		{name: "workspace template", resource: `{"apiVersion":"gizclaw.admin/v1alpha1","kind":"WorkspaceTemplate","metadata":{"name":"template"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"SingleAgentGraphWorkflowTemplate","metadata":{"name":"template"},"spec":{}}}`},
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
		Credentials:        newFakeCredentials(),
		Gears:              newFakeGears(),
		MiniMax:            newFakeMiniMax(),
		Workspaces:         newFakeWorkspaces(),
		WorkspaceTemplates: newFakeWorkspaceTemplates(),
	})
	tests := []struct {
		name     string
		resource string
	}{
		{name: "credential", resource: `{"apiVersion":"unsupported","kind":"Credential","metadata":{"name":"name"},"spec":{"provider":"minimax","method":"api_key","body":{"api_key":"secret"}}}`},
		{name: "gear config", resource: `{"apiVersion":"unsupported","kind":"GearConfig","metadata":{"name":"gear"},"spec":{}}`},
		{name: "minimax tenant", resource: `{"apiVersion":"unsupported","kind":"MiniMaxTenant","metadata":{"name":"tenant"},"spec":{"app_id":"app","group_id":"group","credential_name":"credential"}}`},
		{name: "resource list", resource: `{"apiVersion":"unsupported","kind":"ResourceList","metadata":{"name":"bundle"},"spec":{"items":[]}}`},
		{name: "voice", resource: `{"apiVersion":"unsupported","kind":"Voice","metadata":{"name":"voice"},"spec":{"provider":{"kind":"minimax","name":"tenant"},"source":"manual"}}`},
		{name: "workspace", resource: `{"apiVersion":"unsupported","kind":"Workspace","metadata":{"name":"workspace"},"spec":{"workspace_template_name":"template"}}`},
		{name: "workspace template", resource: `{"apiVersion":"unsupported","kind":"WorkspaceTemplate","metadata":{"name":"template"},"spec":{"apiVersion":"gizclaw.flowcraft/v1alpha1","kind":"SingleAgentGraphWorkflowTemplate","metadata":{"name":"template"},"spec":{}}}`},
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
