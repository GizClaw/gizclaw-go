package resourcemanager

import (
	"context"
	"fmt"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/credential"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gear"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/mmx"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/workspace"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/workspacetemplate"
)

// Services groups the admin services that own concrete resource writes.
type Services struct {
	Credentials        credential.CredentialAdminService
	Gears              gear.GearsAdminService
	MiniMax            mmx.MiniMaxAdminService
	Workspaces         workspace.WorkspaceAdminService
	WorkspaceTemplates workspacetemplate.WorkspaceTemplateAdminService
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
	case apitypes.ResourceKindGearConfig:
		if m.services.Gears == nil {
			return apitypes.Resource{}, missingService("gears")
		}
		item, err := m.getGearConfig(ctx, adminservice.PublicKey(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		return resourceFromGearConfig(name, item)
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
	case apitypes.ResourceKindWorkspaceTemplate:
		if m.services.WorkspaceTemplates == nil {
			return apitypes.Resource{}, missingService("workspace templates")
		}
		item, exists, err := m.getWorkspaceTemplate(ctx, adminservice.WorkspaceTemplateName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromWorkspaceTemplate(name, item)
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
	case string(apitypes.ResourceKindGearConfig), "GearConfigResource":
		if m.services.Gears == nil {
			return apitypes.Resource{}, missingService("gears")
		}
		item, err := resource.AsGearConfigResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_GEAR_CONFIG_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putGearConfig(ctx, adminservice.PublicKey(pathParam(item.Metadata.Name)), item.Spec); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindGearConfig, item.Metadata.Name)
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
	case string(apitypes.ResourceKindWorkspaceTemplate), "WorkspaceTemplateResource":
		if m.services.WorkspaceTemplates == nil {
			return apitypes.Resource{}, missingService("workspace templates")
		}
		item, err := resource.AsWorkspaceTemplateResource()
		if err != nil {
			return apitypes.Resource{}, applyError(400, "INVALID_WORKSPACE_TEMPLATE_RESOURCE", err.Error())
		}
		if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
			return apitypes.Resource{}, err
		}
		if err := m.putWorkspaceTemplate(ctx, adminservice.WorkspaceTemplateName(pathParam(item.Metadata.Name)), item.Spec); err != nil {
			return apitypes.Resource{}, err
		}
		return m.Get(ctx, apitypes.ResourceKindWorkspaceTemplate, item.Metadata.Name)
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
	case apitypes.ResourceKindGearConfig:
		return apitypes.Resource{}, applyError(400, "UNSUPPORTED_RESOURCE_DELETE", "GearConfig cannot be deleted independently")
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
	case apitypes.ResourceKindWorkspaceTemplate:
		if m.services.WorkspaceTemplates == nil {
			return apitypes.Resource{}, missingService("workspace templates")
		}
		item, exists, err := m.deleteWorkspaceTemplate(ctx, adminservice.WorkspaceTemplateName(pathParam(name)))
		if err != nil {
			return apitypes.Resource{}, err
		}
		if !exists {
			return apitypes.Resource{}, notFound(kind, name)
		}
		return resourceFromWorkspaceTemplate(name, item)
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
	case string(apitypes.ResourceKindGearConfig), "GearConfigResource":
		return m.applyGearConfig(ctx, resource)
	case string(apitypes.ResourceKindMiniMaxTenant), "MiniMaxTenantResource":
		return m.applyMiniMaxTenant(ctx, resource)
	case string(apitypes.ResourceKindVolcTenant), "VolcTenantResource":
		return m.applyVolcTenant(ctx, resource)
	case string(apitypes.ResourceKindResourceList), "ResourceListResource":
		return m.applyResourceList(ctx, resource)
	case string(apitypes.ResourceKindVoice), "VoiceResource":
		return m.applyVoice(ctx, resource)
	case string(apitypes.ResourceKindWorkspace), "WorkspaceResource":
		return m.applyWorkspace(ctx, resource)
	case string(apitypes.ResourceKindWorkspaceTemplate), "WorkspaceTemplateResource":
		return m.applyWorkspaceTemplate(ctx, resource)
	default:
		return apitypes.ApplyResult{}, applyError(400, "UNKNOWN_RESOURCE_KIND", fmt.Sprintf("unknown resource kind %q", kind))
	}
}
