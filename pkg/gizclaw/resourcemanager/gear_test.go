package resourcemanager

import (
	"context"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func TestApplyPeerConfigUpdatesResource(t *testing.T) {
	peers := newFakePeers()
	peers.configs["gear-key"] = apitypes.Configuration{}
	manager := New(Services{Peers: peers})

	result, err := manager.Apply(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "PeerConfig",
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
	if peers.putCount != 1 {
		t.Fatalf("putCount = %d, want 1", peers.putCount)
	}
	if peers.configs["gear-key"].Firmware == nil || peers.configs["gear-key"].Firmware.Channel == nil {
		t.Fatal("stored firmware channel is nil, want stable")
	}
}

func TestGetPeerConfigReturnsResource(t *testing.T) {
	channel := apitypes.GearFirmwareChannel("stable")
	peers := newFakePeers()
	peers.configs["gear-key"] = apitypes.Configuration{
		Firmware: &apitypes.FirmwareConfig{Channel: &channel},
	}
	manager := New(Services{Peers: peers})

	resource, err := manager.Get(context.Background(), apitypes.ResourceKindPeerConfig, "gear-key")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	config, err := resource.AsPeerConfigResource()
	if err != nil {
		t.Fatalf("AsPeerConfigResource returned error: %v", err)
	}
	if config.Metadata.Name != "gear-key" {
		t.Fatalf("metadata.name = %q, want gear-key", config.Metadata.Name)
	}
	if config.Spec.Firmware == nil || config.Spec.Firmware.Channel == nil || *config.Spec.Firmware.Channel != "stable" {
		t.Fatalf("firmware channel = %#v, want stable", config.Spec.Firmware)
	}
}

func TestPutPeerConfigWritesResource(t *testing.T) {
	peers := newFakePeers()
	peers.configs["gear-key"] = apitypes.Configuration{}
	manager := New(Services{Peers: peers})

	_, err := manager.Put(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "PeerConfig",
		"metadata": {"name": "gear-key"},
		"spec": {
			"firmware": {"channel": "testing"}
		}
	}`))
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if peers.putCount != 1 {
		t.Fatalf("putCount = %d, want 1", peers.putCount)
	}
}

func TestApplyPeerConfigUnchangedSkipsPut(t *testing.T) {
	channel := apitypes.GearFirmwareChannel("stable")
	peers := newFakePeers()
	peers.configs["gear-key"] = apitypes.Configuration{
		Firmware: &apitypes.FirmwareConfig{Channel: &channel},
	}
	manager := New(Services{Peers: peers})

	result, err := manager.Apply(context.Background(), mustResource(t, `{
		"apiVersion": "gizclaw.admin/v1alpha1",
		"kind": "PeerConfig",
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
	if peers.putCount != 0 {
		t.Fatalf("putCount = %d, want 0", peers.putCount)
	}
}

func TestGearServiceErrorResponses(t *testing.T) {
	peers := newFakePeers()
	manager := New(Services{Peers: peers})

	_, err := manager.getPeerConfig(context.Background(), "missing")
	assertResourceError(t, err, 404, "GEAR_NOT_FOUND")

	peers.configs["gear-key"] = apitypes.Configuration{}
	peers.putStatus = 400
	err = manager.putPeerConfig(context.Background(), "gear-key", apitypes.Configuration{})
	assertResourceError(t, err, 400, "INVALID_PARAMS")

	peers.putStatus = 404
	err = manager.putPeerConfig(context.Background(), "gear-key", apitypes.Configuration{})
	assertResourceError(t, err, 404, "GEAR_NOT_FOUND")
}

type fakePeers struct {
	configs   map[string]apitypes.Configuration
	putCount  int
	putStatus int
}

func newFakePeers() *fakePeers {
	return &fakePeers{configs: map[string]apitypes.Configuration{}}
}

func (f *fakePeers) ListPeers(context.Context, adminservice.ListPeersRequestObject) (adminservice.ListPeersResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) ListPeersByCertification(context.Context, adminservice.ListPeersByCertificationRequestObject) (adminservice.ListPeersByCertificationResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) ListPeersByFirmware(context.Context, adminservice.ListPeersByFirmwareRequestObject) (adminservice.ListPeersByFirmwareResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) ResolvePeerByIMEI(context.Context, adminservice.ResolvePeerByIMEIRequestObject) (adminservice.ResolvePeerByIMEIResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) ListPeersByLabel(context.Context, adminservice.ListPeersByLabelRequestObject) (adminservice.ListPeersByLabelResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) ResolvePeerBySN(context.Context, adminservice.ResolvePeerBySNRequestObject) (adminservice.ResolvePeerBySNResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) DeletePeer(context.Context, adminservice.DeletePeerRequestObject) (adminservice.DeletePeerResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) GetPeer(context.Context, adminservice.GetPeerRequestObject) (adminservice.GetPeerResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) GetPeerConfig(_ context.Context, request adminservice.GetPeerConfigRequestObject) (adminservice.GetPeerConfigResponseObject, error) {
	config, ok := f.configs[string(request.PublicKey)]
	if !ok {
		return adminservice.GetPeerConfig404JSONResponse(apitypes.NewErrorResponse("GEAR_NOT_FOUND", "not found")), nil
	}
	return adminservice.GetPeerConfig200JSONResponse(config), nil
}

func (f *fakePeers) PutPeerConfig(_ context.Context, request adminservice.PutPeerConfigRequestObject) (adminservice.PutPeerConfigResponseObject, error) {
	switch f.putStatus {
	case 400:
		return adminservice.PutPeerConfig400JSONResponse(apitypes.NewErrorResponse("INVALID_PARAMS", "invalid")), nil
	case 404:
		return adminservice.PutPeerConfig404JSONResponse(apitypes.NewErrorResponse("GEAR_NOT_FOUND", "not found")), nil
	}
	f.putCount++
	f.configs[string(request.PublicKey)] = *request.Body
	return adminservice.PutPeerConfig200JSONResponse(*request.Body), nil
}

func (f *fakePeers) GetPeerInfo(context.Context, adminservice.GetPeerInfoRequestObject) (adminservice.GetPeerInfoResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) GetPeerRuntime(context.Context, adminservice.GetPeerRuntimeRequestObject) (adminservice.GetPeerRuntimeResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) ApprovePeer(context.Context, adminservice.ApprovePeerRequestObject) (adminservice.ApprovePeerResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) BlockPeer(context.Context, adminservice.BlockPeerRequestObject) (adminservice.BlockPeerResponseObject, error) {
	return nil, nil
}

func (f *fakePeers) RefreshPeer(context.Context, adminservice.RefreshPeerRequestObject) (adminservice.RefreshPeerResponseObject, error) {
	return nil, nil
}
