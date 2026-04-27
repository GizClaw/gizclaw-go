package resourcemanager

import (
	"context"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func (m *Manager) applyWorkspaceTemplate(ctx context.Context, resource apitypes.Resource) (apitypes.ApplyResult, error) {
	if m.services.WorkspaceTemplates == nil {
		return apitypes.ApplyResult{}, missingService("workspace templates")
	}
	item, err := resource.AsWorkspaceTemplateResource()
	if err != nil {
		return apitypes.ApplyResult{}, applyError(400, "INVALID_WORKSPACE_TEMPLATE_RESOURCE", err.Error())
	}
	if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
		return apitypes.ApplyResult{}, err
	}
	name := adminservice.WorkspaceTemplateName(pathParam(item.Metadata.Name))
	existing, exists, err := m.getWorkspaceTemplate(ctx, name)
	if err != nil {
		return apitypes.ApplyResult{}, err
	}
	if exists {
		same, err := semanticEqual(existing, item.Spec)
		if err != nil {
			return apitypes.ApplyResult{}, applyError(500, "RESOURCE_COMPARE_FAILED", err.Error())
		}
		if same {
			return applyResult(apitypes.ApplyActionUnchanged, apitypes.ResourceKindWorkspaceTemplate, item.Metadata.Name), nil
		}
	}
	if err := m.putWorkspaceTemplate(ctx, name, item.Spec); err != nil {
		return apitypes.ApplyResult{}, err
	}
	if exists {
		return applyResult(apitypes.ApplyActionUpdated, apitypes.ResourceKindWorkspaceTemplate, item.Metadata.Name), nil
	}
	return applyResult(apitypes.ApplyActionCreated, apitypes.ResourceKindWorkspaceTemplate, item.Metadata.Name), nil
}

func (m *Manager) getWorkspaceTemplate(ctx context.Context, name adminservice.WorkspaceTemplateName) (apitypes.WorkflowTemplateDocument, bool, error) {
	response, err := m.services.WorkspaceTemplates.GetWorkspaceTemplate(ctx, adminservice.GetWorkspaceTemplateRequestObject{Name: name})
	if err != nil {
		return apitypes.WorkflowTemplateDocument{}, false, err
	}
	switch response := response.(type) {
	case adminservice.GetWorkspaceTemplate200JSONResponse:
		return apitypes.WorkflowTemplateDocument(response), true, nil
	case adminservice.GetWorkspaceTemplate404JSONResponse:
		return apitypes.WorkflowTemplateDocument{}, false, nil
	case adminservice.GetWorkspaceTemplate500JSONResponse:
		return apitypes.WorkflowTemplateDocument{}, false, responseError(500, "GET_WORKSPACE_TEMPLATE_FAILED", "failed to get workspace template", response)
	default:
		return apitypes.WorkflowTemplateDocument{}, false, unexpectedResponse("GetWorkspaceTemplate", response)
	}
}

func (m *Manager) putWorkspaceTemplate(ctx context.Context, name adminservice.WorkspaceTemplateName, body apitypes.WorkflowTemplateDocument) error {
	response, err := m.services.WorkspaceTemplates.PutWorkspaceTemplate(ctx, adminservice.PutWorkspaceTemplateRequestObject{Name: name, Body: &body})
	if err != nil {
		return err
	}
	switch response := response.(type) {
	case adminservice.PutWorkspaceTemplate200JSONResponse:
		return nil
	case adminservice.PutWorkspaceTemplate400JSONResponse:
		return responseError(400, "PUT_WORKSPACE_TEMPLATE_FAILED", "failed to put workspace template", response)
	case adminservice.PutWorkspaceTemplate500JSONResponse:
		return responseError(500, "PUT_WORKSPACE_TEMPLATE_FAILED", "failed to put workspace template", response)
	default:
		return unexpectedResponse("PutWorkspaceTemplate", response)
	}
}

func (m *Manager) deleteWorkspaceTemplate(ctx context.Context, name adminservice.WorkspaceTemplateName) (apitypes.WorkflowTemplateDocument, bool, error) {
	response, err := m.services.WorkspaceTemplates.DeleteWorkspaceTemplate(ctx, adminservice.DeleteWorkspaceTemplateRequestObject{Name: name})
	if err != nil {
		return apitypes.WorkflowTemplateDocument{}, false, err
	}
	switch response := response.(type) {
	case adminservice.DeleteWorkspaceTemplate200JSONResponse:
		return apitypes.WorkflowTemplateDocument(response), true, nil
	case adminservice.DeleteWorkspaceTemplate404JSONResponse:
		return apitypes.WorkflowTemplateDocument{}, false, nil
	case adminservice.DeleteWorkspaceTemplate500JSONResponse:
		return apitypes.WorkflowTemplateDocument{}, false, responseError(500, "DELETE_WORKSPACE_TEMPLATE_FAILED", "failed to delete workspace template", response)
	default:
		return apitypes.WorkflowTemplateDocument{}, false, unexpectedResponse("DeleteWorkspaceTemplate", response)
	}
}

func resourceFromWorkspaceTemplate(name string, item apitypes.WorkflowTemplateDocument) (apitypes.Resource, error) {
	return marshalResource(apitypes.WorkspaceTemplateResource{
		ApiVersion: apitypes.ResourceAPIVersionGizclawAdminv1alpha1,
		Kind:       apitypes.WorkspaceTemplateResourceKind(apitypes.ResourceKindWorkspaceTemplate),
		Metadata:   apitypes.ResourceMetadata{Name: name},
		Spec:       item,
	})
}
