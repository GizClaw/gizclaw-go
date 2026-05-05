package giznet

import (
	"errors"
	"testing"
)

type allowAllSecurityPolicy struct{}

func (allowAllSecurityPolicy) AllowPeer(PublicKey) bool {
	return true
}

func (allowAllSecurityPolicy) AllowService(_ PublicKey, service uint64) bool {
	return service == 0
}

func TestListenerCloseLeavesEventQueueOpen(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	listener, err := (&ListenConfig{
		Addr:           "127.0.0.1:0",
		SecurityPolicy: allowAllSecurityPolicy{},
	}).Listen(key)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer func() { _ = listener.Close() }()

	acceptErrCh := make(chan error, 1)
	go func() {
		_, err := listener.Accept()
		acceptErrCh <- err
	}()

	if err := listener.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if err := <-acceptErrCh; !errors.Is(err, ErrClosed) {
		t.Fatalf("Accept after Close err=%v, want %v", err, ErrClosed)
	}

	select {
	case _, ok := <-listener.events:
		if !ok {
			t.Fatal("event queue should remain open after Close")
		}
	default:
	}

	if delivered := listener.onPeerEvent(PeerEvent{}); delivered {
		t.Fatal("onPeerEvent should reject events after Close")
	}
}

func TestListenConfigPeerEventHandleFuncReceivesEvents(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	events := make(chan PeerEvent, 2)
	cfg := ListenConfig{
		Addr: "127.0.0.1:0",
		PeerEventHandler: PeerEventHandleFunc(func(ev PeerEvent) {
			events <- ev
		}),
	}
	listener, err := cfg.Listen(key)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	offline := PeerEvent{PublicKey: key.Public, State: PeerStateOffline}
	if delivered := listener.onPeerEvent(offline); !delivered {
		t.Fatal("onPeerEvent should deliver offline event")
	}

	if got := <-events; got != offline {
		t.Fatalf("offline event=%+v, want %+v", got, offline)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestListenerReleaseConnOnlyReleasesMatchingConn(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	conn := &Conn{pk: key.Public}
	other := &Conn{pk: key.Public}

	listener := &Listener{
		established: map[PublicKey]*Conn{
			key.Public: conn,
		},
	}

	if err := listener.releaseConn(other, nil); err != nil {
		t.Fatalf("releaseConn(stale) error = %v", err)
	}
	if got := listener.established[key.Public]; got != conn {
		t.Fatal("releaseConn should ignore stale conn ownership for peer")
	}

	if err := listener.releaseConn(conn, nil); err != nil {
		t.Fatalf("releaseConn(active) error = %v", err)
	}
	if _, ok := listener.established[key.Public]; ok {
		t.Fatal("releaseConn should remove listener ownership for peer")
	}
}

func TestListenerReleaseConnRunsCallbackWhileLocked(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	conn := &Conn{pk: key.Public}
	listener := &Listener{
		established: map[PublicKey]*Conn{
			key.Public: conn,
		},
	}
	locked := false

	if err := listener.releaseConn(conn, func() error {
		locked = !listener.mu.TryLock()
		if !locked {
			listener.mu.Unlock()
		}
		return nil
	}); err != nil {
		t.Fatalf("releaseConn error = %v", err)
	}
	if !locked {
		t.Fatal("releaseConn callback should run while listener mutex is held")
	}
	if _, ok := listener.established[key.Public]; ok {
		t.Fatal("releaseConn should remove listener ownership after callback")
	}
}

func TestListenerPeerAndDialDoNotReturnConnAfterClose(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	conn := &Conn{pk: key.Public}
	listener := &Listener{
		established: map[PublicKey]*Conn{
			key.Public: conn,
		},
	}
	listener.closed.Store(true)

	if got, ok := listener.Peer(key.Public); ok || got != nil {
		t.Fatalf("Peer after close = %v, %v; want nil, false", got, ok)
	}
	if got, err := listener.Dial(key.Public, nil); !errors.Is(err, ErrClosed) || got != nil {
		t.Fatalf("Dial after close = %v, %v; want nil, %v", got, err, ErrClosed)
	}
}
