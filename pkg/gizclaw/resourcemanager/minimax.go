package resourcemanager

import (
	"context"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func (m *Manager) applyMiniMaxTenant(ctx context.Context, resource apitypes.Resource) (apitypes.ApplyResult, error) {
	if m.services.MiniMax == nil {
		return apitypes.ApplyResult{}, missingService("minimax")
	}
	item, err := resource.AsMiniMaxTenantResource()
	if err != nil {
		return apitypes.ApplyResult{}, applyError(400, "INVALID_MINIMAX_TENANT_RESOURCE", err.Error())
	}
	if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
		return apitypes.ApplyResult{}, err
	}
	name := adminservice.MiniMaxTenantName(pathParam(item.Metadata.Name))
	existing, exists, err := m.getMiniMaxTenant(ctx, name)
	if err != nil {
		return apitypes.ApplyResult{}, err
	}
	if exists {
		same, err := semanticEqual(miniMaxTenantSpec(existing), item.Spec)
		if err != nil {
			return apitypes.ApplyResult{}, applyError(500, "RESOURCE_COMPARE_FAILED", err.Error())
		}
		if same {
			return applyResult(apitypes.ApplyActionUnchanged, apitypes.ResourceKindMiniMaxTenant, item.Metadata.Name), nil
		}
	}
	if err := m.putMiniMaxTenant(ctx, name, miniMaxTenantUpsert(item)); err != nil {
		return apitypes.ApplyResult{}, err
	}
	if exists {
		return applyResult(apitypes.ApplyActionUpdated, apitypes.ResourceKindMiniMaxTenant, item.Metadata.Name), nil
	}
	return applyResult(apitypes.ApplyActionCreated, apitypes.ResourceKindMiniMaxTenant, item.Metadata.Name), nil
}

func (m *Manager) applyVoice(ctx context.Context, resource apitypes.Resource) (apitypes.ApplyResult, error) {
	if m.services.MiniMax == nil {
		return apitypes.ApplyResult{}, missingService("minimax")
	}
	item, err := resource.AsVoiceResource()
	if err != nil {
		return apitypes.ApplyResult{}, applyError(400, "INVALID_VOICE_RESOURCE", err.Error())
	}
	if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
		return apitypes.ApplyResult{}, err
	}
	id := adminservice.VoiceID(pathParam(item.Metadata.Name))
	existing, exists, err := m.getVoice(ctx, id)
	if err != nil {
		return apitypes.ApplyResult{}, err
	}
	if exists {
		same, err := semanticEqual(voiceSpec(existing), item.Spec)
		if err != nil {
			return apitypes.ApplyResult{}, applyError(500, "RESOURCE_COMPARE_FAILED", err.Error())
		}
		if same {
			return applyResult(apitypes.ApplyActionUnchanged, apitypes.ResourceKindVoice, item.Metadata.Name), nil
		}
	}
	if err := m.putVoice(ctx, id, voiceUpsert(item)); err != nil {
		return apitypes.ApplyResult{}, err
	}
	if exists {
		return applyResult(apitypes.ApplyActionUpdated, apitypes.ResourceKindVoice, item.Metadata.Name), nil
	}
	return applyResult(apitypes.ApplyActionCreated, apitypes.ResourceKindVoice, item.Metadata.Name), nil
}

func (m *Manager) getMiniMaxTenant(ctx context.Context, name adminservice.MiniMaxTenantName) (apitypes.MiniMaxTenant, bool, error) {
	response, err := m.services.MiniMax.GetMiniMaxTenant(ctx, adminservice.GetMiniMaxTenantRequestObject{Name: name})
	if err != nil {
		return apitypes.MiniMaxTenant{}, false, err
	}
	switch response := response.(type) {
	case adminservice.GetMiniMaxTenant200JSONResponse:
		return apitypes.MiniMaxTenant(response), true, nil
	case adminservice.GetMiniMaxTenant404JSONResponse:
		return apitypes.MiniMaxTenant{}, false, nil
	case adminservice.GetMiniMaxTenant500JSONResponse:
		return apitypes.MiniMaxTenant{}, false, responseError(500, "GET_MINIMAX_TENANT_FAILED", "failed to get minimax tenant", response)
	default:
		return apitypes.MiniMaxTenant{}, false, unexpectedResponse("GetMiniMaxTenant", response)
	}
}

