package giznoise

import (
	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/giznet/internal/core"
	"github.com/GizClaw/gizclaw-go/pkgs/giznet/internal/noise"
)

// CipherMode selects the low-level Noise cipher mode used by giznet.
type CipherMode string

const (
	// CipherModeChaChaPoly uses ChaCha20-Poly1305 and is the default.
	CipherModeChaChaPoly CipherMode = "chacha_poly"
	// CipherModeAES256GCM uses AES-256-GCM.
	CipherModeAES256GCM CipherMode = "aes_256_gcm"
	// CipherModePlaintext disables encryption for diagnostics while preserving wire overhead.
	CipherModePlaintext CipherMode = "plaintext"
)

type ListenConfig struct {
	Addr string

	// SecurityPolicy decides whether inbound peers and services are allowed.
	// If nil, only peers already registered by dialing are accepted and only service 0 is allowed.
	SecurityPolicy giznet.SecurityPolicy

	// PeerEventHandler is called synchronously from the Noise peer event path.
	// The handler must not block.
	PeerEventHandler giznet.PeerEventHandler

	// CipherMode selects the low-level Noise cipher mode.
	// If empty, ChaCha20-Poly1305 is used for backwards compatibility.
	CipherMode CipherMode
}

func Listen(key *giznet.KeyPair) (*Listener, error) {
	return new(ListenConfig).Listen(key)
}

func (c *ListenConfig) Listen(key *giznet.KeyPair) (*Listener, error) {
	l := &Listener{
		closedCh:    make(chan struct{}),
		established: make(map[giznet.PublicKey]*Conn),
		events:      make(chan giznet.PeerEvent, 64),
	}
	if c != nil {
		l.evtHandler = c.PeerEventHandler
	}

	// Append our internal handler last so listener-level Conn ownership and
	// peer event handling stay in sync with core peer state changes.
	allOpts := c.options()
	allOpts = append(allOpts, core.WithOnPeerEvent(l.onPeerEvent))
	u, err := core.NewUDP(toNoiseKeyPair(key), allOpts...)
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
		opts = append(opts, core.WithAllowFunc(func(pk noise.PublicKey) bool {
			return c.SecurityPolicy.AllowPeer(fromNoisePublicKey(pk))
		}))
		opts = append(opts, core.WithServiceMuxConfig(core.ServiceMuxConfig{
			OnNewService: func(pk noise.PublicKey, service uint64) bool {
				return c.SecurityPolicy.AllowService(fromNoisePublicKey(pk), service)
			},
		}))
	}
	if c.CipherMode != "" {
		opts = append(opts, core.WithCipherMode(noise.CipherMode(c.CipherMode)))
	}
	return opts
}
