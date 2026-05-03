package gizhttp

import (
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

// newListenerNode creates a giznet.Listener for tests using only public APIs.
func newListenerNode(t *testing.T, key *giznet.KeyPair, opts ...giznet.Option) *giznet.Listener {
	t.Helper()

	defaults := []giznet.Option{
		giznet.WithBindAddr("127.0.0.1:0"),
		giznet.WithAllowUnknown(true),
		giznet.WithDecryptWorkers(1),
	}
	l, err := giznet.Listen(key, append(defaults, opts...)...)
	if err != nil {
		t.Fatalf("giznet.Listen failed: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	u := l.UDP()
	go func() {
		buf := make([]byte, 65535)
		for {
			if _, _, err := u.ReadFrom(buf); err != nil {
				return
			}
		}
	}()

	return l
}

func connectListenerNodes(t *testing.T, client *giznet.Listener, clientKey *giznet.KeyPair, server *giznet.Listener, serverKey *giznet.KeyPair) (*giznet.Conn, *giznet.Conn) {
	t.Helper()

	server.SetPeerEndpoint(clientKey.Public, client.HostInfo().Addr)

	acceptCh := make(chan *giznet.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			errCh <- err
			return
		}
		acceptCh <- conn
	}()

	clientConn, err := client.Dial(serverKey.Public, server.HostInfo().Addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}

	select {
	case serverConn := <-acceptCh:
		return clientConn, serverConn
	case err := <-errCh:
		t.Fatalf("Accept failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Accept timeout")
	}
	return nil, nil
}

func waitForPeerEstablished(t *testing.T, u *giznet.UDP, pk giznet.PublicKey) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		info := u.PeerInfo(pk)
		if info != nil && info.State == giznet.PeerStateEstablished {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	info := u.PeerInfo(pk)
	if info == nil {
		t.Fatalf("peer %x was not registered before timeout", pk)
	}
	t.Fatalf("peer %x state=%v, want %v", pk, info.State, giznet.PeerStateEstablished)
}
