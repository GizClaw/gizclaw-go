package core

import (
	"net"
	"testing"
	"time"
)

func TestUDPAddrMethods(t *testing.T) {
	netAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
	addr := UDPAddrFromNetAddr(netAddr)

	if addr.Network() != "udp" {
		t.Errorf("Network() = %s, want udp", addr.Network())
	}

	expected := "127.0.0.1:8080"
	if addr.String() != expected {
		t.Errorf("String() = %s, want %s", addr.String(), expected)
	}
}

func TestUDPTransportSetWriteDeadline(t *testing.T) {
	transport, err := NewUDPTransport("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewUDPTransport() error = %v", err)
	}
	defer transport.Close()

	deadline := time.Now().Add(1 * time.Second)
	err = transport.SetWriteDeadline(deadline)
	if err != nil {
		t.Errorf("SetWriteDeadline() error = %v", err)
	}

	err = transport.SetWriteDeadline(time.Time{})
	if err != nil {
		t.Errorf("SetWriteDeadline(zero) error = %v", err)
	}
}

func TestUDPTransportSendToResolve(t *testing.T) {
	transport, err := NewUDPTransport("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewUDPTransport() error = %v", err)
	}
	defer transport.Close()

	err = transport.SendTo([]byte("test"), NewMockAddr("127.0.0.1:12345"))
	_ = err
}

func TestNewUDPTransportError(t *testing.T) {
	_, err := NewUDPTransport("invalid:address:format")
	if err == nil {
		t.Error("NewUDPTransport(invalid) should return error")
	}
}

func TestMockAddr(t *testing.T) {
	addr := NewMockAddr("test-addr")

	if addr.Network() != "mock" {
		t.Errorf("Network() = %s, want mock", addr.Network())
	}

	if addr.String() != "test-addr" {
		t.Errorf("String() = %s, want test-addr", addr.String())
	}
}

func TestMockTransport(t *testing.T) {
	t1 := NewMockTransport("peer1")
	t2 := NewMockTransport("peer2")

	t1.Connect(t2)

	testData := []byte("hello world")
	if err := t1.SendTo(testData, t2.LocalAddr()); err != nil {
		t.Fatalf("SendTo() error = %v", err)
	}

	buf := make([]byte, 1024)
	n, from, err := t2.RecvFrom(buf)
	if err != nil {
		t.Fatalf("RecvFrom() error = %v", err)
	}

	if string(buf[:n]) != "hello world" {
		t.Errorf("RecvFrom() data = %s, want hello world", string(buf[:n]))
	}

	if from.String() != "peer1" {
		t.Errorf("RecvFrom() from = %s, want peer1", from.String())
	}

	if err := t2.SendTo([]byte("reply"), t1.LocalAddr()); err != nil {
		t.Fatalf("SendTo() error = %v", err)
	}

	n, from, err = t1.RecvFrom(buf)
	if err != nil {
		t.Fatalf("RecvFrom() error = %v", err)
	}

	if string(buf[:n]) != "reply" {
		t.Errorf("RecvFrom() data = %s, want reply", string(buf[:n]))
	}
}

func TestMockTransportClose(t *testing.T) {
	transport := NewMockTransport("test")

	if err := transport.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	err := transport.SendTo([]byte("test"), NewMockAddr("peer"))
	if err != ErrMockTransportClosed {
		t.Errorf("SendTo() after close error = %v, want ErrMockTransportClosed", err)
	}

	buf := make([]byte, 1024)
	_, _, err = transport.RecvFrom(buf)
	if err != ErrMockTransportClosed {
		t.Errorf("RecvFrom() after close error = %v, want ErrMockTransportClosed", err)
	}

	if err := transport.Close(); err != nil {
		t.Errorf("Double Close() error = %v", err)
	}
}

func TestMockTransportSetWriteDeadline(t *testing.T) {
	transport := NewMockTransport("test")
	defer transport.Close()

	err := transport.SetWriteDeadline(time.Now())
	if err != nil {
		t.Errorf("SetWriteDeadline() error = %v", err)
	}
}

func TestMockTransportNoPeer(t *testing.T) {
	transport := NewMockTransport("test")
	defer transport.Close()

	err := transport.SendTo([]byte("test"), NewMockAddr("peer"))
	if err != ErrMockNoPeer {
		t.Errorf("SendTo() without peer error = %v, want ErrMockNoPeer", err)
	}
}

func TestMockTransportInjectPacket(t *testing.T) {
	transport := NewMockTransport("test")
	defer transport.Close()

	from := NewMockAddr("sender")
	if err := transport.InjectPacket([]byte("injected"), from); err != nil {
		t.Fatalf("InjectPacket() error = %v", err)
	}

	buf := make([]byte, 1024)
	n, addr, err := transport.RecvFrom(buf)
	if err != nil {
		t.Fatalf("RecvFrom() error = %v", err)
	}

	if string(buf[:n]) != "injected" {
		t.Errorf("RecvFrom() data = %s, want injected", string(buf[:n]))
	}

	if addr.String() != "sender" {
		t.Errorf("RecvFrom() from = %s, want sender", addr.String())
	}
}
