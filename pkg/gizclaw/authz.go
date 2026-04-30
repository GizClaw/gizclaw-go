package gizclaw

import (
	"context"
	"errors"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	gearpkg "github.com/GizClaw/gizclaw-go/pkg/gizclaw/gear"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

type GearsSecurityPolicy struct {
	Gears *gearpkg.Server
}

var _ SecurityPolicy = GearsSecurityPolicy{}

func (p GearsSecurityPolicy) AllowPeerService(publicKey giznet.PublicKey, service uint64) bool {
	switch service {
	case ServiceRPC, ServiceServerPublic:
		return true
	}
	if p.Gears == nil {
		return false
	}
	switch service {
	case ServiceGear:
		gear, err := p.Gears.LoadGear(context.Background(), publicKey.String())
		if errors.Is(err, gearpkg.ErrGearNotFound) {
			return true
		}
		if err != nil {
			return false
		}
		return gear.Status == apitypes.GearStatusActive
	case ServiceAdmin:
		gear, err := p.Gears.LoadGear(context.Background(), publicKey.String())
		if err != nil {
			return false
		}
		return gear.Status == apitypes.GearStatusActive
	default:
		return false
	}
}
