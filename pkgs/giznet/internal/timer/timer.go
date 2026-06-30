package timer

import (
	"sync"
	"time"
)

// Timer runs a callback at the most recently configured deadline.
// Reset and Close are safe to call concurrently.
type Timer struct {
	fn func()

	mu       sync.Mutex
	deadline time.Time
	armed    bool
	closed   bool

	wake      chan struct{}
	closeCh   chan struct{}
	closeOnce sync.Once
}

// New creates a timer that invokes fn whenever its deadline expires.
// The timer is initially disarmed; call Reset to schedule the first event.
func New(fn func()) *Timer {
	if fn == nil {
		panic("timer: nil callback")
	}

	t := &Timer{
		fn:      fn,
		wake:    make(chan struct{}, 1),
		closeCh: make(chan struct{}),
	}
	go t.loop()
	return t
}

// Reset schedules the callback at deadline. Passing a zero deadline disarms the timer.
// It returns false if the timer is closed.
func (t *Timer) Reset(deadline time.Time) bool {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return false
	}
	if deadline.IsZero() {
		t.deadline = time.Time{}
		t.armed = false
		t.mu.Unlock()
		t.signal()
		return true
	}
	t.deadline = deadline
	t.armed = true
	t.mu.Unlock()

	t.signal()
	return true
}

// Close stops the timer. It does not wait for an already running callback.
func (t *Timer) Close() {
	t.closeOnce.Do(func() {
		t.mu.Lock()
		t.closed = true
		t.armed = false
		t.mu.Unlock()
		close(t.closeCh)
	})
}

func (t *Timer) signal() {
	select {
	case t.wake <- struct{}{}:
	default:
	}
}

func (t *Timer) loop() {
	timer := time.NewTimer(time.Hour)
	stopTimer(timer)
	defer timer.Stop()

	for {
		deadline, ok := t.currentDeadline()
		if !ok {
			select {
			case <-t.wake:
				continue
			case <-t.closeCh:
				return
			}
		}

		resetTimer(timer, time.Until(deadline))

		select {
		case <-timer.C:
			if !t.consume(deadline) {
				continue
			}
			t.fn()
		case <-t.wake:
			continue
		case <-t.closeCh:
			return
		}
	}
}

func (t *Timer) currentDeadline() (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || !t.armed {
		return time.Time{}, false
	}
	return t.deadline, true
}

func (t *Timer) consume(deadline time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || !t.armed || !t.deadline.Equal(deadline) {
		return false
	}
	t.armed = false
	return true
}

func resetTimer(timer *time.Timer, delay time.Duration) {
	if delay < 0 {
		delay = 0
	}
	stopTimer(timer)
	timer.Reset(delay)
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
