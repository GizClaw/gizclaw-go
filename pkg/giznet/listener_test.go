package giznet

import (
	"errors"
	"testing"
)

func TestListenerCloseLeavesPeerEventsOpen(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}
	listener, err := Listen(key, WithBindAddr("127.0.0.1:0"), WithAllowUnknown(true))
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
			t.Fatal("events channel should remain open after Close")
		}
	default:
	}

	if delivered := listener.onPeerEvent(PeerEvent{}); delivered {
		t.Fatal("onPeerEvent should reject events after Close")
	}
}
