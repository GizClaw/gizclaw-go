package resourcemanager

import (
	"context"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func TestApplyWorkspaceTemplateCreatesResource(t *testing.T) {
	templates := newFakeWorkspaceTemplates()
	manager := New(Services{WorkspaceTemplates: templates})

	result, err := manager.Apply(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "WorkspaceTemplate",
		"metadata": {"name": "template"},
		"spec": {
			"apiVersion": "gizclaw.flowcraft/v1alpha1",
			"kind": "SingleAgentGraphWorkflowTemplate",
			"metadata": {"name": "template"},
			"spec": {"prompt": "hello"}
		}
	}`))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Action != apitypes.ApplyActionCreated {
		t.Fatalf("action = %q, want created", result.Action)
	}
	if templates.putCount != 1 {
		t.Fatalf("putCount = %d, want 1", templates.putCount)
	}
	if _, ok := templates.items["template"]; !ok {
		t.Fatal("stored template missing")
	}
}

func TestGetWorkspaceTemplateReturnsResource(t *testing.T) {
	templates := newFakeWorkspaceTemplates()
	templates.items["template"] = mustWorkflowTemplateDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "SingleAgentGraphWorkflowTemplate",
		"metadata": {"name": "template"},
		"spec": {"prompt": "hello"}
	}`)
	manager := New(Services{WorkspaceTemplates: templates})

	resource, err := manager.Get(context.Background(), apitypes.ResourceKindWorkspaceTemplate, "template")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	template, err := resource.AsWorkspaceTemplateResource()
	if err != nil {
		t.Fatalf("AsWorkspaceTemplateResource returned error: %v", err)
	}
	if template.Metadata.Name != "template" {
		t.Fatalf("metadata.name = %q, want template", template.Metadata.Name)
	}
}

func TestPutWorkspaceTemplateWritesResource(t *testing.T) {
	templates := newFakeWorkspaceTemplates()
	manager := New(Services{WorkspaceTemplates: templates})

	_, err := manager.Put(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "WorkspaceTemplate",
		"metadata": {"name": "template"},
		"spec": {
			"apiVersion": "gizclaw.flowcraft/v1alpha1",
			"kind": "SingleAgentGraphWorkflowTemplate",
			"metadata": {"name": "template"},
			"spec": {"prompt": "hello"}
		}
	}`))
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if templates.putCount != 1 {
		t.Fatalf("putCount = %d, want 1", templates.putCount)
	}
}

func TestApplyWorkspaceTemplateUnchangedSkipsPut(t *testing.T) {
	templates := newFakeWorkspaceTemplates()
	templates.items["template"] = mustWorkflowTemplateDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "SingleAgentGraphWorkflowTemplate",
		"metadata": {"name": "template"},
		"spec": {"prompt": "hello"}
	}`)
	manager := New(Services{WorkspaceTemplates: templates})

	result, err := manager.Apply(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "WorkspaceTemplate",
		"metadata": {"name": "template"},
		"spec": {
			"apiVersion": "gizclaw.flowcraft/v1alpha1",
			"kind": "SingleAgentGraphWorkflowTemplate",
			"metadata": {"name": "template"},
			"spec": {"prompt": "hello"}
		}
	}`))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Action != apitypes.ApplyActionUnchanged {
		t.Fatalf("action = %q, want unchanged", result.Action)
	}
	if templates.putCount != 0 {
		t.Fatalf("putCount = %d, want 0", templates.putCount)
	}
}

func TestApplyWorkspaceTemplateUpdatesResource(t *testing.T) {
	templates := newFakeWorkspaceTemplates()
	templates.items["template"] = mustWorkflowTemplateDocument(t, `{
		"apiVersion": "gizclaw.flowcraft/v1alpha1",
		"kind": "SingleAgentGraphWorkflowTemplate",
		"metadata": {"name": "template"},
		"spec": {"prompt": "old"}
	}`)
	manager := New(Services{WorkspaceTemplates: templates})

	result, err := manager.Apply(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "WorkspaceTemplate",
		"metadata": {"name": "template"},
		"spec": {
			"apiVersion": "gizclaw.flowcraft/v1alpha1",
			"kind": "SingleAgentGraphWorkflowTemplate",
			"metadata": {"name": "template"},
			"spec": {"prompt": "new"}
		}
	}`))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Action != apitypes.ApplyActionUpdated {
		t.Fatalf("action = %q, want updated", result.Action)
	}
	if templates.putCount != 1 {
		t.Fatalf("putCount = %d, want 1", templates.putCount)
	}
}

