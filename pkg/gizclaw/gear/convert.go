package gear

import (
	"encoding/json"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/serverpublic"
)

func convertViaJSON[T any](in any) (T, error) {
	var out T
	data, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

func toAdminRegistrationList(items []gearservice.Gear) adminservice.RegistrationList {
	out := make([]adminservice.Registration, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminRegistration(item))
	}
	return adminservice.RegistrationList{Items: out}
}

func toAdminRegistration(gear gearservice.Gear) adminservice.Registration {
	return adminservice.Registration{
		ApprovedAt:     gear.ApprovedAt,
		AutoRegistered: gear.AutoRegistered,
		CreatedAt:      gear.CreatedAt,
		PublicKey:      gear.PublicKey,
		Role:           adminservice.GearRole(gear.Role),
		Status:         adminservice.GearStatus(gear.Status),
		UpdatedAt:      gear.UpdatedAt,
	}
}

func toAdminConfiguration(in gearservice.Configuration) (adminservice.Configuration, error) {
	return convertViaJSON[adminservice.Configuration](in)
}

func toAdminDeviceInfo(in gearservice.DeviceInfo) (adminservice.DeviceInfo, error) {
	return convertViaJSON[adminservice.DeviceInfo](in)
}

func toAdminRuntime(in gearservice.Runtime) adminservice.Runtime {
	return adminservice.Runtime{
		LastAddr:   in.LastAddr,
		LastSeenAt: in.LastSeenAt,
		Online:     in.Online,
	}
}

func toAdminGear(in gearservice.Gear) (adminservice.Gear, error) {
	cfg, err := toAdminConfiguration(in.Configuration)
	if err != nil {
		return adminservice.Gear{}, err
	}
	info, err := toAdminDeviceInfo(in.Device)
	if err != nil {
		return adminservice.Gear{}, err
	}
	return adminservice.Gear{
		ApprovedAt:     in.ApprovedAt,
		AutoRegistered: in.AutoRegistered,
		Configuration:  cfg,
		CreatedAt:      in.CreatedAt,
		Device:         info,
		PublicKey:      in.PublicKey,
		Role:           adminservice.GearRole(in.Role),
		Status:         adminservice.GearStatus(in.Status),
		UpdatedAt:      in.UpdatedAt,
	}, nil
}

func toAdminRefreshResult(in gearservice.RefreshResult) (adminservice.RefreshResult, error) {
	return convertViaJSON[adminservice.RefreshResult](in)
}

func toAdminOTASummary(in gearservice.OTASummary) (adminservice.OTASummary, error) {
	return convertViaJSON[adminservice.OTASummary](in)
}

func toGearRegistrationList(items []gearservice.Gear) gearservice.RegistrationList {
	out := make([]gearservice.Registration, 0, len(items))
	for _, item := range items {
		out = append(out, toGearRegistration(item))
	}
	return gearservice.RegistrationList{Items: out}
}

func toGearRegistration(gear gearservice.Gear) gearservice.Registration {
	return gearservice.Registration{
		ApprovedAt:     gear.ApprovedAt,
		AutoRegistered: gear.AutoRegistered,
		CreatedAt:      gear.CreatedAt,
		PublicKey:      gear.PublicKey,
		Role:           gear.Role,
		Status:         gear.Status,
		UpdatedAt:      gear.UpdatedAt,
	}
}

func toPublicConfiguration(in gearservice.Configuration) (serverpublic.Configuration, error) {
	return convertViaJSON[serverpublic.Configuration](in)
}

func toPublicDeviceInfo(in gearservice.DeviceInfo) (serverpublic.DeviceInfo, error) {
	return convertViaJSON[serverpublic.DeviceInfo](in)
}

func toGearDeviceInfo(in serverpublic.DeviceInfo) (gearservice.DeviceInfo, error) {
	return convertViaJSON[gearservice.DeviceInfo](in)
}

func toGearRegistrationRequest(in serverpublic.RegistrationRequest) (gearservice.RegistrationRequest, error) {
	return convertViaJSON[gearservice.RegistrationRequest](in)
}

func toPublicRegistration(gear gearservice.Gear) serverpublic.Registration {
	return serverpublic.Registration{
		ApprovedAt:     gear.ApprovedAt,
		AutoRegistered: gear.AutoRegistered,
		CreatedAt:      gear.CreatedAt,
		PublicKey:      gear.PublicKey,
		Role:           serverpublic.GearRole(gear.Role),
		Status:         serverpublic.GearStatus(gear.Status),
		UpdatedAt:      gear.UpdatedAt,
	}
}

func toPublicRuntime(in gearservice.Runtime) serverpublic.Runtime {
	return serverpublic.Runtime{
		LastAddr:   in.LastAddr,
		LastSeenAt: in.LastSeenAt,
		Online:     in.Online,
	}
}

func toPublicGear(in gearservice.Gear) (serverpublic.Gear, error) {
	cfg, err := toPublicConfiguration(in.Configuration)
	if err != nil {
		return serverpublic.Gear{}, err
	}
	info, err := toPublicDeviceInfo(in.Device)
	if err != nil {
		return serverpublic.Gear{}, err
	}
	return serverpublic.Gear{
		ApprovedAt:     in.ApprovedAt,
		AutoRegistered: in.AutoRegistered,
		Configuration:  cfg,
		CreatedAt:      in.CreatedAt,
		Device:         info,
		PublicKey:      in.PublicKey,
		Role:           serverpublic.GearRole(in.Role),
		Status:         serverpublic.GearStatus(in.Status),
		UpdatedAt:      in.UpdatedAt,
	}, nil
}

func toPublicRegistrationResult(in gearservice.RegistrationResult) (serverpublic.RegistrationResult, error) {
	gear, err := toPublicGear(in.Gear)
	if err != nil {
		return serverpublic.RegistrationResult{}, err
	}
	return serverpublic.RegistrationResult{
		Gear:         gear,
		Registration: toPublicRegistration(in.Gear),
	}, nil
}
