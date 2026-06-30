package giznet

type SecurityPolicy interface {
	AllowPeer(PublicKey) bool
	AllowService(PublicKey, uint64) bool
}

type PeerEventHandler interface {
	HandlePeerEvent(PeerEvent)
}

type PeerEventHandleFunc func(PeerEvent)

func (f PeerEventHandleFunc) HandlePeerEvent(ev PeerEvent) {
	f(ev)
}
