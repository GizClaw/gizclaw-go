package kcp

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// serviceMuxPair creates a connected pair of ServiceMux instances.
func serviceMuxPair() (client, server *ServiceMux) {
	var clientMux, serverMux *ServiceMux

	clientMux = NewServiceMux(ServiceMuxConfig{
		IsClient: true,
		Output: func(service uint64, data []byte) error {
			return serverMux.Input(service, data)
		},
	})
	serverMux = NewServiceMux(ServiceMuxConfig{
		IsClient: false,
		Output: func(service uint64, data []byte) error {
			return clientMux.Input(service, data)
		},
	})

	return clientMux, serverMux
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

func TestServiceMux_OpenWriteThenAccept(t *testing.T) {
	client, server := serviceMuxPair()
	defer client.Close()
	defer server.Close()

	stream, err := client.OpenStream(1)
	if err != nil {
		t.Fatalf("client OpenStream failed: %v", err)
	}
	defer stream.Close()

	msg := []byte("hello-direct-kcp")
	writeErr := make(chan error, 1)
	go func() {
		_, err := stream.Write(msg)
		writeErr <- err
	}()

	accepted, service, err := server.AcceptStream()
	if err != nil {
		t.Fatalf("server AcceptStream failed: %v", err)
	}
	defer accepted.Close()

	if service != 1 {
		t.Fatalf("accepted service=%d, want 1", service)
	}

	if got := readExactWithTimeout(t, accepted, len(msg), 5*time.Second); !bytes.Equal(got, msg) {
		t.Fatalf("server recv mismatch: got=%q want=%q", got, msg)
	}

	select {
	case err := <-writeErr:
		if err != nil {
			t.Fatalf("client write failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("client write did not complete")
	}
}

func TestServiceMux_BidirectionalDataPath(t *testing.T) {
	client, server := serviceMuxPair()
	defer client.Close()
	defer server.Close()

	clientStream, err := client.OpenStream(0)
	if err != nil {
		t.Fatalf("client OpenStream failed: %v", err)
	}
	defer clientStream.Close()

	request := []byte("req")
	if _, err := clientStream.Write(request); err != nil {
		t.Fatalf("client write req failed: %v", err)
	}

	serverStream, svc, err := server.AcceptStream()
	if err != nil {
		t.Fatalf("server AcceptStream failed: %v", err)
	}
	defer serverStream.Close()
	if svc != 0 {
		t.Fatalf("accepted service=%d, want 0", svc)
	}

	if got := readExactWithTimeout(t, serverStream, len(request), 5*time.Second); !bytes.Equal(got, request) {
		t.Fatalf("server recv req mismatch: got=%q want=%q", got, request)
	}

	response := []byte("resp")
	if _, err := serverStream.Write(response); err != nil {
		t.Fatalf("server write resp failed: %v", err)
	}

	if got := readExactWithTimeout(t, clientStream, len(response), 5*time.Second); !bytes.Equal(got, response) {
		t.Fatalf("client recv resp mismatch: got=%q want=%q", got, response)
	}
}

func TestServiceMux_AcceptStreamOn(t *testing.T) {
	client, server := serviceMuxPair()
	defer client.Close()
	defer server.Close()

	const svc uint64 = 7
	clientStream, err := client.OpenStream(svc)
	if err != nil {
		t.Fatalf("client OpenStream failed: %v", err)
	}
	defer clientStream.Close()

	msg := []byte("accept-on-service")
	if _, err := clientStream.Write(msg); err != nil {
		t.Fatalf("client write failed: %v", err)
	}

	serverStream, err := server.AcceptStreamOn(svc)
	if err != nil {
		t.Fatalf("server AcceptStreamOn failed: %v", err)
	}
	defer serverStream.Close()

	if got := readExactWithTimeout(t, serverStream, len(msg), 5*time.Second); !bytes.Equal(got, msg) {
		t.Fatalf("server recv mismatch: got=%q want=%q", got, msg)
	}
}

func TestServiceMux_RejectService(t *testing.T) {
	mux := NewServiceMux(ServiceMuxConfig{
		OnNewService: func(service uint64) bool {
			return service != 99
		},
	})
	defer mux.Close()

	err := mux.Input(99, []byte{0x01, 0x02, 0x03})
	if !errors.Is(err, ErrServiceRejected) {
		t.Fatalf("Input(rejected service) err=%v, want %v", err, ErrServiceRejected)
	}
}

func TestServiceMux_OutputErrorObservable(t *testing.T) {
	injected := errors.New("injected output error")

	var callbackCount atomic.Uint64
	var callbackService atomic.Uint64

	mux := NewServiceMux(ServiceMuxConfig{
		Output: func(service uint64, data []byte) error {
			_ = service
			_ = data
			return injected
		},
		OnOutputError: func(service uint64, err error) {
			if !errors.Is(err, injected) {
				t.Errorf("OnOutputError err=%v, want injected", err)
			}
			callbackService.Store(service)
			callbackCount.Add(1)
		},
	})
	defer mux.Close()

	stream, err := mux.OpenStream(0)
	if err != nil {
		t.Fatalf("OpenStream failed: %v", err)
	}
	defer stream.Close()

	_ = stream.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
	_, _ = stream.Write([]byte("trigger-output"))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if mux.OutputErrorCount() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := mux.OutputErrorCount(); got == 0 {
		t.Fatalf("OutputErrorCount=%d, want > 0", got)
	}
	if got := callbackCount.Load(); got == 0 {
		t.Fatalf("OnOutputError callback count=%d, want > 0", got)
	}
	if got := callbackService.Load(); got != 0 {
		t.Fatalf("OnOutputError service=%d, want 0", got)
	}
}

func TestServiceMux_CloseUnblocksAccept(t *testing.T) {
	_, server := serviceMuxPair()

	done := make(chan error, 1)
	go func() {
		_, _, err := server.AcceptStream()
		done <- err
	}()

	time.Sleep(100 * time.Millisecond)
	_ = server.Close()

	select {
	case err := <-done:
		if !errors.Is(err, ErrAcceptQueueClosed) && !errors.Is(err, ErrServiceMuxClosed) {
			t.Fatalf("AcceptStream err=%v, want %v or %v", err, ErrAcceptQueueClosed, ErrServiceMuxClosed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("AcceptStream did not unblock after Close")
	}
}

func TestServiceMux_OpenAfterConnCloseRecreates(t *testing.T) {
	mux := NewServiceMux(ServiceMuxConfig{})
	defer mux.Close()

	first, err := mux.OpenStream(0)
	if err != nil {
		t.Fatalf("first OpenStream failed: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}

	second, err := mux.OpenStream(0)
	if err != nil {
		t.Fatalf("second OpenStream failed: %v", err)
	}

	if first == second {
		t.Fatal("expected recreated stream instance, got same net.Conn")
	}
}

var _ net.Conn = (*KCPConn)(nil)
