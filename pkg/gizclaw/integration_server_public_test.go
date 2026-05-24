package gizclaw_test

import (
	"context"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
)

func TestIntegrationServerPublicRegisterAndReadBack(t *testing.T) {
	ts := startTestServer(t)
	device := newTestClient(t, ts)
	if device.PeerConn() == nil {
		t.Fatal("PeerConn returned nil")
	}

	result, err := register(context.Background(), device, gearservice.RegistrationRequest{
		Device: apitypes.DeviceInfo{
			Name: strPtr("demo-device"),
			Sn:   strPtr("sn-001"),
			Hardware: &apitypes.HardwareInfo{
				Manufacturer: strPtr("Acme"),
				Model:        strPtr("M1"),
			},
		},
	})
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if result.Gear.PublicKey == "" {
		t.Fatal("empty public key after register")
	}

	info, err := getInfo(context.Background(), device)
	if err != nil {
		t.Fatalf("GetInfo error: %v", err)
	}
	if info.Name == nil || *info.Name != "demo-device" {
		t.Fatalf("device name = %+v", info.Name)
	}

	registration, err := getRegistration(context.Background(), device)
	if err != nil {
		t.Fatalf("GetRegistration error: %v", err)
	}
	if registration.Role != apitypes.GearRoleUnspecified {
		t.Fatalf("role = %q", registration.Role)
	}

	if _, err := getServerInfo(context.Background(), device); err != nil {
		t.Fatalf("GetServerInfo error: %v", err)
	}
	if _, err := putInfo(context.Background(), device, apitypes.DeviceInfo{
		Name: strPtr("demo-device-2"),
		Sn:   strPtr("sn-002"),
	}); err != nil {
		t.Fatalf("PutInfo error: %v", err)
	}
	if _, err := getRuntime(context.Background(), device); err != nil {
		t.Fatalf("GetRuntime error: %v", err)
	}
	if _, err := getConfig(context.Background(), device); err != nil {
		t.Fatalf("GetConfig error: %v", err)
	}

}
