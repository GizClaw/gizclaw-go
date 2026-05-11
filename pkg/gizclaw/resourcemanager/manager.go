package resourcemanager

import (
	"context"
	"fmt"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/credential"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/mmx"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/modelcatalog"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/peer"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/workflow"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/workspace"
)

// Services groups the admin services that own concrete resource writes.
type Services struct {
	Credentials credential.CredentialAdminService
	Peers       peer.PeerAdminService
	Models      modelcatalog.AdminService
	MiniMax     mmx.MiniMaxAdminService
	Workspaces  workspace.WorkspaceAdminService
	Workflows   workflow.WorkflowAdminService
}

// Manager applies declarative admin resources by delegating to owner services.
type Manager struct {
	services Services
}

// Error is returned for apply failures that should map cleanly to HTTP later.
type Error struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New creates a resource manager using the provided owner services.
func New(services Services) *Manager {
	return &Manager{services: services}
}

// Get loads the current state of a named resource and returns it as a declarative resource.
func (m *Manager) Get(ctx context.Context, kind apitypes.ResourceKind, name string) (apitypes.Resource, error) {
	if m == nil {
		return apitypes.Resource{}, applyError(500, "RESOURCE_MANAGER_NOT_CONFIGURED", "resource manager is not configured")
	}
	if name == "" {
		return apitypes.Resource{}, applyError(400, "INVALID_RESOURCE", "metadata.name is required")
	}
	switch kind {
	case apitypes.ResourceKindCredential:
		if m.services.Credentials == nil {
			return apitypes.Resource{}, missingService("credentials")
		}
		item, exists, err := m.getCredential(ctx, adminservice.CredentialName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromCredential(item)
	case apitypes.ResourceKindPeerConfig:
		if m.services.Peers == nil {
			return apitypes.Resource{}, missingService("peers")
		}
		item, err := m.getPeerConfig(ctx, adminservice.PublicKey(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		return resourceFromPeerConfig(name, item)
	case apitypes.ResourceKindModel:
		if m.services.Models == nil {
			return apitypes.Resource{}, missingService("models")
		}
		item, exists, err := m.getModel(ctx, adminservice.ModelID(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromModel(item)
	case apitypes.ResourceKindMiniMaxTenant:
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("minimax")
		}
		item, exists, err := m.getMiniMaxTenant(ctx, adminservice.MiniMaxTenantName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromMiniMaxTenant(item)
	case apitypes.ResourceKindVolcTenant:
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("volc")
		}
		item, exists, err := m.getVolcTenant(ctx, adminservice.VolcTenantName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromVolcTenant(item)
	case apitypes.ResourceKindVoice:
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("minimax")
		}
		item, exists, err := m.getVoice(ctx, adminservice.VoiceID(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromVoice(item)
	case apitypes.ResourceKindWorkspace:
		if m.services.Workspaces == nil {
			return apitypes.Resource{}, missingService("workspaces")
		}
		item, exists, err := m.getWorkspace(ctx, adminservice.WorkspaceName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromWorkspace(item)
	case apitypes.ResourceKindWorkflow:
		if m.services.Workflows == nil {
			return apitypes.Resource{}, missingService("workflows")
		}
		item, exists, err := m.getWorkflow(ctx, adminservice.WorkflowName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromWorkflow(name, item)
	case apitypes.ResourceKindResourceList:
		return apitypes.Resource{}, applyError(400, "UNSUPPORTED_RESOURCE_GET", "ResourceList is not stored as a named resource")
	default:
		return apitypes.Resource{}, applyError(400, "UNKNOWN_RESOURCE_KIND", fmt.Sprintf("unknown resource kind %q", kind))
	}
}

// Put writes the provided resource and returns the stored resource state.
func (m *Manager) Put(ctx context.Context, resource apitypes.Resource) (apitypes.Resource, error) {
	if m == nil {
		return apitypes.Resource{}, applyError(500, "RESOURCE_MANAGER_NOT_CONFIGURED", "resource manager is not configured")
	}
	kind, err := resource.Discriminator()
	if err != nil {
		return apitypes.Resource{}, applyError(400, "INVALID_RESOURCE", err.Error())
	}
	switch kind {
	case string(apitypes.ResourceKindCredential), "CredentialResource":
		if m.services.Credentials == nil {
			return apitypes.Resource{}, missingService("credentials")
		}
		item, err := resource.AsCredentialResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_CREDENTIAL_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putCredential(ctx, adminservice.CredentialName(pathParam(item.Metadata.Name)), credentialUpsert(item)); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindCredential, item.Metadata.Name)
	case string(apitypes.ResourceKindPeerConfig), "PeerConfigResource":
		if m.services.Peers == nil {
			return apitypes.Resource{}, missingService("peers")
		}
		item, err := resource.AsPeerConfigResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_PEER_CONFIG_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putPeerConfig(ctx, adminservice.PublicKey(pathParam(item.Metadata.Name)), item.Spec); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindPeerConfig, item.Metadata.Name)
	case string(apitypes.ResourceKindMiniMaxTenant), "MiniMaxTenantResource":
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("minimax")
		}
		item, err := resource.AsMiniMaxTenantResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_MINIMAX_TENANT_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putMiniMaxTenant(ctx, adminservice.MiniMaxTenantName(pathParam(item.Metadata.Name)), miniMaxTenantUpsert(item)); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindMiniMaxTenant, item.Metadata.Name)
	case string(apitypes.ResourceKindModel), "ModelResource":
		if m.services.Models == nil {
			return apitypes.Resource{}, missingService("models")
		}
		item, err := resource.AsModelResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_MODEL_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putModel(ctx, adminservice.ModelID(pathParam(item.Metadata.Name)), modelUpsert(item)); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindModel, item.Metadata.Name)
	case string(apitypes.ResourceKindVolcTenant), "VolcTenantResource":
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("volc")
		}
		item, err := resource.AsVolcTenantResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_VOLC_TENANT_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putVolcTenant(ctx, adminservice.VolcTenantName(pathParam(item.Metadata.Name)), volcTenantUpsert(item)); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindVolcTenant, item.Metadata.Name)
	case string(apitypes.ResourceKindResourceList), "ResourceListResource":
		list, err := resource.AsResourceListResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_RESOURCE_LIST", err.Error())
		}
		if err := validateResourceHeader(list.ApiVersion, list.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		items := make([]apitypes.Resource, 0, len(list.Spec.Items))
		for _, item := range list.Spec.Items {
			stored, err := m.Put(ctx, item)
			if err != nil {
				return apitypes.Resource{}, err
			}
			items = append(items, stored)
		}
		return resourceFromResourceList(list.Metadata.Name, items)
	case string(apitypes.ResourceKindVoice), "VoiceResource":
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("minimax")
		}
		item, err := resource.AsVoiceResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_VOICE_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putVoice(ctx, adminservice.VoiceID(pathParam(item.Metadata.Name)), voiceUpsert(item)); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindVoice, item.Metadata.Name)
	case string(apitypes.ResourceKindWorkspace), "WorkspaceResource":
		if m.services.Workspaces == nil {
			return apitypes.Resource{}, missingService("workspaces")
		}
		item, err := resource.AsWorkspaceResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_WORKSPACE_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putWorkspace(ctx, adminservice.WorkspaceName(pathParam(item.Metadata.Name)), workspaceUpsert(item)); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindWorkspace, item.Metadata.Name)
	case string(apitypes.ResourceKindWorkflow), "WorkflowResource":
		if m.services.Workflows == nil {
			return apitypes.Resource{}, missingService("workflows")
		}
		item, err := resource.AsWorkflowResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_WORKFLOW_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putWorkflow(ctx, adminservice.WorkflowName(pathParam(item.Metadata.Name)), item.Spec); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindWorkflow, item.Metadata.Name)
	default:
		return apitypes.Resource{}, applyError(400, "UNKNOWN_RESOURCE_KIND", fmt.Sprintf("unknown resource kind %q", kind))
	}
}

