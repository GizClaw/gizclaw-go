package peer

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	core "github.com/haivivi/giztoy/go/pkg/net/core"
	"github.com/haivivi/giztoy/go/pkg/net/noise"
)

func TestNilListenerGuard(t *testing.T) {
	var l *Listener

	if _, err := l.Accept(); !errors.Is(err, ErrNilListener) {
		t.Fatalf("Accept(nil listener) err=%v, want %v", err, ErrNilListener)
	}

	if _, err := l.Peer(noise.PublicKey{}); !errors.Is(err, ErrNilListener) {
		t.Fatalf("Peer(nil listener) err=%v, want %v", err, ErrNilListener)
	}

	if err := l.Close(); !errors.Is(err, ErrNilListener) {
		t.Fatalf("Close(nil listener) err=%v, want %v", err, ErrNilListener)
	}
}

func TestNilConnGuard(t *testing.T) {
	var c *Conn

	if _, err := c.OpenRPC(); !errors.Is(err, ErrNilConn) {
		t.Fatalf("OpenRPC(nil conn) err=%v, want %v", err, ErrNilConn)
	}
	if _, err := c.AcceptRPC(); !errors.Is(err, ErrNilConn) {
		t.Fatalf("AcceptRPC(nil conn) err=%v, want %v", err, ErrNilConn)
	}
	if err := c.SendEvent(Event{V: 1, Name: "x"}); !errors.Is(err, ErrNilConn) {
		t.Fatalf("SendEvent(nil conn) err=%v, want %v", err, ErrNilConn)
	}
	if _, err := c.ReadEvent(); !errors.Is(err, ErrNilConn) {
		t.Fatalf("ReadEvent(nil conn) err=%v, want %v", err, ErrNilConn)
	}
	if err := c.SendOpusFrame(nil); !errors.Is(err, ErrNilConn) {
		t.Fatalf("SendOpusFrame(nil conn) err=%v, want %v", err, ErrNilConn)
	}
	if _, err := c.ReadOpusFrame(); !errors.Is(err, ErrNilConn) {
		t.Fatalf("ReadOpusFrame(nil conn) err=%v, want %v", err, ErrNilConn)
	}
	if err := c.Close(); !errors.Is(err, ErrNilConn) {
		t.Fatalf("Close(nil conn) err=%v, want %v", err, ErrNilConn)
	}
	if got := c.PublicKey(); got != (noise.PublicKey{}) {
		t.Fatalf("PublicKey(nil conn) = %v, want zero key", got)
	}
}

func TestListenAndCloseOwnedListener(t *testing.T) {
	key, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	l, err := Listen(key, core.WithBindAddr("127.0.0.1:0"), core.WithAllowUnknown(true))
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if _, err := l.Accept(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Accept after Close err=%v, want %v", err, ErrClosed)
	}
}

func TestListenerAcceptAndConnEventOpus(t *testing.T) {
	serverKey, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Generate server key failed: %v", err)
	}
	clientKey, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Generate client key failed: %v", err)
	}

	serverUDP, err := core.NewUDP(serverKey, core.WithBindAddr("127.0.0.1:0"), core.WithAllowUnknown(true))
	if err != nil {
		t.Fatalf("NewUDP(server) failed: %v", err)
	}
	defer serverUDP.Close()

	clientUDP, err := core.NewUDP(clientKey, core.WithBindAddr("127.0.0.1:0"), core.WithAllowUnknown(true))
	if err != nil {
		t.Fatalf("NewUDP(client) failed: %v", err)
	}
	defer clientUDP.Close()

	startReadLoop(serverUDP)
	startReadLoop(clientUDP)

	serverListener, err := Wrap(serverUDP)
	if err != nil {
		t.Fatalf("Wrap(server) failed: %v", err)
	}
	defer serverListener.Close()

	clientListener, err := Wrap(clientUDP)
	if err != nil {
		t.Fatalf("Wrap(client) failed: %v", err)
	}
	defer clientListener.Close()

	acceptCh := make(chan *Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		c, err := serverListener.Accept()
		if err != nil {
			errCh <- err
			return
		}
		acceptCh <- c
	}()

	clientUDP.SetPeerEndpoint(serverKey.Public, serverUDP.HostInfo().Addr)
	serverUDP.SetPeerEndpoint(clientKey.Public, clientUDP.HostInfo().Addr)

	if err := clientUDP.Connect(serverKey.Public); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	waitEstablished(t, serverUDP, clientKey.Public)
	waitEstablished(t, clientUDP, serverKey.Public)

	var serverConn *Conn
	select {
	case serverConn = <-acceptCh:
	case err := <-errCh:
		t.Fatalf("Listener.Accept failed: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("Listener.Accept timeout")
	}

	clientConn, err := clientListener.Peer(serverKey.Public)
	if err != nil {
		t.Fatalf("clientListener.Peer failed: %v", err)
	}

	evt := Event{V: PrologueVersion, Name: "hello"}
	if err := clientConn.SendEvent(evt); err != nil {
		t.Fatalf("SendEvent failed: %v", err)
	}

	gotEvent, err := serverConn.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent failed: %v", err)
	}
	if gotEvent.Name != evt.Name || gotEvent.V != PrologueVersion {
		t.Fatalf("event mismatch: got=%+v want=%+v", gotEvent, evt)
	}
	if gotPK := serverConn.PublicKey(); gotPK != clientKey.Public {
		t.Fatalf("serverConn.PublicKey() mismatch")
	}

	wantStamp := EpochMillis(1234567890123)
	wantRawFrame := []byte("opus-frame")
	frame := StampOpusFrame(wantRawFrame, wantStamp)
	if err := clientConn.SendOpusFrame(frame); err != nil {
		t.Fatalf("SendOpusFrame failed: %v", err)
	}

	gotFrame, err := serverConn.ReadOpusFrame()
	if err != nil {
		t.Fatalf("ReadOpusFrame failed: %v", err)
	}
	if gotFrame.Version() != OpusFrameVersion {
		t.Fatalf("opus frame version=%d, want %d", gotFrame.Version(), OpusFrameVersion)
	}
	if gotFrame.Stamp() != wantStamp {
		t.Fatalf("opus frame stamp=%d, want %d", gotFrame.Stamp(), wantStamp)
	}
	if !bytes.Equal(gotFrame.Frame(), wantRawFrame) {
		t.Fatalf("opus frame payload mismatch: got=%q want=%q", gotFrame.Frame(), wantRawFrame)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("clientConn.Close failed: %v", err)
	}
}

