package gizhttp

import (
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

type testSecurityPolicy struct {
	allowService func(giznet.PublicKey, uint64) bool
}

func (p testSecurityPolicy) AllowPeer(giznet.PublicKey) bool {
	return true
}

func (p testSecurityPolicy) AllowService(pk giznet.PublicKey, service uint64) bool {
	if p.allowService == nil {
		return service == 0
	}
	return p.allowService(pk, service)
}

// newListenerNode creates a giznet.Listener for tests using only public APIs.
func newListenerNode(t *testing.T, key *giznet.KeyPair, cfgs ...giznet.ListenConfig) *giznet.Listener {
	t.Helper()

	cfg := giznet.ListenConfig{
		Addr:           "127.0.0.1:0",
		SecurityPolicy: testSecurityPolicy{},
	}
	if len(cfgs) > 0 {
		if cfgs[0].SecurityPolicy != nil {
			cfg.SecurityPolicy = cfgs[0].SecurityPolicy
		}
		cfg.PeerEventHandler = cfgs[0].PeerEventHandler
	}
	l, err := (&cfg).Listen(key)
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
