package giznet

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/GizClaw/gizclaw-go/pkg/giznet/internal/core"
)

type Listener struct {
	mu sync.Mutex

	udp       *core.UDP
	closeOnce sync.Once
	closedCh  chan struct{}
	closed    atomic.Bool
	// established tracks peers that already have an active Conn owned by this
	// listener. A peer public key can have at most one active Conn here until
	// that Conn is closed and releases the entry.
	established map[PublicKey]*Conn
	events      chan PeerEvent
	evtHandler  PeerEventHandler
}

func (l *Listener) onPeerEvent(ev PeerEvent) bool {
	if l.closed.Load() {
		return false
	}
	if l.evtHandler != nil {
		l.evtHandler.HandlePeerEvent(ev)
	}

	select {
	case l.events <- ev:
		return true
	default:
		return false
	}
}

func (l *Listener) Accept() (*Conn, error) {
	if l == nil {
		return nil, ErrNilListener
	}

	for {
		select {
		case <-l.closedCh:
			return nil, ErrClosed
		case ev, ok := <-l.events:
			if !ok {
				return nil, ErrClosed
			}
			if ev.State != core.PeerStateEstablished {
				continue
			}
			l.mu.Lock()
			if conn, ok := l.established[ev.PublicKey]; ok {
				l.mu.Unlock()
				return conn, nil
			}
			l.mu.Unlock()

			peer, err := l.udp.GetPeer(ev.PublicKey)
			if err != nil {
				continue
			}
			smux, err := peer.ServiceMux()
			if err != nil {
				continue
			}
			l.mu.Lock()
			if existing, ok := l.established[ev.PublicKey]; ok {
				l.mu.Unlock()
				return existing, nil
			}
			conn := &Conn{pk: ev.PublicKey, peer: peer, smux: smux, listener: l}
			l.established[ev.PublicKey] = conn
			l.mu.Unlock()
			return conn, nil
		}
	}
}

// Peer returns the active Conn owned by this listener for pk.
func (l *Listener) Peer(pk PublicKey) (*Conn, bool) {
	if l == nil {
		return nil, false
	}
	if l.closed.Load() {
		return nil, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	conn, ok := l.established[pk]
	return conn, ok
}

func (l *Listener) releaseConn(conn *Conn, fn func() error) error {
	if l == nil || conn == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if fn != nil {
		if err := fn(); err != nil {
			return err
		}
	}
	if l.established[conn.pk] == conn {
		delete(l.established, conn.pk)
	}
	return nil
}

func (l *Listener) SetPeerEndpoint(pk PublicKey, endpoint *net.UDPAddr) {
	l.udp.SetPeerEndpoint(pk, endpoint)
}

func (l *Listener) Connect(pk PublicKey) error {
	return l.udp.Connect(pk)
}

// Dial sets the peer endpoint, performs a synchronous Noise IK handshake,
// and returns this listener's active Conn for the peer.
func (l *Listener) Dial(pk PublicKey, addr *net.UDPAddr) (*Conn, error) {
	if l == nil {
		return nil, ErrNilListener
	}
	if l.closed.Load() {
		return nil, ErrClosed
	}
	l.mu.Lock()
	if conn, ok := l.established[pk]; ok {
		l.mu.Unlock()
		return conn, nil
	}
	l.mu.Unlock()

	l.SetPeerEndpoint(pk, addr)
	if err := l.Connect(pk); err != nil {
		return nil, err
	}

	l.mu.Lock()
	if conn, ok := l.established[pk]; ok {
		l.mu.Unlock()
		return conn, nil
	}
	l.mu.Unlock()

	peer, err := l.udp.GetPeer(pk)
	if err != nil {
		return nil, err
	}
	smux, err := peer.ServiceMux()
	if err != nil {
		return nil, err
	}
	conn := &Conn{pk: pk, peer: peer, smux: smux, listener: l}
	l.mu.Lock()
	if existing, ok := l.established[pk]; ok {
		l.mu.Unlock()
		return existing, nil
	}
	l.established[pk] = conn
	l.mu.Unlock()
	return conn, nil
}

func (l *Listener) UDP() *UDP {
	return l.udp
}

func (l *Listener) HostInfo() *HostInfo {
	return l.udp.HostInfo()
}

func (l *Listener) Close() error {
	if l == nil {
		return ErrNilListener
	}

	var err error
	l.closeOnce.Do(func() {
		close(l.closedCh)
		l.closed.Store(true)
		// Do not close l.events here. UDP teardown can race with a final
		// onPeerEvent callback, and callers already observe shutdown via
		// closedCh/ErrClosed from Accept.
		err = l.udp.Close()
	})

	return err
}
