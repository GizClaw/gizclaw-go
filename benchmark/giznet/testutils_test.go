package giznet_test

import (
	"net"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

type benchSecurityPolicy struct {
	allowService func(giznet.PublicKey, uint64) bool
}

func (p benchSecurityPolicy) AllowPeer(giznet.PublicKey) bool {
	return true
}

func (p benchSecurityPolicy) AllowService(pk giznet.PublicKey, service uint64) bool {
	if p.allowService == nil {
		return service == 0
	}
	return p.allowService(pk, service)
}

// peerBenchMux is the minimal mux surface used by public benchmarks.
type peerBenchMux interface {
	Write(protocol byte, data []byte) (n int, err error)
	Read(buf []byte) (protocol byte, n int, err error)
	OpenStream(service uint64) (net.Conn, error)
	AcceptStream(service uint64) (net.Conn, error)
}

func newBenchUDPNode(tb testing.TB, key *giznet.KeyPair) *giznet.UDP {
	tb.Helper()

	l, err := (&giznet.ListenConfig{
		Addr:           "127.0.0.1:0",
		SecurityPolicy: benchSecurityPolicy{},
	}).Listen(key)
	if err != nil {
		tb.Fatalf("giznet.Listen failed: %v", err)
	}
	tb.Cleanup(func() { _ = l.Close() })

	u := l.UDP()
	go func() {
		buf := make([]byte, 65535)
		for {
			if _, _, err := u.ReadFrom(buf); err != nil {
				return
			}
		}
	}()

	return u
}

func connectBenchNodes(tb testing.TB, client *giznet.UDP, clientKey *giznet.KeyPair, server *giznet.UDP, serverKey *giznet.KeyPair) {
	tb.Helper()

	client.SetPeerEndpoint(serverKey.Public, server.HostInfo().Addr)
	server.SetPeerEndpoint(clientKey.Public, client.HostInfo().Addr)

	if err := client.Connect(serverKey.Public); err != nil {
		tb.Fatalf("Connect failed: %v", err)
	}

	waitBenchPeerEstablished(tb, client, serverKey.Public)
	waitBenchPeerEstablished(tb, server, clientKey.Public)
}

func waitBenchPeerEstablished(tb testing.TB, u *giznet.UDP, pk giznet.PublicKey) {
	tb.Helper()

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
		tb.Fatalf("peer %x was not registered before timeout", pk)
	}
	tb.Fatalf("peer %x state=%v, want %v", pk, info.State, giznet.PeerStateEstablished)
}

func mustPeerBenchMux(tb testing.TB, u *giznet.UDP, pk giznet.PublicKey) peerBenchMux {
	tb.Helper()

	smux, err := u.PeerServiceMux(pk)
	if err != nil {
		tb.Fatalf("PeerServiceMux failed: %v", err)
	}
	return smux
}

func newBenchListenerNode(tb testing.TB, key *giznet.KeyPair, cfgs ...giznet.ListenConfig) *giznet.Listener {
	tb.Helper()

	cfg := giznet.ListenConfig{
		Addr:           "127.0.0.1:0",
		SecurityPolicy: benchSecurityPolicy{},
	}
	if len(cfgs) > 0 {
		if cfgs[0].SecurityPolicy != nil {
			cfg.SecurityPolicy = cfgs[0].SecurityPolicy
		}
		cfg.PeerEventHandler = cfgs[0].PeerEventHandler
	}
	l, err := (&cfg).Listen(key)
	if err != nil {
		tb.Fatalf("giznet.Listen failed: %v", err)
	}
	tb.Cleanup(func() { _ = l.Close() })

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

func connectBenchListenerNodes(tb testing.TB, client *giznet.Listener, clientKey *giznet.KeyPair, server *giznet.Listener, serverKey *giznet.KeyPair) (*giznet.Conn, *giznet.Conn) {
	tb.Helper()

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
		tb.Fatalf("Dial failed: %v", err)
	}

	select {
	case serverConn := <-acceptCh:
		return clientConn, serverConn
	case err := <-errCh:
		tb.Fatalf("Accept failed: %v", err)
	case <-time.After(5 * time.Second):
		tb.Fatal("Accept timeout")
	}
	return nil, nil
}