// Delete removes a named concrete resource and returns the deleted resource state.
func (m *Manager) Delete(ctx context.Context, kind apitypes.ResourceKind, name string) (apitypes.Resource, error) {
	if m == nil {
		return apitypes.Resource{}, applyError(500, "RESOURCE_MANAGER_NOT_CONFIGURED", "resource manager is not configured")
	}
	if name == "" {
		return apitypes.Resource{}, applyError(400, "INVALID_RESOURCE", "metadata.name is required")
	}
	switch kind {
	case apitypes.ResourceKindCredential:
		if m.services.Credentials == nil {
			return apitypes.Resource{}, missingService("credentials")
		}
		item, exists, err := m.deleteCredential(ctx, adminservice.CredentialName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromCredential(item)
	case apitypes.ResourceKindPeerConfig:
		return apitypes.Resource{}, applyError(400, "UNSUPPORTED_RESOURCE_DELETE", "PeerConfig cannot be deleted independently")
	case apitypes.ResourceKindModel:
		if m.services.Models == nil {
			return apitypes.Resource{}, missingService("models")
		}
		item, exists, err := m.deleteModel(ctx, adminservice.ModelID(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromModel(item)
	case apitypes.ResourceKindMiniMaxTenant:
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("minimax")
		}
		item, exists, err := m.deleteMiniMaxTenant(ctx, adminservice.MiniMaxTenantName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromMiniMaxTenant(item)
	case apitypes.ResourceKindVolcTenant:
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("volc")
		}
		item, exists, err := m.deleteVolcTenant(ctx, adminservice.VolcTenantName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromVolcTenant(item)
	case apitypes.ResourceKindVoice:
		if m.services.MiniMax == nil {
			return apitypes.Resource{}, missingService("minimax")
		}
		item, exists, err := m.deleteVoice(ctx, adminservice.VoiceID(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromVoice(item)
	case apitypes.ResourceKindWorkspace:
		if m.services.Workspaces == nil {
			return apitypes.Resource{}, missingService("workspaces")
		}
		item, exists, err := m.deleteWorkspace(ctx, adminservice.WorkspaceName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromWorkspace(item)
	case apitypes.ResourceKindWorkflow:
		if m.services.Workflows == nil {
			return apitypes.Resource{}, missingService("workflows")
		}
		item, exists, err := m.deleteWorkflow(ctx, adminservice.WorkflowName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromWorkflow(name, item)
	case apitypes.ResourceKindResourceList:
		return apitypes.Resource{}, applyError(400, "UNSUPPORTED_RESOURCE_DELETE", "ResourceList is not stored as a named resource")
	default:
		return apitypes.Resource{}, applyError(400, "UNKNOWN_RESOURCE_KIND", fmt.Sprintf("unknown resource kind %q", kind))
	}
}

// Apply creates, updates, or leaves unchanged the provided resource.
func (m *Manager) Apply(ctx context.Context, resource apitypes.Resource) (apitypes.ApplyResult, error) {
	if m == nil {
		return apitypes.ApplyResult{}, applyError(500, "RESOURCE_MANAGER_NOT_CONFIGURED", "resource manager is not configured")
	}
	kind, err := resource.Discriminator()
	if err != nil {
		return apitypes.ApplyResult{}, applyError(400, "INVALID_RESOURCE", err.Error())
	}
	switch kind {
	case string(apitypes.ResourceKindCredential), "CredentialResource":
		return m.applyCredential(ctx, resource)
	case string(apitypes.ResourceKindPeerConfig), "PeerConfigResource":
		return m.applyPeerConfig(ctx, resource)
	case string(apitypes.ResourceKindMiniMaxTenant), "MiniMaxTenantResource":
		return m.applyMiniMaxTenant(ctx, resource)
	case string(apitypes.ResourceKindModel), "ModelResource":
		return m.applyModel(ctx, resource)
	case string(apitypes.ResourceKindVolcTenant), "VolcTenantResource":
		return m.applyVolcTenant(ctx, resource)
	case string(apitypes.ResourceKindResourceList), "ResourceListResource":
		return m.applyResourceList(ctx, resource)
	case string(apitypes.ResourceKindVoice), "VoiceResource":
		return m.applyVoice(ctx, resource)
	case string(apitypes.ResourceKindWorkspace), "WorkspaceResource":
		return m.applyWorkspace(ctx, resource)
	case string(apitypes.ResourceKindWorkflow), "WorkflowResource":
		return m.applyWorkflow(ctx, resource)
	default:
		return apitypes.ApplyResult{}, applyError(400, "UNKNOWN_RESOURCE_KIND", fmt.Sprintf("unknown resource kind %q", kind))
	}
}
