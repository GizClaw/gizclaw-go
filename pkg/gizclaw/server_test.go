package gizclaw

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gear"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
	"github.com/GizClaw/gizclaw-go/pkg/store/depotstore"
)

type testGiznetSecurityPolicy struct {
	allowService func(giznet.PublicKey, uint64) bool
}

func (p testGiznetSecurityPolicy) AllowPeer(giznet.PublicKey) bool {
	return true
}

func (p testGiznetSecurityPolicy) AllowService(pk giznet.PublicKey, service uint64) bool {
	if p.allowService == nil {
		return service == 0
	}
	return p.allowService(pk, service)
}

func TestServerListenRequiresGearStore(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	server := &Server{KeyPair: keyPair, DepotStore: depotstore.Dir(t.TempDir())}
	err = server.Listen()
	if err == nil || !strings.Contains(err.Error(), "nil gear store") {
		t.Fatalf("Listen error = %v, want nil gear store", err)
	}
}

func TestServerListenRequiresDepotStore(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	server := &Server{KeyPair: keyPair, GearStore: mustBadgerInMemory(t, nil)}
	err = server.Listen()
	if err == nil || !strings.Contains(err.Error(), "nil depot store") {
		t.Fatalf("Listen error = %v, want nil depot store", err)
	}
}

func TestServerListenValidatesReceiverAndKeyPair(t *testing.T) {
	t.Run("nil server", func(t *testing.T) {
		var server *Server
		if err := server.Listen(); err == nil || !strings.Contains(err.Error(), "nil server") {
			t.Fatalf("Listen() err = %v", err)
		}
	})

	t.Run("nil key pair", func(t *testing.T) {
		server := &Server{}
		if err := server.Listen(); err == nil || !strings.Contains(err.Error(), "nil key pair") {
			t.Fatalf("Listen() nil key pair err = %v", err)
		}
	})
}

func TestServerServeReturnsNilAfterClose(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	server := &Server{
		KeyPair:    keyPair,
		ListenAddr: "127.0.0.1:0",
		GearStore:  mustBadgerInMemory(t, nil),
		DepotStore: depotstore.Dir(t.TempDir()),
	}
	if err := server.Listen(); err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve()
	}()

	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Serve() after Close() error = %v, want nil", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve() did not return after Close()")
	}
}

func TestServerPublicKeyAndPeerServiceAccessors(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	service := &PeerService{}
	server := &Server{KeyPair: keyPair, peerService: service}
	if got := server.PublicKey(); got != keyPair.Public {
		t.Fatalf("PublicKey() = %v, want %v", got, keyPair.Public)
	}
	if got := server.PeerService(); got != service {
		t.Fatalf("PeerService() = %v, want %v", got, service)
	}
}

func TestServerSecurityPolicyAllowServiceUsesGearPolicy(t *testing.T) {
	var nilServer *Server
	if (*ServerSecurityPolicy)(nilServer).AllowService(giznet.PublicKey{}, ServiceRPC) {
		t.Fatal("nil server should deny all services")
	}

	gearKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair gear error = %v", err)
	}
	adminKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair admin error = %v", err)
	}
	gearsServer := &gear.Server{Store: mustBadgerInMemory(t, nil)}
	if _, err := gearsServer.SaveGear(context.Background(), apitypes.Gear{
		PublicKey:     gearKey.Public.String(),
		Role:          apitypes.GearRoleGear,
		Status:        apitypes.GearStatusActive,
		Device:        apitypes.DeviceInfo{},
		Configuration: apitypes.Configuration{},
	}); err != nil {
		t.Fatalf("SaveGear gear error = %v", err)
	}
	if _, err := gearsServer.SaveGear(context.Background(), apitypes.Gear{
		PublicKey:     adminKey.Public.String(),
		Role:          apitypes.GearRoleAdmin,
		Status:        apitypes.GearStatusActive,
		Device:        apitypes.DeviceInfo{},
		Configuration: apitypes.Configuration{},
	}); err != nil {
		t.Fatalf("SaveGear admin error = %v", err)
	}
	server := &Server{manager: NewManager(gearsServer)}
	policy := (*ServerSecurityPolicy)(server)
	if !policy.AllowService(gearKey.Public, ServiceRPC) {
		t.Fatal("gear should allow rpc")
	}
	if !policy.AllowService(gearKey.Public, ServiceServerPublic) {
		t.Fatal("gear should allow server public")
	}
	if policy.AllowService(gearKey.Public, ServiceAdmin) {
		t.Fatal("non-admin gear should not allow admin")
	}
	if !policy.AllowService(adminKey.Public, ServiceAdmin) {
		t.Fatal("active admin gear should allow admin")
	}
	configuredKey, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair configured error = %v", err)
	}
	server.AdminPublicKey = configuredKey.Public.String()
	if !policy.AllowService(configuredKey.Public, ServiceAdmin) {
		t.Fatal("configured admin public key should allow admin")
	}
	server.AdminPublicKey = "not-a-public-key"
	if policy.AllowService(configuredKey.Public, ServiceAdmin) {
		t.Fatal("invalid configured admin public key should not allow admin")
	}
}

