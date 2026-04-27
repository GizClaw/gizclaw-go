package resourcemanager

import (
	"context"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func (m *Manager) applyGearConfig(ctx context.Context, resource apitypes.Resource) (apitypes.ApplyResult, error) {
	if m.services.Gears == nil {
		return apitypes.ApplyResult{}, missingService("gears")
	}
	item, err := resource.AsGearConfigResource()
	if err != nil {
		return apitypes.ApplyResult{}, applyError(400, "INVALID_GEAR_CONFIG_RESOURCE", err.Error())
	}
	if err := validateResourceHeader(item.ApiVersion, item.Metadata.Name); err != nil {
		return apitypes.ApplyResult{}, err
	}
	publicKey := adminservice.PublicKey(pathParam(item.Metadata.Name))
	existing, err := m.getGearConfig(ctx, publicKey)
	if err != nil {
		return apitypes.ApplyResult{}, err
	}
	same, err := semanticEqual(existing, item.Spec)
	if err != nil {
		return apitypes.ApplyResult{}, applyError(500, "RESOURCE_COMPARE_FAILED", err.Error())
	}
	if same {
		return applyResult(apitypes.ApplyActionUnchanged, apitypes.ResourceKindGearConfig, item.Metadata.Name), nil
	}
	if err := m.putGearConfig(ctx, publicKey, item.Spec); err != nil {
		return apitypes.ApplyResult{}, err
	}
	return applyResult(apitypes.ApplyActionUpdated, apitypes.ResourceKindGearConfig, item.Metadata.Name), nil
}

func (m *Manager) getGearConfig(ctx context.Context, publicKey adminservice.PublicKey) (apitypes.Configuration, error) {
	response, err := m.services.Gears.GetGearConfig(ctx, adminservice.GetGearConfigRequestObject{PublicKey: publicKey})
	if err != nil {
		return apitypes.Configuration{}, err
	}
	switch response := response.(type) {
	case adminservice.GetGearConfig200JSONResponse:
		return apitypes.Configuration(response), nil
	case adminservice.GetGearConfig404JSONResponse:
		return apitypes.Configuration{}, responseError(404, "GEAR_CONFIG_NOT_FOUND", "gear config not found", response)
	default:
		return apitypes.Configuration{}, unexpectedResponse("GetGearConfig", response)
	}
}

func (m *Manager) putGearConfig(ctx context.Context, publicKey adminservice.PublicKey, body apitypes.Configuration) error {
	response, err := m.services.Gears.PutGearConfig(ctx, adminservice.PutGearConfigRequestObject{PublicKey: publicKey, Body: &body})
	if err != nil {
		return err
	}
	switch response := response.(type) {
	case adminservice.PutGearConfig200JSONResponse:
		return nil
	case adminservice.PutGearConfig400JSONResponse:
		return responseError(400, "PUT_GEAR_CONFIG_FAILED", "failed to put gear config", response)
	case adminservice.PutGearConfig404JSONResponse:
		return responseError(404, "PUT_GEAR_CONFIG_FAILED", "failed to put gear config", response)
	default:
		return unexpectedResponse("PutGearConfig", response)
	}
}

func resourceFromGearConfig(name string, item apitypes.Configuration) (apitypes.Resource, error) {
	return marshalResource(apitypes.GearConfigResource{
		ApiVersion: apitypes.ResourceAPIVersionGizclawAdminv1alpha1,
		Kind:       apitypes.GearConfigResourceKind(apitypes.ResourceKindGearConfig),
		Metadata:   apitypes.ResourceMetadata{Name: name},
		Spec:       item,
	})
}
