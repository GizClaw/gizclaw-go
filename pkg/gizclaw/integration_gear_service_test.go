package gizclaw_test

import (
	"context"
	"testing"

	apitypes "github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/serverpublic"
)

func TestIntegrationGearServiceLifecycle(t *testing.T) {
	ts := startTestServer(t)

	admin := newTestClient(t, ts)
	adminResult, err := register(context.Background(), admin, serverpublic.RegistrationRequest{
		Device:            apitypes.DeviceInfo{Name: strPtr("admin")},
		RegistrationToken: strPtr("admin_default"),
	})
	if err != nil {
		t.Fatalf("admin register error: %v", err)
	}

	device := newTestClient(t, ts)
	deviceResult, err := register(context.Background(), device, serverpublic.RegistrationRequest{
		Device: apitypes.DeviceInfo{
			Name: strPtr("device"),
			Sn:   strPtr("sn/1"),
			Hardware: &apitypes.HardwareInfo{
				Depot: strPtr("demo-main"),
				Imeis: &[]apitypes.GearIMEI{{Name: strPtr("main"), Tac: "12345678", Serial: "0000001"}},
				Labels: &[]apitypes.GearLabel{{
					Key:   "batch",
					Value: "cn/east",
				}},
			},
		},
		RegistrationToken: strPtr("device_default"),
	})
	if err != nil {
		t.Fatalf("device register error: %v", err)
	}

	items, err := listGears(context.Background(), admin)
	if err != nil {
		t.Fatalf("ListGears error: %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("ListGears returned %d items", len(items))
	}

	if _, err := approveGear(context.Background(), admin, deviceResult.Gear.PublicKey, apitypes.GearRoleDevice); err != nil {
		t.Fatalf("ApproveGear error: %v", err)
	}
	if _, err := getGear(context.Background(), admin, deviceResult.Gear.PublicKey); err != nil {
		t.Fatalf("GetGear error: %v", err)
	}
	if publicKey, err := resolveGearBySN(context.Background(), admin, "sn/1"); err != nil || publicKey != deviceResult.Gear.PublicKey {
		t.Fatalf("ResolveGearBySN = %q, %v", publicKey, err)
	}
	if publicKey, err := resolveGearByIMEI(context.Background(), admin, "12345678", "0000001"); err != nil || publicKey != deviceResult.Gear.PublicKey {
		t.Fatalf("ResolveGearByIMEI = %q, %v", publicKey, err)
	}
	if _, err := putGearConfig(context.Background(), admin, deviceResult.Gear.PublicKey, apitypes.Configuration{
		Certifications: &[]apitypes.GearCertification{{
			Type:      apitypes.GearCertificationType("certification"),
			Authority: apitypes.GearCertificationAuthority("ce"),
			Id:        "ce/001",
		}},
		Firmware: &apitypes.FirmwareConfig{Channel: func() *apitypes.GearFirmwareChannel {
			ch := apitypes.GearFirmwareChannel("stable")
			return &ch
		}()},
	}); err != nil {
		t.Fatalf("PutGearConfig error: %v", err)
	}
	if _, err := getGearInfo(context.Background(), admin, deviceResult.Gear.PublicKey); err != nil {
		t.Fatalf("GetGearInfo error: %v", err)
	}
	if _, err := getGearConfig(context.Background(), admin, deviceResult.Gear.PublicKey); err != nil {
		t.Fatalf("GetGearConfig error: %v", err)
	}
	if _, err := getGearRuntime(context.Background(), admin, deviceResult.Gear.PublicKey); err != nil {
		t.Fatalf("GetGearRuntime error: %v", err)
	}
	if _, err := listGearsByLabel(context.Background(), admin, "batch", "cn/east"); err != nil {
		t.Fatalf("ListGearsByLabel error: %v", err)
	}
	if _, err := listGearsByCertification(context.Background(), admin, apitypes.GearCertificationType("certification"), apitypes.GearCertificationAuthority("ce"), "ce/001"); err != nil {
		t.Fatalf("ListGearsByCertification error: %v", err)
	}
	if _, err := listGearsByFirmware(context.Background(), admin, "demo-main", apitypes.GearFirmwareChannel("stable")); err != nil {
		t.Fatalf("ListGearsByFirmware error: %v", err)
	}
	if _, err := blockGear(context.Background(), admin, deviceResult.Gear.PublicKey); err != nil {
		t.Fatalf("BlockGear error: %v", err)
	}
	if _, err := deleteGear(context.Background(), admin, adminResult.Gear.PublicKey); err != nil {
		t.Fatalf("DeleteGear error: %v", err)
	}
}
