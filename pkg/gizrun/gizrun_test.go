package gizrun

import (
	"log/slog"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizrun/internal/log/sink"
)

func TestStart(t *testing.T) {
	resetInitHooksForTest(t)

	Start()
	t.Cleanup(Stop)

	if stats := sink.CurrentStats(); !stats.Running {
		t.Fatalf("log sink is not running: %+v", stats)
	}
	slog.Info("init starts async sink")
}

func TestStartCanRunAfterStop(t *testing.T) {
	resetInitHooksForTest(t)
	InitAt(0, func() error { return nil })

	Start()
	Stop()
	Start()
	t.Cleanup(Stop)

	if stats := sink.CurrentStats(); !stats.Running {
		t.Fatalf("log sink is not running after restart: %+v", stats)
	}
}

func TestInitAt(t *testing.T) {
	resetInitHooksForTest(t)

	var got []int
	InitAt(0x20, func() error { got = append(got, 2); return nil })
	InitAt(0x10, func() error { got = append(got, 1); return nil })
	InitAt(0x20, func() error { got = append(got, 3); return nil })
	InitAt(0x30, nil)

	Start()
	t.Cleanup(Stop)

	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("init hook count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("init hook order = %#v, want %#v", got, want)
		}
	}
}

func resetInitHooksForTest(t *testing.T) {
	t.Helper()
	initHooks.next = 1
	initHooks.hooks = []initHook{{
		seq: 0,
		fn: func() error {
			if flags.registered {
				return nil
			}
			flags.registered = true
			return nil
		},
	}}
}
