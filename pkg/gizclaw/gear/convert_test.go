package gear

import (
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/serverpublic"
)

func TestConvertHelpers(t *testing.T) {
	now := time.Unix(1_700_600_000, 0).UTC()
	autoRegistered := true
	stable := gearservice.GearFirmwareChannel("stable")
	deviceName := "convert-device"
	gear := gearservice.Gear{
		PublicKey:      "peer-convert",
		Role:           gearservice.GearRolePeer,
		Status:         gearservice.GearStatusActive,
		AutoRegistered: &autoRegistered,
		CreatedAt:      now,
		UpdatedAt:      now,
		Configuration: gearservice.Configuration{
			Firmware: &gearservice.FirmwareConfig{Channel: &stable},
		},
		Device: gearservice.DeviceInfo{
			Name: &deviceName,
		},
	}

	registration := toGearRegistration(gear)
	if registration.PublicKey != gear.PublicKey || registration.Role != gear.Role {
		t.Fatalf("toGearRegistration = %+v", registration)
	}

	publicRegistration := toPublicRegistration(gear)
	if publicRegistration.PublicKey != gear.PublicKey || publicRegistration.Role != serverpublic.GearRole(gear.Role) {
		t.Fatalf("toPublicRegistration = %+v", publicRegistration)
	}

	cfg, err := toPublicConfiguration(gear.Configuration)
	if err != nil {
		t.Fatalf("toPublicConfiguration error: %v", err)
	}
	if cfg.Firmware == nil || cfg.Firmware.Channel == nil || *cfg.Firmware.Channel != serverpublic.GearFirmwareChannel(stable) {
		t.Fatalf("toPublicConfiguration = %+v", cfg)
	}

	publicRuntime := toPublicRuntime(gearservice.Runtime{Online: true, LastSeenAt: now})
	if !publicRuntime.Online || !publicRuntime.LastSeenAt.Equal(now) {
		t.Fatalf("toPublicRuntime = %+v", publicRuntime)
	}

	result, err := toPublicRegistrationResult(gearservice.RegistrationResult{Gear: gear, Registration: registration})
	if err != nil {
		t.Fatalf("toPublicRegistrationResult error: %v", err)
	}
	if result.Registration.PublicKey != gear.PublicKey || result.Gear.PublicKey != gear.PublicKey {
		t.Fatalf("toPublicRegistrationResult = %+v", result)
	}

	adminRegistrations := toAdminRegistrationList([]gearservice.Gear{gear})
	if len(adminRegistrations.Items) != 1 || adminRegistrations.Items[0].PublicKey != gear.PublicKey {
		t.Fatalf("toAdminRegistrationList = %+v", adminRegistrations)
	}

	adminGear, err := toAdminGear(gear)
	if err != nil {
		t.Fatalf("toAdminGear error: %v", err)
	}
	if adminGear.PublicKey != gear.PublicKey || adminGear.Configuration.Firmware == nil {
		t.Fatalf("toAdminGear = %+v", adminGear)
	}

	adminOTA, err := toAdminOTASummary(gearservice.OTASummary{
		Depot:          "demo",
		Channel:        "stable",
		FirmwareSemver: "1.0.0",
		Files: []gearservice.DepotFile{{
			Path:   "bundles/fw.bin",
			Sha256: "sha256",
			Md5:    "md5",
		}},
	})
	if err != nil {
		t.Fatalf("toAdminOTASummary error: %v", err)
	}
	if adminOTA.Depot != "demo" || len(adminOTA.Files) != 1 {
		t.Fatalf("toAdminOTASummary = %+v", adminOTA)
	}

	gearRegistrations := toGearRegistrationList([]gearservice.Gear{gear})
	if len(gearRegistrations.Items) != 1 || gearRegistrations.Items[0].PublicKey != gear.PublicKey {
		t.Fatalf("toGearRegistrationList = %+v", gearRegistrations)
	}

	convertedDevice, err := toGearDeviceInfo(serverpublic.DeviceInfo{
		Name: gear.Device.Name,
		Sn:   gear.Device.Sn,
	})
	if err != nil {
		t.Fatalf("toGearDeviceInfo error: %v", err)
	}
	if convertedDevice.Name == nil || *convertedDevice.Name != *gear.Device.Name {
		t.Fatalf("toGearDeviceInfo = %+v", convertedDevice)
	}

	adminRuntime := toAdminRuntime(gearservice.Runtime{Online: true, LastSeenAt: now})
	if !adminRuntime.Online || !adminRuntime.LastSeenAt.Equal(now) {
		t.Fatalf("toAdminRuntime = %+v", adminRuntime)
	}

	adminRefresh, err := toAdminRefreshResult(gearservice.RefreshResult{
		Gear:          gear,
		UpdatedFields: &[]string{"device.name"},
	})
	if err != nil {
		t.Fatalf("toAdminRefreshResult error: %v", err)
	}
	if adminRefresh.Gear.PublicKey != gear.PublicKey || adminRefresh.Gear.Role != adminservice.GearRolePeer {
		t.Fatalf("toAdminRefreshResult = %+v", adminRefresh)
	}
}
