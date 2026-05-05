package gizclaw

import (
	"context"
	"errors"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	gearpkg "github.com/GizClaw/gizclaw-go/pkg/gizclaw/gear"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

type ServerSecurityPolicy Server

var _ giznet.SecurityPolicy = (*ServerSecurityPolicy)(nil)

func (p *ServerSecurityPolicy) AllowPeer(giznet.PublicKey) bool {
	return p != nil
}

func (p *ServerSecurityPolicy) AllowService(publicKey giznet.PublicKey, service uint64) bool {
	if p == nil {
		return false
	}
	server := (*Server)(p)
	manager := server.manager
	adminPublicKey := server.AdminPublicKey
	if service == ServiceAdmin && adminPublicKey != "" {
		key, err := giznet.KeyFromHex(adminPublicKey)
		if err == nil && key == publicKey {
			return true
		}
	}
	if manager == nil {
		return false
	}
	switch service {
	case ServiceRPC, ServiceServerPublic:
		return true
	}
	if manager.Gears == nil {
		return false
	}
	switch service {
	case ServiceGear:
		gear, err := manager.Gears.LoadGear(context.Background(), publicKey.String())
		if errors.Is(err, gearpkg.ErrGearNotFound) {
			return true
		}
		if err != nil {
			return false
		}
		return gear.Status == apitypes.GearStatusActive
	case ServiceAdmin:
		gear, err := manager.Gears.LoadGear(context.Background(), publicKey.String())
		if err != nil {
			return false
		}
		return gear.Status == apitypes.GearStatusActive && gear.Role == apitypes.GearRoleAdmin
	default:
		return false
	}
}
