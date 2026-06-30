package giznoise

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
	"github.com/GizClaw/gizclaw-go/pkgs/giznet/internal/core"
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
	established map[giznet.PublicKey]*Conn
	events      chan giznet.PeerEvent
	evtHandler  giznet.PeerEventHandler
}

func (l *Listener) onPeerEvent(ev core.PeerEvent) bool {
	if l.closed.Load() {
		return false
	}
	peerEvent := giznet.PeerEvent{
		PublicKey: fromNoisePublicKey(ev.PublicKey),
		State:     fromCorePeerState(ev.State),
	}
	if l.evtHandler != nil {
		l.evtHandler.HandlePeerEvent(peerEvent)
	}

	select {
	case l.events <- peerEvent:
		return true
	default:
		return false
	}
}

func (l *Listener) Accept() (giznet.Conn, error) {
	return l.AcceptConn()
}

func (l *Listener) AcceptConn() (*Conn, error) {
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
			if ev.State != giznet.PeerStateEstablished {
				continue
			}
			l.mu.Lock()
			if conn, ok := l.established[ev.PublicKey]; ok {
				l.mu.Unlock()
				return conn, nil
			}
			l.mu.Unlock()

			peer, err := l.udp.GetPeer(toNoisePublicKey(ev.PublicKey))
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
func (l *Listener) Peer(pk giznet.PublicKey) (*Conn, bool) {
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

func (l *Listener) SetPeerEndpoint(pk giznet.PublicKey, endpoint *net.UDPAddr) {
	l.udp.SetPeerEndpoint(toNoisePublicKey(pk), endpoint)
}

func (l *Listener) Connect(pk giznet.PublicKey) error {
	return l.udp.Connect(toNoisePublicKey(pk))
}

// Dial sets the peer endpoint, performs a synchronous Noise IK handshake,
// and returns this listener's active Conn for the peer.
func (l *Listener) Dial(pk giznet.PublicKey, addr *net.UDPAddr) (giznet.Conn, error) {
	return l.DialConn(pk, addr)
}

func (l *Listener) DialConn(pk giznet.PublicKey, addr *net.UDPAddr) (*Conn, error) {
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

	peer, err := l.udp.GetPeer(toNoisePublicKey(pk))
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
	if l == nil || l.udp == nil {
		return nil
	}
	return &UDP{inner: l.udp}
}

func (l *Listener) HostInfo() *HostInfo {
	return fromCoreHostInfo(l.udp.HostInfo())
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
