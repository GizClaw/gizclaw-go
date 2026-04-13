package gizclaw_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/giztoy/giztoy-go/cmd/internal/integration/testutil"
	"github.com/giztoy/giztoy-go/cmd/internal/server"
	"github.com/giztoy/giztoy-go/cmd/internal/stores"
	"github.com/giztoy/giztoy-go/pkg/gears"
	"github.com/giztoy/giztoy-go/pkg/gizclaw"
)

func TestDialAndPing(t *testing.T) {
	root := t.TempDir()
	ts := startTestServerWithConfig(t, server.Config{
		ListenAddr: testutil.AllocateUDPAddr(t),
		Stores: map[string]stores.Config{
			"mem": {Kind: stores.KindKeyValue, Backend: "memory"},
			"fw":  {Kind: stores.KindFS, Backend: "filesystem", Dir: filepath.Join(root, "firmware")},
		},
		Gears: server.GearsConfig{
			Store: "mem",
			RegistrationTokens: map[string]gears.RegistrationToken{
				"device_default": {Role: gears.GearRoleDevice},
			},
		},
		Depots: server.DepotsConfig{Store: "fw"},
	})

	c := newTestClient(t, ts)

	var ping *gizclaw.PingResult
	var pingErr error
	if err := testutil.WaitUntil(testutil.ReadyTimeout, func() error {
		ping, pingErr = c.Ping()
		return pingErr
	}); err != nil {
		t.Fatalf("Ping err=%v", pingErr)
	}
	if ping.ServerTime.IsZero() {
		t.Fatal("ServerTime is zero")
	}
	if ping.RTT <= 0 {
		t.Fatalf("RTT=%v", ping.RTT)
	}
	if ping.ClockDiff > time.Second || ping.ClockDiff < -time.Second {
		t.Fatalf("ClockDiff=%v (too large for localhost)", ping.ClockDiff)
	}
}