func TestConnOpenAcceptRPC(t *testing.T) {
	serverKey, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Generate server key failed: %v", err)
	}
	clientKey, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Generate client key failed: %v", err)
	}

	serverUDP, err := core.NewUDP(serverKey, core.WithBindAddr("127.0.0.1:0"), core.WithAllowUnknown(true))
	if err != nil {
		t.Fatalf("NewUDP(server) failed: %v", err)
	}
	defer serverUDP.Close()

	clientUDP, err := core.NewUDP(clientKey, core.WithBindAddr("127.0.0.1:0"), core.WithAllowUnknown(true))
	if err != nil {
		t.Fatalf("NewUDP(client) failed: %v", err)
	}
	defer clientUDP.Close()

	startReadLoop(serverUDP)
	startReadLoop(clientUDP)

	serverListener, err := Wrap(serverUDP)
	if err != nil {
		t.Fatalf("Wrap(server) failed: %v", err)
	}
	defer serverListener.Close()

	clientListener, err := Wrap(clientUDP)
	if err != nil {
		t.Fatalf("Wrap(client) failed: %v", err)
	}
	defer clientListener.Close()

	acceptConnCh := make(chan *Conn, 1)
	acceptErrCh := make(chan error, 1)
	go func() {
		c, err := serverListener.Accept()
		if err != nil {
			acceptErrCh <- err
			return
		}
		acceptConnCh <- c
	}()

	clientUDP.SetPeerEndpoint(serverKey.Public, serverUDP.HostInfo().Addr)
	serverUDP.SetPeerEndpoint(clientKey.Public, clientUDP.HostInfo().Addr)

	if err := clientUDP.Connect(serverKey.Public); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	waitEstablished(t, serverUDP, clientKey.Public)
	waitEstablished(t, clientUDP, serverKey.Public)

	var serverConn *Conn
	select {
	case serverConn = <-acceptConnCh:
	case err := <-acceptErrCh:
		t.Fatalf("Listener.Accept failed: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("Listener.Accept timeout")
	}

	clientConn, err := clientListener.Peer(serverKey.Public)
	if err != nil {
		t.Fatalf("clientListener.Peer failed: %v", err)
	}

	rpcAcceptCh := make(chan net.Conn, 1)
	rpcErrCh := make(chan error, 1)
	go func() {
		s, err := serverConn.AcceptRPC()
		if err != nil {
			rpcErrCh <- err
			return
		}
		rpcAcceptCh <- s
	}()

	clientStream, err := clientConn.OpenRPC()
	if err != nil {
		t.Fatalf("OpenRPC failed: %v", err)
	}
	defer clientStream.Close()

	req := []byte(`{"method":"ping"}`)
	if _, err := clientStream.Write(req); err != nil {
		t.Fatalf("client stream write req failed: %v", err)
	}

	var serverStream net.Conn
	select {
	case serverStream = <-rpcAcceptCh:
		defer serverStream.Close()
	case err := <-rpcErrCh:
		t.Fatalf("AcceptRPC failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("AcceptRPC timeout")
	}

	if got := readExactWithTimeout(t, serverStream, len(req), 5*time.Second); !bytes.Equal(got, req) {
		t.Fatalf("server stream request mismatch: got=%q want=%q", got, req)
	}

	resp := []byte(`{"ok":true}`)
	if _, err := serverStream.Write(resp); err != nil {
		t.Fatalf("server stream write resp failed: %v", err)
	}
	if got := readExactWithTimeout(t, clientStream, len(resp), 5*time.Second); !bytes.Equal(got, resp) {
		t.Fatalf("client stream response mismatch: got=%q want=%q", got, resp)
	}
}

func startReadLoop(u *core.UDP) {
	go func() {
		buf := make([]byte, 65535)
		for {
			if _, _, err := u.ReadFrom(buf); err != nil {
				return
			}
		}
	}()
}

func waitEstablished(t *testing.T, u *core.UDP, pk noise.PublicKey) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if info := u.PeerInfo(pk); info != nil && info.State == core.PeerStateEstablished {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	info := u.PeerInfo(pk)
	if info == nil {
		t.Fatal("peer info is nil")
	}
	t.Fatalf("peer state not established: got=%s", info.State)
}

func readExactWithTimeout(t *testing.T, r io.Reader, n int, timeout time.Duration) []byte {
	t.Helper()

	errCh := make(chan error, 1)
	buf := make([]byte, n)
	go func() {
		_, err := io.ReadFull(r, buf)
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ReadFull failed: %v", err)
		}
		return buf
	case <-time.After(timeout):
		t.Fatalf("ReadFull timeout after %s", timeout)
		return nil
	}
}
