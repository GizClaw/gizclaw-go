package giznoise

import (
	"io"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkgs/giznet"
)

type allowAllPolicy struct{}

func (allowAllPolicy) AllowPeer(giznet.PublicKey) bool {
	return true
}

func (allowAllPolicy) AllowService(giznet.PublicKey, uint64) bool {
	return true
}

func TestListenerDialAcceptAndUDPDiagnostics(t *testing.T) {
	serverKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(server) error = %v", err)
	}
	clientKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(client) error = %v", err)
	}

	server, err := (&ListenConfig{Addr: "127.0.0.1:0", SecurityPolicy: allowAllPolicy{}}).Listen(serverKey)
	if err != nil {
		t.Fatalf("server Listen error = %v", err)
	}
	defer server.Close()
	go drainUDP(server.UDP())

	client, err := (&ListenConfig{Addr: "127.0.0.1:0", SecurityPolicy: allowAllPolicy{}}).Listen(clientKey)
	if err != nil {
		t.Fatalf("client Listen error = %v", err)
	}
	defer client.Close()
	go drainUDP(client.UDP())

	accepted := make(chan giznet.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			errCh <- err
			return
		}
		accepted <- conn
	}()

	clientConn, err := client.Dial(serverKey.Public, server.HostInfo().Addr)
	if err != nil {
		t.Fatalf("Dial error = %v", err)
	}
	defer clientConn.Close()

	var serverConn giznet.Conn
	select {
	case serverConn = <-accepted:
	case err := <-errCh:
		t.Fatalf("Accept error = %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Accept timeout")
	}
	defer serverConn.Close()

	if got := client.HostInfo().PublicKey; got != clientKey.Public {
		t.Fatalf("client HostInfo public key = %v, want %v", got, clientKey.Public)
	}
	if got := serverConn.PublicKey(); got != clientKey.Public {
		t.Fatalf("server Conn public key = %v, want %v", got, clientKey.Public)
	}
	if info := serverConn.PeerInfo(); info == nil || info.PublicKey != clientKey.Public || info.State != giznet.PeerStateEstablished {
		t.Fatalf("server PeerInfo = %+v", info)
	}
	if _, ok := server.Peer(clientKey.Public); !ok {
		t.Fatal("server listener did not retain accepted peer")
	}

	service := serverConn.ListenService(9)
	defer service.Close()
	if service.Addr().Network() != "kcp-service" {
		t.Fatalf("service addr network = %q", service.Addr().Network())
	}
	acceptedStream := make(chan streamResult, 1)
	go func() {
		stream, err := service.Accept()
		acceptedStream <- streamResult{conn: stream, err: err}
	}()

	clientStream, err := clientConn.Dial(9)
	if err != nil {
		t.Fatalf("client service Dial error = %v", err)
	}
	defer clientStream.Close()

	var serverStream streamResult
	select {
	case serverStream = <-acceptedStream:
	case <-time.After(5 * time.Second):
		t.Fatal("service Accept timeout")
	}
	if serverStream.err != nil {
		t.Fatalf("service Accept error = %v", serverStream.err)
	}
	defer serverStream.conn.Close()

	if err := serverStream.conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("server stream SetDeadline error = %v", err)
	}
	if err := clientStream.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("client stream SetDeadline error = %v", err)
	}
	if _, err := clientStream.Write([]byte("ping")); err != nil {
		t.Fatalf("client stream Write error = %v", err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(serverStream.conn, buf); err != nil {
		t.Fatalf("server stream ReadFull error = %v", err)
	}
	if string(buf) != "ping" {
		t.Fatalf("server stream read %q", string(buf))
	}
	if err := serverConn.CloseService(9); err != nil {
		t.Fatalf("CloseService error = %v", err)
	}
}

type streamResult struct {
	conn interface {
		io.Reader
		io.Writer
		SetDeadline(time.Time) error
		Close() error
	}
	err error
}

func TestNilListenerAndConnErrors(t *testing.T) {
	var listener *Listener
	if _, err := listener.AcceptConn(); err != ErrNilListener {
		t.Fatalf("nil AcceptConn error = %v, want %v", err, ErrNilListener)
	}
	if err := listener.Close(); err != ErrNilListener {
		t.Fatalf("nil Close error = %v, want %v", err, ErrNilListener)
	}
	var conn *Conn
	if err := conn.Close(); err != ErrNilConn {
		t.Fatalf("nil Conn Close error = %v, want %v", err, ErrNilConn)
	}
	if conn.PublicKey() != (giznet.PublicKey{}) {
		t.Fatal("nil Conn PublicKey is not zero")
	}
	if conn.PeerInfo() != nil {
		t.Fatal("nil Conn PeerInfo is not nil")
	}
}

func drainUDP(u *UDP) {
	buf := make([]byte, 65535)
	for {
		if _, _, err := u.ReadFrom(buf); err != nil {
			return
		}
	}
}
