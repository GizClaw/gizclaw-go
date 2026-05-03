package core

import (
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/giznet/internal/noise"
)

func TestOptions(t *testing.T) {
	key, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	cfg := FullSocketConfig()
	cfg.RecvBufSize = 2 * 1024 * 1024
	cfg.SendBufSize = 2 * 1024 * 1024

	u, err := NewUDP(
		key,
		WithBindAddr("127.0.0.1:0"),
		WithAllowUnknown(true),
		WithRawChanSize(17),
		WithSocketConfig(cfg),
		WithServiceMuxConfig(ServiceMuxConfig{
			OnNewService: func(peer noise.PublicKey, service uint64) bool {
				return service == 1
			},
		}),
	)
	if err != nil {
		t.Fatalf("NewUDP failed: %v", err)
	}

	if cap(u.decryptChan) != 17 {
		t.Fatalf("decryptChan cap=%d, want 17", cap(u.decryptChan))
	}
	if u.socketConfig != cfg {
		t.Fatalf("socketConfig mismatch: got=%+v want=%+v", u.socketConfig, cfg)
	}
	if u.serviceMuxConfig.OnNewService == nil {
		t.Fatal("serviceMuxConfig should be injected by WithServiceMuxConfig")
	}

	if err := u.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestWithServiceMuxConfigOption(t *testing.T) {
	key, err := noise.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	wantCfg := ServiceMuxConfig{
		OnNewService: func(peer noise.PublicKey, service uint64) bool {
			return service == 9
		},
	}

	u, err := NewUDP(
		key,
		WithBindAddr("127.0.0.1:0"),
		WithServiceMuxConfig(wantCfg),
	)
	if err != nil {
		t.Fatalf("NewUDP failed: %v", err)
	}
	defer u.Close()

	if u.serviceMuxConfig.OnNewService == nil {
		t.Fatal("serviceMuxConfig.OnNewService is nil")
	}
	if u.serviceMuxConfig.OnNewService(noise.PublicKey{}, 9) != wantCfg.OnNewService(noise.PublicKey{}, 9) {
		t.Fatal("serviceMuxConfig.OnNewService not applied")
	}
}