func TestWorkspaceTemplateServiceErrorResponses(t *testing.T) {
	templates := newFakeWorkspaceTemplates()
	manager := New(Services{WorkspaceTemplates: templates})

	templates.getStatus = 500
	_, _, err := manager.getWorkspaceTemplate(context.Background(), "template")
	assertResourceError(t, err, 500, "INTERNAL_ERROR")

	templates.getStatus = 0
	templates.putStatus = 400
	err = manager.putWorkspaceTemplate(context.Background(), "template", apitypes.WorkflowTemplateDocument{})
	assertResourceError(t, err, 400, "INVALID_TEMPLATE")

	templates.putStatus = 500
	err = manager.putWorkspaceTemplate(context.Background(), "template", apitypes.WorkflowTemplateDocument{})
	assertResourceError(t, err, 500, "INTERNAL_ERROR")
}

type fakeWorkspaceTemplates struct {
	items     map[string]apitypes.WorkflowTemplateDocument
	putCount  int
	getStatus int
	putStatus int
}

func newFakeWorkspaceTemplates() *fakeWorkspaceTemplates {
	return &fakeWorkspaceTemplates{items: map[string]apitypes.WorkflowTemplateDocument{}}
}

func (f *fakeWorkspaceTemplates) ListWorkspaceTemplates(context.Context, adminservice.ListWorkspaceTemplatesRequestObject) (adminservice.ListWorkspaceTemplatesResponseObject, error) {
	return nil, nil
}

func (f *fakeWorkspaceTemplates) CreateWorkspaceTemplate(context.Context, adminservice.CreateWorkspaceTemplateRequestObject) (adminservice.CreateWorkspaceTemplateResponseObject, error) {
	return nil, nil
}

func (f *fakeWorkspaceTemplates) DeleteWorkspaceTemplate(_ context.Context, request adminservice.DeleteWorkspaceTemplateRequestObject) (adminservice.DeleteWorkspaceTemplateResponseObject, error) {
	item, ok := f.items[string(request.Name)]
	if !ok {
		return adminservice.DeleteWorkspaceTemplate404JSONResponse(apitypes.NewErrorResponse("WORKSPACE_TEMPLATE_NOT_FOUND", "not found")), nil
	}
	delete(f.items, string(request.Name))
	return adminservice.DeleteWorkspaceTemplate200JSONResponse(item), nil
}

func (f *fakeWorkspaceTemplates) GetWorkspaceTemplate(_ context.Context, request adminservice.GetWorkspaceTemplateRequestObject) (adminservice.GetWorkspaceTemplateResponseObject, error) {
	if f.getStatus == 500 {
		return adminservice.GetWorkspaceTemplate500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", "failed")), nil
	}
	item, ok := f.items[string(request.Name)]
	if !ok {
		return adminservice.GetWorkspaceTemplate404JSONResponse(apitypes.NewErrorResponse("WORKSPACE_TEMPLATE_NOT_FOUND", "not found")), nil
	}
	return adminservice.GetWorkspaceTemplate200JSONResponse(item), nil
}

func (f *fakeWorkspaceTemplates) PutWorkspaceTemplate(_ context.Context, request adminservice.PutWorkspaceTemplateRequestObject) (adminservice.PutWorkspaceTemplateResponseObject, error) {
	switch f.putStatus {
	case 400:
		return adminservice.PutWorkspaceTemplate400JSONResponse(apitypes.NewErrorResponse("INVALID_TEMPLATE", "invalid")), nil
	case 500:
		return adminservice.PutWorkspaceTemplate500JSONResponse(apitypes.NewErrorResponse("INTERNAL_ERROR", "failed")), nil
	}
	f.putCount++
	f.items[string(request.Name)] = *request.Body
	return adminservice.PutWorkspaceTemplate200JSONResponse(*request.Body), nil
}
