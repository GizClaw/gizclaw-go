package kcp

import (
	"io"
	"sync"
	"testing"
	"time"
)

func readExactWithTimeout(t *testing.T, r io.Reader, n int, timeout time.Duration) []byte {
	t.Helper()

	buf := make([]byte, n)
	errCh := make(chan error, 1)
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

type lcg struct {
	mu    sync.Mutex
	state uint64
}

func newLCG(seed uint64) *lcg {
	return &lcg{state: seed}
}

func (l *lcg) next() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.state = l.state*6364136223846793005 + 1442695040888963407
	return float64((l.state>>33)&0x7FFFFFFF) / float64(0x7FFFFFFF)
}

func (l *lcg) shouldDrop(rate float64) bool {
	if rate <= 0 {
		return false
	}
	return l.next() < rate
}