func (m *Manager) putMiniMaxTenant(ctx context.Context, name adminservice.MiniMaxTenantName, body adminservice.MiniMaxTenantUpsert) error {
	response, err := m.services.MiniMax.PutMiniMaxTenant(ctx, adminservice.PutMiniMaxTenantRequestObject{Name: name, Body: &body})
	if err != nil {
		return err
	}
	switch response := response.(type) {
	case adminservice.PutMiniMaxTenant200JSONResponse:
		return nil
	case adminservice.PutMiniMaxTenant400JSONResponse:
		return responseError(400, "PUT_MINIMAX_TENANT_FAILED", "failed to put minimax tenant", response)
	case adminservice.PutMiniMaxTenant500JSONResponse:
		return responseError(500, "PUT_MINIMAX_TENANT_FAILED", "failed to put minimax tenant", response)
	default:
		return unexpectedResponse("PutMiniMaxTenant", response)
	}
}

func (m *Manager) deleteMiniMaxTenant(ctx context.Context, name adminservice.MiniMaxTenantName) (apitypes.MiniMaxTenant, bool, error) {
	response, err := m.services.MiniMax.DeleteMiniMaxTenant(ctx, adminservice.DeleteMiniMaxTenantRequestObject{Name: name})
	if err != nil {
		return apitypes.MiniMaxTenant{}, false, err
	}
	switch response := response.(type) {
	case adminservice.DeleteMiniMaxTenant200JSONResponse:
		return apitypes.MiniMaxTenant(response), true, nil
	case adminservice.DeleteMiniMaxTenant404JSONResponse:
		return apitypes.MiniMaxTenant{}, false, nil
	case adminservice.DeleteMiniMaxTenant500JSONResponse:
		return apitypes.MiniMaxTenant{}, false, responseError(500, "DELETE_MINIMAX_TENANT_FAILED", "failed to delete minimax tenant", response)
	default:
		return apitypes.MiniMaxTenant{}, false, unexpectedResponse("DeleteMiniMaxTenant", response)
	}
}

func (m *Manager) getVoice(ctx context.Context, id adminservice.VoiceID) (apitypes.Voice, bool, error) {
	response, err := m.services.MiniMax.GetVoice(ctx, adminservice.GetVoiceRequestObject{Id: id})
	if err != nil {
		return apitypes.Voice{}, false, err
	}
	switch response := response.(type) {
	case adminservice.GetVoice200JSONResponse:
		return apitypes.Voice(response), true, nil
	case adminservice.GetVoice404JSONResponse:
		return apitypes.Voice{}, false, nil
	case adminservice.GetVoice500JSONResponse:
		return apitypes.Voice{}, false, responseError(500, "GET_VOICE_FAILED", "failed to get voice", response)
	default:
		return apitypes.Voice{}, false, unexpectedResponse("GetVoice", response)
	}
}

func (m *Manager) putVoice(ctx context.Context, id adminservice.VoiceID, body adminservice.VoiceUpsert) error {
	response, err := m.services.MiniMax.PutVoice(ctx, adminservice.PutVoiceRequestObject{Id: id, Body: &body})
	if err != nil {
		return err
	}
	switch response := response.(type) {
	case adminservice.PutVoice200JSONResponse:
		return nil
	case adminservice.PutVoice400JSONResponse:
		return responseError(400, "PUT_VOICE_FAILED", "failed to put voice", response)
	case adminservice.PutVoice409JSONResponse:
		return responseError(409, "PUT_VOICE_FAILED", "failed to put voice", response)
	case adminservice.PutVoice500JSONResponse:
		return responseError(500, "PUT_VOICE_FAILED", "failed to put voice", response)
	default:
		return unexpectedResponse("PutVoice", response)
	}
}

