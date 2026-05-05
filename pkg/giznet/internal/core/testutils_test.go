package core

import (
	"net"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/giznet/internal/noise"
)

// Non-ProtocolKCP payload markers used across core tests (legacy EVENT/OPUS bytes).
const (
	testDirectProtoA byte = 0x03
	testDirectProtoB byte = 0x10
)

func mustServiceMux(t *testing.T, u *UDP, pk noise.PublicKey) *ServiceMux {
	t.Helper()

	smux, err := u.PeerServiceMux(pk)
	if err != nil {
		t.Fatalf("PeerServiceMux failed: %v", err)
	}
	return smux
}

func mustOpenStream(t *testing.T, u *UDP, pk noise.PublicKey, service uint64) net.Conn {
	t.Helper()

	stream, err := mustServiceMux(t, u, pk).OpenStream(service)
	if err != nil {
		t.Fatalf("OpenStream(service=%d) failed: %v", service, err)
	}
	return stream
}

func mustAcceptStream(t *testing.T, u *UDP, pk noise.PublicKey, service uint64) net.Conn {
	t.Helper()

	stream, err := mustServiceMux(t, u, pk).AcceptStream(service)
	if err != nil {
		t.Fatalf("AcceptStream(service=%d) failed: %v", service, err)
	}
	return stream
}

// createConnectedPair creates two connected UDP instances for testing.
// Returns (server, client, serverKey, clientKey). Caller must Close both UDPs.
func createConnectedPair(t *testing.T, serverOpts ...Option) (*UDP, *UDP, *noise.KeyPair, *noise.KeyPair) {
	t.Helper()

	serverKey, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	clientKey, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	baseOpts := []Option{WithBindAddr("127.0.0.1:0"), WithAllowFunc(func(noise.PublicKey) bool {
		return true
	})}
	server, err := NewUDP(serverKey, append(baseOpts, serverOpts...)...)
	if err != nil {
		t.Fatalf("NewUDP server: %v", err)
	}

	client, err := NewUDP(clientKey, WithBindAddr("127.0.0.1:0"), WithAllowFunc(func(noise.PublicKey) bool {
		return true
	}))
	if err != nil {
		server.Close()
		t.Fatalf("NewUDP client: %v", err)
	}

	serverAddr := server.HostInfo().Addr
	clientAddr := client.HostInfo().Addr
	server.SetPeerEndpoint(clientKey.Public, clientAddr)
	client.SetPeerEndpoint(serverKey.Public, serverAddr)

	// Start receive loops so handshake and packet routing can progress.
	go func() {
		buf := make([]byte, 65535)
		for {
			if _, _, err := client.ReadFrom(buf); err != nil {
				return
			}
		}
	}()
	go func() {
		buf := make([]byte, 65535)
		for {
			if _, _, err := server.ReadFrom(buf); err != nil {
				return
			}
		}
	}()

	if err := client.Connect(serverKey.Public); err != nil {
		client.Close()
		server.Close()
		t.Fatalf("Handshake failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	return server, client, serverKey, clientKey
}