func TestServerPeerEventHandlerMarksManagerOffline(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}
	server := &Server{manager: &Manager{}}
	server.manager.SetPeerUp(keyPair.Public, &giznet.Conn{})

	(*serverPeerEventHandler)(server).HandlePeerEvent(giznet.PeerEvent{PublicKey: keyPair.Public, State: giznet.PeerStateOffline})
	runtime := server.manager.PeerRuntime(context.Background(), keyPair.Public)
	if runtime.Online || !runtime.LastSeenAt.IsZero() {
		t.Fatalf("runtime after offline event = %+v", runtime)
	}
}

func TestResolveGearTarget(t *testing.T) {
	ctx := context.Background()
	store := mustBadgerInMemory(t, nil)
	gearsServer := &gear.Server{Store: store}

	saveGear := func(t *testing.T, publicKey string, device apitypes.DeviceInfo, config apitypes.Configuration) {
		t.Helper()
		if _, err := gearsServer.SaveGear(ctx, apitypes.Gear{
			PublicKey:     publicKey,
			Role:          apitypes.GearRoleGear,
			Status:        apitypes.GearStatusActive,
			Device:        device,
			Configuration: config,
		}); err != nil {
			t.Fatalf("SaveGear(%s) error = %v", publicKey, err)
		}
	}

	saveGear(t, "missing-depot", apitypes.DeviceInfo{}, apitypes.Configuration{
		Firmware: &apitypes.FirmwareConfig{Channel: func() *apitypes.GearFirmwareChannel {
			ch := apitypes.GearFirmwareChannel("stable")
			return &ch
		}()},
	})
	if _, _, err := resolveGearTarget(ctx, gearsServer, "missing-depot"); err == nil || !strings.Contains(err.Error(), "missing depot") {
		t.Fatalf("resolveGearTarget(missing depot) err = %v", err)
	}

	saveGear(t, "missing-channel", apitypes.DeviceInfo{
		Hardware: &apitypes.HardwareInfo{Depot: func() *string { v := "demo-main"; return &v }()},
	}, apitypes.Configuration{})
	if _, _, err := resolveGearTarget(ctx, gearsServer, "missing-channel"); err == nil || !strings.Contains(err.Error(), "missing channel") {
		t.Fatalf("resolveGearTarget(missing channel) err = %v", err)
	}

	if _, _, err := resolveGearTarget(ctx, gearsServer, "missing-gear"); !errors.Is(err, gear.ErrGearNotFound) {
		t.Fatalf("resolveGearTarget(missing gear) err = %v", err)
	}

	saveGear(t, "valid", apitypes.DeviceInfo{
		Hardware: &apitypes.HardwareInfo{Depot: func() *string { v := "demo-main"; return &v }()},
	}, apitypes.Configuration{
		Firmware: &apitypes.FirmwareConfig{Channel: func() *apitypes.GearFirmwareChannel {
			ch := apitypes.GearFirmwareChannel("stable")
			return &ch
		}()},
	})
	depot, channel, err := resolveGearTarget(ctx, gearsServer, "valid")
	if err != nil {
		t.Fatalf("resolveGearTarget(valid) err = %v", err)
	}
	if depot != "demo-main" || channel != "stable" {
		t.Fatalf("resolveGearTarget(valid) = (%q, %q)", depot, channel)
	}
}
