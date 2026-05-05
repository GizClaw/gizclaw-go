package gizclaw

import (
	"context"
	"errors"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gear"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

func TestManagerSetPeerDownDeletesPeer(t *testing.T) {
	manager := &Manager{}
	key := giznet.PublicKey{1}
	conn := &giznet.Conn{}

	manager.SetPeerUp(key, conn)
	if runtime := manager.PeerRuntime(context.Background(), key); !runtime.Online {
		t.Fatalf("peer should be online before removal: %+v", runtime)
	}

	manager.SetPeerDown(key)
	if _, ok := manager.Peer(key); ok {
		t.Fatal("peer should be removed")
	}
	if runtime := manager.PeerRuntime(context.Background(), key); runtime.Online || !runtime.LastSeenAt.IsZero() {
		t.Fatalf("runtime after removal = %+v", runtime)
	}
}

func TestManagerSetPeerUpReplacesConnection(t *testing.T) {
	manager := &Manager{}
	key := giznet.PublicKey{1}
	oldConn := &giznet.Conn{}
	newConn := &giznet.Conn{}

	manager.SetPeerUp(key, oldConn)
	manager.SetPeerUp(key, newConn)

	got, ok := manager.Peer(key)
	if !ok || got != newConn {
		t.Fatalf("ActivePeer after replacement = %v, %v", got, ok)
	}
	if runtime := manager.PeerRuntime(context.Background(), key); !runtime.Online || !runtime.LastSeenAt.IsZero() {
		t.Fatalf("runtime after replacement = %+v", runtime)
	}
}

func TestManagerSetPeerUpAndDownUpdatesRuntime(t *testing.T) {
	manager := &Manager{}
	key := giznet.PublicKey{1}
	conn := &giznet.Conn{}

	manager.SetPeerUp(key, conn)
	if got, ok := manager.Peer(key); !ok || got != conn {
		t.Fatalf("active peer after set = %v, %v", got, ok)
	}
	if runtime := manager.PeerRuntime(context.Background(), key); !runtime.Online || !runtime.LastSeenAt.IsZero() {
		t.Fatalf("runtime after set = %+v, want online with no peer info", runtime)
	}

	manager.SetPeerDown(key)
	if runtime := manager.PeerRuntime(context.Background(), key); runtime.Online || !runtime.LastSeenAt.IsZero() {
		t.Fatalf("runtime after remove = %+v", runtime)
	}
}

func TestManagerEnsureGearCreatesDefaultUnspecifiedGear(t *testing.T) {
	service := &gear.Server{Store: mustBadgerInMemory(t, nil)}
	manager := NewManager(service)
	ctx := context.Background()
	key := giznet.PublicKey{1}

	created, err := manager.EnsureGear(ctx, key)
	if err != nil {
		t.Fatalf("EnsureGear error = %v", err)
	}
	if created.PublicKey != key.String() {
		t.Fatalf("PublicKey = %q, want %q", created.PublicKey, key.String())
	}
	if created.Role != apitypes.GearRoleUnspecified {
		t.Fatalf("Role = %q, want unspecified", created.Role)
	}
	if created.Status != apitypes.GearStatusActive {
		t.Fatalf("Status = %q, want active", created.Status)
	}
	if created.AutoRegistered == nil || !*created.AutoRegistered {
		t.Fatalf("AutoRegistered = %v, want true", created.AutoRegistered)
	}

	loaded, err := service.LoadGear(ctx, key.String())
	if err != nil {
		t.Fatalf("LoadGear error = %v", err)
	}
	if loaded.Role != apitypes.GearRoleUnspecified || loaded.Status != apitypes.GearStatusActive {
		t.Fatalf("loaded gear = %+v", loaded)
	}
}

func TestManagerEnsureGearPreservesExistingGear(t *testing.T) {
	service := &gear.Server{Store: mustBadgerInMemory(t, nil)}
	manager := NewManager(service)
	ctx := context.Background()
	key := giznet.PublicKey{1}
	if _, err := service.SaveGear(ctx, apitypes.Gear{
		PublicKey:     key.String(),
		Role:          apitypes.GearRoleAdmin,
		Status:        apitypes.GearStatusBlocked,
		Device:        apitypes.DeviceInfo{},
		Configuration: apitypes.Configuration{},
	}); err != nil {
		t.Fatalf("SaveGear error = %v", err)
	}

	got, err := manager.EnsureGear(ctx, key)
	if err != nil {
		t.Fatalf("EnsureGear error = %v", err)
	}
	if got.Role != apitypes.GearRoleAdmin || got.Status != apitypes.GearStatusBlocked {
		t.Fatalf("EnsureGear overwrote existing gear: %+v", got)
	}
}

func TestManagerRefreshDeviceErrors(t *testing.T) {
	service := &gear.Server{Store: mustBadgerInMemory(t, nil)}
	manager := NewManager(service)
	ctx := context.Background()
	missingKey := giznet.PublicKey{1}
	deviceKey := giznet.PublicKey{2}

	if _, _, err := manager.RefreshGear(ctx, missingKey); !errors.Is(err, gear.ErrGearNotFound) {
		t.Fatalf("RefreshGear missing err = %v", err)
	}

	if _, err := service.SaveGear(ctx, apitypes.Gear{
		PublicKey:     deviceKey.String(),
		Role:          apitypes.GearRoleUnspecified,
		Status:        apitypes.GearStatusUnspecified,
		Device:        apitypes.DeviceInfo{},
		Configuration: apitypes.Configuration{},
	}); err != nil {
		t.Fatalf("SaveGear error: %v", err)
	}

	if _, online, err := manager.RefreshGear(ctx, deviceKey); !errors.Is(err, ErrDeviceOffline) {
		t.Fatalf("RefreshGear offline err = %v", err)
	} else if online {
		t.Fatal("offline RefreshGear should report online=false")
	}
}

func TestApplyPeerRefreshIdentifiersSkipsUnchangedCollections(t *testing.T) {
	name := "primary"
	sn := "sn-1"
	gear := apitypes.Gear{
		Device: apitypes.DeviceInfo{
			Sn: &sn,
			Hardware: &apitypes.HardwareInfo{
				Imeis: &[]apitypes.GearIMEI{{
					Name:   &name,
					Tac:    "12345678",
					Serial: "0000001",
				}},
				Labels: &[]apitypes.GearLabel{{
					Key:   "batch",
					Value: "cn-east",
				}},
			},
		},
	}
	identifiers := apitypes.RefreshIdentifiers{
		Sn: &sn,
		Imeis: &[]apitypes.GearIMEI{{
			Name:   &name,
			Tac:    "12345678",
			Serial: "0000001",
		}},
		Labels: &[]apitypes.GearLabel{{
			Key:   "batch",
			Value: "cn-east",
		}},
	}

	var updatedFields []string
	applyPeerRefreshIdentifiers(&gear, identifiers, &updatedFields)

	if len(updatedFields) != 0 {
		t.Fatalf("applyPeerRefreshIdentifiers() updatedFields = %v, want none", updatedFields)
	}
}

func TestApplyPeerRefreshIdentifiersUpdatesChangedCollections(t *testing.T) {
	name := "primary"
	nextName := "secondary"
	gear := apitypes.Gear{
		Device: apitypes.DeviceInfo{
			Hardware: &apitypes.HardwareInfo{
				Imeis: &[]apitypes.GearIMEI{{
					Name:   &name,
					Tac:    "12345678",
					Serial: "0000001",
				}},
				Labels: &[]apitypes.GearLabel{{
					Key:   "batch",
					Value: "cn-east",
				}},
			},
		},
	}
	identifiers := apitypes.RefreshIdentifiers{
		Imeis: &[]apitypes.GearIMEI{{
			Name:   &nextName,
			Tac:    "87654321",
			Serial: "0000009",
		}},
		Labels: &[]apitypes.GearLabel{{
			Key:   "batch",
			Value: "cn-west",
		}},
	}

	var updatedFields []string
	applyPeerRefreshIdentifiers(&gear, identifiers, &updatedFields)

	if len(updatedFields) != 2 {
		t.Fatalf("applyPeerRefreshIdentifiers() updatedFields = %v, want 2 entries", updatedFields)
	}
	if gear.Device.Hardware == nil || gear.Device.Hardware.Imeis == nil || (*gear.Device.Hardware.Imeis)[0].Tac != "87654321" {
		t.Fatalf("IMEIs not updated: %+v", gear.Device.Hardware)
	}
	if gear.Device.Hardware.Labels == nil || (*gear.Device.Hardware.Labels)[0].Value != "cn-west" {
		t.Fatalf("labels not updated: %+v", gear.Device.Hardware)
	}
}

func TestIsPeerDisconnectedError(t *testing.T) {
	t.Run("closed connection errors are offline", func(t *testing.T) {
		if !isPeerDisconnectedError(errors.New("gizhttp: read response: kcp: conn closed: local")) {
			t.Fatal("conn closed error should be treated as disconnected")
		}
	})

	t.Run("generic read response errors stay online", func(t *testing.T) {
		if isPeerDisconnectedError(errors.New("gizhttp: read response: malformed HTTP response")) {
			t.Fatal("generic read response error should not be treated as disconnected")
		}
	})
}
