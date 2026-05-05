package giznet

import "github.com/GizClaw/gizclaw-go/pkg/giznet/internal/core"

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

type ListenConfig struct {
	Addr string

	// SecurityPolicy decides whether inbound peers and services are allowed.
	// If nil, only peers already registered by dialing are accepted and only service 0 is allowed.
	SecurityPolicy SecurityPolicy

	// PeerEventHandler is called synchronously from the Noise peer event path.
	// The handler must not block.
	PeerEventHandler PeerEventHandler
}

func Listen(key *KeyPair) (*Listener, error) {
	return new(ListenConfig).Listen(key)
}

func (c *ListenConfig) Listen(key *KeyPair) (*Listener, error) {
	l := &Listener{
		closedCh:    make(chan struct{}),
		established: make(map[PublicKey]*Conn),
		events:      make(chan PeerEvent, 64),
	}
	if c != nil {
		l.evtHandler = c.PeerEventHandler
	}

	// Append our internal handler last so listener-level Conn ownership and
	// peer event handling stay in sync with core peer state changes.
	allOpts := c.options()
	allOpts = append(allOpts, core.WithOnPeerEvent(l.onPeerEvent))
	u, err := core.NewUDP(key, allOpts...)
	if err != nil {
		return nil, err
	}
	l.udp = u

	return l, nil
}

func (c *ListenConfig) options() []core.Option {
	if c == nil {
		return nil
	}
	opts := make([]core.Option, 0, 3)
	if c.Addr != "" {
		opts = append(opts, core.WithBindAddr(c.Addr))
	}
	if c.SecurityPolicy != nil {
		opts = append(opts, core.WithAllowFunc(c.SecurityPolicy.AllowPeer))
		opts = append(opts, core.WithServiceMuxConfig(core.ServiceMuxConfig{
			OnNewService: c.SecurityPolicy.AllowService,
		}))
	}
	return opts
}
