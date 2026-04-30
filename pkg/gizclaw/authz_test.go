package gizclaw

import (
	"context"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gear"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

func TestGearsSecurityPolicyAllowsProtectedServicesForActiveGear(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	service := &gear.Server{Store: mustBadgerInMemory(t, nil)}
	if _, err := service.SaveGear(context.Background(), apitypes.Gear{
		PublicKey:     keyPair.Public.String(),
		Role:          apitypes.GearRoleUnspecified,
		Status:        apitypes.GearStatusActive,
		Device:        apitypes.DeviceInfo{},
		Configuration: apitypes.Configuration{},
	}); err != nil {
		t.Fatalf("SaveGear error = %v", err)
	}

	policy := GearsSecurityPolicy{Gears: service}
	if !policy.AllowPeerService(keyPair.Public, ServiceAdmin) {
		t.Fatal("active gear should allow admin service")
	}
	if !policy.AllowPeerService(keyPair.Public, ServiceGear) {
		t.Fatal("active gear should allow gear service")
	}
	if !policy.AllowPeerService(keyPair.Public, ServiceServerPublic) {
		t.Fatal("active gear should allow server public service")
	}
	if policy.AllowPeerService(keyPair.Public, ServicePeerPublic) {
		t.Fatal("active gear should not allow peer public service")
	}
	if policy.AllowPeerService(keyPair.Public, 0xffff) {
		t.Fatal("active gear should not allow unknown service")
	}
}

func TestGearsSecurityPolicyAllowsPublicServicesWithoutGearLookup(t *testing.T) {
	policy := GearsSecurityPolicy{}
	if !policy.AllowPeerService(giznet.PublicKey{}, ServiceRPC) {
		t.Fatal("policy should allow rpc service")
	}
	if !policy.AllowPeerService(giznet.PublicKey{}, ServiceServerPublic) {
		t.Fatal("policy should allow server public service")
	}
}

func TestGearsSecurityPolicyAllowsGearServiceForUnknownGear(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	policy := GearsSecurityPolicy{Gears: &gear.Server{Store: mustBadgerInMemory(t, nil)}}
	if !policy.AllowPeerService(keyPair.Public, ServiceGear) {
		t.Fatal("unknown gear should allow gear service for registration")
	}
	if policy.AllowPeerService(keyPair.Public, ServiceAdmin) {
		t.Fatal("unknown gear should not allow admin service")
	}
}

func TestGearsSecurityPolicyDeniesProtectedServicesForBlockedGear(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	service := &gear.Server{Store: mustBadgerInMemory(t, nil)}
	ctx := context.Background()
	if _, err := service.SaveGear(ctx, apitypes.Gear{
		PublicKey:     keyPair.Public.String(),
		Role:          apitypes.GearRoleUnspecified,
		Status:        apitypes.GearStatusBlocked,
		Device:        apitypes.DeviceInfo{},
		Configuration: apitypes.Configuration{},
	}); err != nil {
		t.Fatalf("SaveGear error = %v", err)
	}

	policy := GearsSecurityPolicy{Gears: service}
	if policy.AllowPeerService(keyPair.Public, ServiceAdmin) {
		t.Fatal("blocked gear should not allow admin service")
	}
	if policy.AllowPeerService(keyPair.Public, ServiceGear) {
		t.Fatal("blocked gear should not allow gear service")
	}
}
