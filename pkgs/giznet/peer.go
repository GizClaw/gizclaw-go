package giznet

import (
	"net"
	"time"
)

type PeerState int

const (
	PeerStateNew PeerState = iota
	PeerStateConnecting
	PeerStateEstablished
	PeerStateFailed
	PeerStateOffline
)

func (s PeerState) String() string {
	switch s {
	case PeerStateNew:
		return "new"
	case PeerStateConnecting:
		return "connecting"
	case PeerStateEstablished:
		return "established"
	case PeerStateFailed:
		return "failed"
	case PeerStateOffline:
		return "offline"
	default:
		return "unknown"
	}
}

type PeerEvent struct {
	PublicKey PublicKey
	State     PeerState
}

type PeerInfo struct {
	PublicKey PublicKey
	Endpoint  net.Addr
	State     PeerState
	RxBytes   uint64
	TxBytes   uint64
	LastSeen  time.Time
}
