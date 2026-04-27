package resourcemanager

import (
	"context"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func TestApplyGearConfigUpdatesResource(t *testing.T) {
	gears := newFakeGears()
	gears.configs["gear-key"] = apitypes.Configuration{}
	manager := New(Services{Gears: gears})

	result, err := manager.Apply(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "GearConfig",
		"metadata": {"name": "gear-key"},
		"spec": {
			"firmware": {"channel": "stable"}
		}
	}`))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Action != apitypes.ApplyActionUpdated {
		t.Fatalf("action = %q, want updated", result.Action)
	}
	if gears.putCount != 1 {
		t.Fatalf("putCount = %d, want 1", gears.putCount)
	}
	if gears.configs["gear-key"].Firmware == nil || gears.configs["gear-key"].Firmware.Channel == nil {
		t.Fatal("stored firmware channel is nil, want stable")
	}
}

func TestGetGearConfigReturnsResource(t *testing.T) {
	channel := apitypes.GearFirmwareChannel("stable")
	gears := newFakeGears()
	gears.configs["gear-key"] = apitypes.Configuration{
		Firmware: &apitypes.FirmwareConfig{Channel: &channel},
	}
	manager := New(Services{Gears: gears})

	resource, err := manager.Get(context.Background(), apitypes.ResourceKindGearConfig, "gear-key")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	config, err := resource.AsGearConfigResource()
	if err != nil {
		t.Fatalf("AsGearConfigResource returned error: %v", err)
	}
	if config.Metadata.Name != "gear-key" {
		t.Fatalf("metadata.name = %q, want gear-key", config.Metadata.Name)
	}
	if config.Spec.Firmware == nil || config.Spec.Firmware.Channel == nil || *config.Spec.Firmware.Channel != "stable" {
		t.Fatalf("firmware channel = %#v, want stable", config.Spec.Firmware)
	}
}

func TestPutGearConfigWritesResource(t *testing.T) {
	gears := newFakeGears()
	gears.configs["gear-key"] = apitypes.Configuration{}
	manager := New(Services{Gears: gears})

	_, err := manager.Put(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "GearConfig",
		"metadata": {"name": "gear-key"},
		"spec": {
			"firmware": {"channel": "testing"}
		}
	}`))
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if gears.putCount != 1 {
		t.Fatalf("putCount = %d, want 1", gears.putCount)
	}
}

func TestApplyGearConfigUnchangedSkipsPut(t *testing.T) {
	channel := apitypes.GearFirmwareChannel("stable")
	gears := newFakeGears()
	gears.configs["gear-key"] = apitypes.Configuration{
		Firmware: &apitypes.FirmwareConfig{Channel: &channel},
	}
	manager := New(Services{Gears: gears})

	result, err := manager.Apply(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "GearConfig",
		"metadata": {"name": "gear-key"},
		"spec": {
			"firmware": {"channel": "stable"}
		}
	}`))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.Action != apitypes.ApplyActionUnchanged {
		t.Fatalf("action = %q, want unchanged", result.Action)
	}
	if gears.putCount != 0 {
		t.Fatalf("putCount = %d, want 0", gears.putCount)
	}
}

func TestGearServiceErrorResponses(t *testing.T) {
	gears := newFakeGears()
	manager := New(Services{Gears: gears})

	_, err := manager.getGearConfig(context.Background(), "missing")
	assertResourceError(t, err, 404, "GEAR_NOT_FOUND")

	gears.configs["gear-key"] = apitypes.Configuration{}
	gears.putStatus = 400
	err = manager.putGearConfig(context.Background(), "gear-key", apitypes.Configuration{})
	assertResourceError(t, err, 400, "INVALID_PARAMS")

	gears.putStatus = 404
	err = manager.putGearConfig(context.Background(), "gear-key", apitypes.Configuration{})
	assertResourceError(t, err, 404, "GEAR_NOT_FOUND")
}

type fakeGears struct {
	configs   map[string]apitypes.Configuration
	putCount  int
	putStatus int
}

func newFakeGears() *fakeGears {
	return &fakeGears{configs: map[string]apitypes.Configuration{}}
}

func (f *fakeGears) ListGears(context.Context, adminservice.ListGearsRequestObject) (adminservice.ListGearsResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) ListByCertification(context.Context, adminservice.ListByCertificationRequestObject) (adminservice.ListByCertificationResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) ListByFirmware(context.Context, adminservice.ListByFirmwareRequestObject) (adminservice.ListByFirmwareResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) ResolveByIMEI(context.Context, adminservice.ResolveByIMEIRequestObject) (adminservice.ResolveByIMEIResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) ListByLabel(context.Context, adminservice.ListByLabelRequestObject) (adminservice.ListByLabelResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) ResolveBySN(context.Context, adminservice.ResolveBySNRequestObject) (adminservice.ResolveBySNResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) DeleteGear(context.Context, adminservice.DeleteGearRequestObject) (adminservice.DeleteGearResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) GetGear(context.Context, adminservice.GetGearRequestObject) (adminservice.GetGearResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) GetGearConfig(_ context.Context, request adminservice.GetGearConfigRequestObject) (adminservice.GetGearConfigResponseObject, error) {
	config, ok := f.configs[string(request.PublicKey)]
	if !ok {
		return adminservice.GetGearConfig404JSONResponse(apitypes.NewErrorResponse("GEAR_NOT_FOUND", "not found")), nil
	}
	return adminservice.GetGearConfig200JSONResponse(config), nil
}

func (f *fakeGears) PutGearConfig(_ context.Context, request adminservice.PutGearConfigRequestObject) (adminservice.PutGearConfigResponseObject, error) {
	switch f.putStatus {
	case 400:
		return adminservice.PutGearConfig400JSONResponse(apitypes.NewErrorResponse("INVALID_PARAMS", "invalid")), nil
	case 404:
		return adminservice.PutGearConfig404JSONResponse(apitypes.NewErrorResponse("GEAR_NOT_FOUND", "not found")), nil
	}
	f.putCount++
	f.configs[string(request.PublicKey)] = *request.Body
	return adminservice.PutGearConfig200JSONResponse(*request.Body), nil
}

func (f *fakeGears) GetGearInfo(context.Context, adminservice.GetGearInfoRequestObject) (adminservice.GetGearInfoResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) GetGearRuntime(context.Context, adminservice.GetGearRuntimeRequestObject) (adminservice.GetGearRuntimeResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) ApproveGear(context.Context, adminservice.ApproveGearRequestObject) (adminservice.ApproveGearResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) BlockGear(context.Context, adminservice.BlockGearRequestObject) (adminservice.BlockGearResponseObject, error) {
	return nil, nil
}

func (f *fakeGears) RefreshGear(context.Context, adminservice.RefreshGearRequestObject) (adminservice.RefreshGearResponseObject, error) {
	return nil, nil
}
