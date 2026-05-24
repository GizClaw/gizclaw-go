package peer

import (
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

func TestValidateGear(t *testing.T) {
	roleErr := validateGear(apitypes.Gear{
		PublicKey: giznet.PublicKey{1}.String(),
		Role:      apitypes.GearRole("bad"),
		Status:    apitypes.GearStatusActive,
	})
	if roleErr == nil {
		t.Fatal("validateGear should fail on invalid role")
	}

	statusErr := validateGear(apitypes.Gear{
		PublicKey: giznet.PublicKey{1}.String(),
		Role:      apitypes.GearRoleServer,
		Status:    apitypes.GearStatus("bad"),
	})
	if statusErr == nil {
		t.Fatal("validateGear should fail on invalid status")
	}
}

func TestValidateConfiguration(t *testing.T) {
	view := "under-12"
	if err := validateConfiguration(apitypes.Configuration{View: &view}); err != nil {
		t.Fatalf("validateConfiguration err = %v", err)
	}
}
