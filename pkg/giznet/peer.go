package giznet

import "github.com/GizClaw/gizclaw-go/pkg/giznet/internal/core"

type PeerState = core.PeerState
type PeerEvent = core.PeerEvent

const (
	PeerStateNew         = core.PeerStateNew
	PeerStateConnecting  = core.PeerStateConnecting
	PeerStateEstablished = core.PeerStateEstablished
	PeerStateFailed      = core.PeerStateFailed
	PeerStateOffline     = core.PeerStateOffline
)