func (m *Manager) deleteVoice(ctx context.Context, id adminservice.VoiceID) (apitypes.Voice, bool, error) {
	response, err := m.services.MiniMax.DeleteVoice(ctx, adminservice.DeleteVoiceRequestObject{Id: id})
	if err != nil {
		return apitypes.Voice{}, false, err
	}
	switch response := response.(type) {
	case adminservice.DeleteVoice200JSONResponse:
		return apitypes.Voice(response), true, nil
	case adminservice.DeleteVoice404JSONResponse:
		return apitypes.Voice{}, false, nil
	case adminservice.DeleteVoice500JSONResponse:
		return apitypes.Voice{}, false, responseError(500, "DELETE_VOICE_FAILED", "failed to delete voice", response)
	default:
		return apitypes.Voice{}, false, unexpectedResponse("DeleteVoice", response)
	}
}

func miniMaxTenantSpec(tenant apitypes.MiniMaxTenant) apitypes.MiniMaxTenantSpec {
	return apitypes.MiniMaxTenantSpec{
		AppId:          tenant.AppId,
		BaseUrl:        tenant.BaseUrl,
		CredentialName: tenant.CredentialName,
		Description:    tenant.Description,
		GroupId:        tenant.GroupId,
	}
}

func miniMaxTenantUpsert(resource apitypes.MiniMaxTenantResource) adminservice.MiniMaxTenantUpsert {
	return adminservice.MiniMaxTenantUpsert{
		AppId:          resource.Spec.AppId,
		BaseUrl:        resource.Spec.BaseUrl,
		CredentialName: resource.Spec.CredentialName,
		Description:    resource.Spec.Description,
		GroupId:        resource.Spec.GroupId,
		Name:           apitypes.MiniMaxTenantName(resource.Metadata.Name),
	}
}

func voiceSpec(voice apitypes.Voice) apitypes.VoiceSpec {
	return apitypes.VoiceSpec{
		Description:       voice.Description,
		Name:              voice.Name,
		Provider:          voice.Provider,
		ProviderVoiceId:   voice.ProviderVoiceId,
		ProviderVoiceType: voice.ProviderVoiceType,
		Raw:               voice.Raw,
		Source:            voice.Source,
	}
}

func voiceUpsert(resource apitypes.VoiceResource) adminservice.VoiceUpsert {
	return adminservice.VoiceUpsert{
		Description:       resource.Spec.Description,
		Id:                apitypes.VoiceID(resource.Metadata.Name),
		Name:              resource.Spec.Name,
		Provider:          resource.Spec.Provider,
		ProviderVoiceId:   resource.Spec.ProviderVoiceId,
		ProviderVoiceType: resource.Spec.ProviderVoiceType,
		Raw:               resource.Spec.Raw,
		Source:            resource.Spec.Source,
	}
}

func resourceFromMiniMaxTenant(item apitypes.MiniMaxTenant) (apitypes.Resource, error) {
	return marshalResource(apitypes.MiniMaxTenantResource{
		ApiVersion: apitypes.ResourceAPIVersionGizclawAdminv1alpha1,
		Kind:       apitypes.MiniMaxTenantResourceKind(apitypes.ResourceKindMiniMaxTenant),
		Metadata:   apitypes.ResourceMetadata{Name: string(item.Name)},
		Spec:       miniMaxTenantSpec(item),
	})
}

func resourceFromVoice(item apitypes.Voice) (apitypes.Resource, error) {
	return marshalResource(apitypes.VoiceResource{
		ApiVersion: apitypes.ResourceAPIVersionGizclawAdminv1alpha1,
		Kind:       apitypes.VoiceResourceKind(apitypes.ResourceKindVoice),
		Metadata:   apitypes.ResourceMetadata{Name: string(item.Id)},
		Spec:       voiceSpec(item),
	})
}
