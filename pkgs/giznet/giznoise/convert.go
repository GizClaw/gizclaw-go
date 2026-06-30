package giznoise

import (
	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/giznet/internal/core"
)

func fromCorePeerState(state core.PeerState) giznet.PeerState {
	switch state {
	case core.PeerStateNew:
		return giznet.PeerStateNew
	case core.PeerStateConnecting:
		return giznet.PeerStateConnecting
	case core.PeerStateEstablished:
		return giznet.PeerStateEstablished
	case core.PeerStateFailed:
		return giznet.PeerStateFailed
	case core.PeerStateOffline:
		return giznet.PeerStateOffline
	default:
		return giznet.PeerStateFailed
	}
}

func fromCorePeerInfo(info *core.PeerInfo) *giznet.PeerInfo {
	if info == nil {
		return nil
	}
	return &giznet.PeerInfo{
		PublicKey: fromNoisePublicKey(info.PublicKey),
		Endpoint:  info.Endpoint,
		State:     fromCorePeerState(info.State),
		RxBytes:   info.RxBytes,
		TxBytes:   info.TxBytes,
		LastSeen:  info.LastSeen,
	}
}
