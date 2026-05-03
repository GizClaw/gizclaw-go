package core

import (
	"context"
	"errors"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/giznet/internal/noise"
)

func TestDialMissingLocalKey(t *testing.T) {
	transport := NewMockTransport("test")
	defer transport.Close()

	remotePK := noise.PublicKey{}
	copy(remotePK[:], []byte("12345678901234567890123456789012"))

	ctx := context.Background()
	_, err := Dial(ctx, transport, transport.LocalAddr(), remotePK, nil)
	if err != ErrMissingLocalKey {
		t.Errorf("Dial() error = %v, want ErrMissingLocalKey", err)
	}
}

func TestDialMissingTransport(t *testing.T) {
	key, _ := noise.GenerateKeyPair()
	remotePK := noise.PublicKey{}
	copy(remotePK[:], []byte("12345678901234567890123456789012"))

	ctx := context.Background()
	_, err := Dial(ctx, nil, NewMockAddr("test"), remotePK, key)
	if err != ErrMissingTransport {
		t.Errorf("Dial() error = %v, want ErrMissingTransport", err)
	}
}

func TestDialMissingRemotePK(t *testing.T) {
	key, _ := noise.GenerateKeyPair()
	transport := NewMockTransport("test")
	defer transport.Close()

	ctx := context.Background()
	_, err := Dial(ctx, transport, transport.LocalAddr(), noise.PublicKey{}, key)
	if err != ErrMissingRemotePK {
		t.Errorf("Dial() error = %v, want ErrMissingRemotePK", err)
	}
}

func TestDialMissingRemoteAddr(t *testing.T) {
	key, _ := noise.GenerateKeyPair()
	transport := NewMockTransport("test")
	defer transport.Close()

	remotePK := noise.PublicKey{}
	copy(remotePK[:], []byte("12345678901234567890123456789012"))

	ctx := context.Background()
	_, err := Dial(ctx, transport, nil, remotePK, key)
	if err != ErrMissingRemoteAddr {
		t.Errorf("Dial() error = %v, want ErrMissingRemoteAddr", err)
	}
}

func TestIsTimeoutError(t *testing.T) {
	if isTimeoutError(nil) {
		t.Error("isTimeoutError(nil) should be false")
	}
	if isTimeoutError(errors.New("some error")) {
		t.Error("isTimeoutError(regular error) should be false")
	}

	timeoutErr := &timeoutTestError{timeout: true}
	if !isTimeoutError(timeoutErr) {
		t.Error("isTimeoutError(timeout error) should be true")
	}

	nonTimeoutErr := &timeoutTestError{timeout: false}
	if isTimeoutError(nonTimeoutErr) {
		t.Error("isTimeoutError(non-timeout error) should be false")
	}
}

type timeoutTestError struct {
	timeout bool
}

func (e *timeoutTestError) Error() string {
	return "test timeout error"
}

func (e *timeoutTestError) Timeout() bool {
	return e.timeout
}
