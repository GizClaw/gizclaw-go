package gear

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/gearservice"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

func TestStoreOpsHelpers(t *testing.T) {
	server := &Server{}
	if _, err := server.store(); err == nil {
		t.Fatal("store should fail when store is nil")
	}
	if (&Server{}).peerRuntime(context.Background(), giznet.PublicKey{1}).Online {
		t.Fatal("zero peerRuntime should be offline")
	}
	if optionalGear(apitypes.Gear{PublicKey: "x"}, nil) == nil {
		t.Fatal("optionalGear should keep value")
	}
	if optionalGear(apitypes.Gear{}, errors.New("boom")) != nil {
		t.Fatal("optionalGear should drop error case")
	}
}

func TestStoreOpsRegisterValidation(t *testing.T) {
	server := &Server{
		Store: mustBadgerInMemory(t, nil),
	}

	_, err := server.register(context.Background(), giznet.PublicKey{}, gearservice.RegistrationRequest{})
	if err == nil || !strings.Contains(err.Error(), "empty public key") {
		t.Fatalf("empty public key err = %v", err)
	}

	publicKey := giznet.PublicKey{1}
	registered, err := server.register(context.Background(), publicKey, gearservice.RegistrationRequest{
		Device: apitypes.DeviceInfo{},
	})
	if err != nil {
		t.Fatalf("register without token error = %v", err)
	}
	if registered.Status != apitypes.GearStatusActive || registered.Role != apitypes.GearRoleUnspecified {
		t.Fatalf("registered gear = %+v", registered)
	}
}

func TestStoreOpsEnsureConnectedGearAndRegister(t *testing.T) {
	server := &Server{Store: mustBadgerInMemory(t, nil)}
	ctx := context.Background()
	publicKey := giznet.PublicKey{1}

	connected, err := server.EnsureConnectedGear(ctx, publicKey)
	if err != nil {
		t.Fatalf("EnsureConnectedGear error = %v", err)
	}
	if connected.Role != apitypes.GearRoleUnspecified || connected.Status != apitypes.GearStatusActive {
		t.Fatalf("connected gear = %+v", connected)
	}

	name := "demo"
	registered, err := server.register(ctx, publicKey, gearservice.RegistrationRequest{
		Device: apitypes.DeviceInfo{Name: &name},
	})
	if err != nil {
		t.Fatalf("register existing connected gear error = %v", err)
	}
	if registered.Role != apitypes.GearRoleUnspecified || registered.Status != apitypes.GearStatusActive {
		t.Fatalf("registered gear = %+v", registered)
	}
	if registered.Device.Name == nil || *registered.Device.Name != "demo" {
		t.Fatalf("registered device = %+v", registered.Device)
	}
}

func TestStoreOpsEnsureConnectedGearPreservesExisting(t *testing.T) {
	server := &Server{Store: mustBadgerInMemory(t, nil)}
	ctx := context.Background()
	publicKey := giznet.PublicKey{1}
	if _, err := server.SaveGear(ctx, apitypes.Gear{
		PublicKey:     publicKey.String(),
		Role:          apitypes.GearRoleAdmin,
		Status:        apitypes.GearStatusBlocked,
		Configuration: apitypes.Configuration{},
	}); err != nil {
		t.Fatalf("SaveGear error = %v", err)
	}

	got, err := server.EnsureConnectedGear(ctx, publicKey)
	if err != nil {
		t.Fatalf("EnsureConnectedGear error = %v", err)
	}
	if got.Role != apitypes.GearRoleAdmin || got.Status != apitypes.GearStatusBlocked {
		t.Fatalf("EnsureConnectedGear overwrote existing gear: %+v", got)
	}
}

func TestStoreOpsLoadAndSaveGear(t *testing.T) {
	server := &Server{Store: mustBadgerInMemory(t, nil)}
	publicKey := giznet.PublicKey{1}
	want := apitypes.Gear{
		PublicKey: publicKey.String(),
		Role:      apitypes.GearRoleGear,
		Status:    apitypes.GearStatusActive,
		Device: apitypes.DeviceInfo{
			Name: func() *string {
				value := "demo"
				return &value
			}(),
		},
		Configuration: apitypes.Configuration{},
	}

	got, err := server.SaveGear(context.Background(), want)
	if err != nil {
		t.Fatalf("SaveGear error = %v", err)
	}
	if got.PublicKey != want.PublicKey {
		t.Fatalf("SaveGear public key = %q, want %q", got.PublicKey, want.PublicKey)
	}

	loaded, err := server.LoadGear(context.Background(), publicKey)
	if err != nil {
		t.Fatalf("LoadGear error = %v", err)
	}
	if loaded.PublicKey != want.PublicKey || loaded.Role != want.Role || loaded.Status != want.Status {
		t.Fatalf("LoadGear = %+v", loaded)
	}
	if loaded.Device.Name == nil || *loaded.Device.Name != "demo" {
		t.Fatalf("LoadGear device name = %+v", loaded.Device.Name)
	}
}

func TestStoreOpsCanonicalizesPublicKeyText(t *testing.T) {
	server := &Server{Store: mustBadgerInMemory(t, nil)}
	ctx := context.Background()
	key, err := giznet.KeyFromHex(strings.Repeat("ab", giznet.KeySize))
	if err != nil {
		t.Fatalf("KeyFromHex error = %v", err)
	}
	hexKey := strings.Repeat("ab", giznet.KeySize)

	saved, err := server.SaveGear(ctx, apitypes.Gear{
		PublicKey:     hexKey,
		Role:          apitypes.GearRoleGear,
		Status:        apitypes.GearStatusActive,
		Configuration: apitypes.Configuration{},
	})
	if err != nil {
		t.Fatalf("SaveGear error = %v", err)
	}
	if saved.PublicKey != key.String() {
		t.Fatalf("SaveGear public key = %q, want canonical %q", saved.PublicKey, key.String())
	}

	loaded, err := server.LoadGear(ctx, key)
	if err != nil {
		t.Fatalf("LoadGear error = %v", err)
	}
	if loaded.PublicKey != key.String() {
		t.Fatalf("LoadGear public key = %q, want canonical %q", loaded.PublicKey, key.String())
	}
}

func TestStoreOpsLoadGearMissing(t *testing.T) {
	server := &Server{Store: mustBadgerInMemory(t, nil)}

	_, err := server.LoadGear(context.Background(), giznet.PublicKey{1})
	if !errors.Is(err, ErrGearNotFound) {
		t.Fatalf("LoadGear missing err = %v", err)
	}
}

func TestStoreOpsSaveGearRejectsInvalidGear(t *testing.T) {
	server := &Server{Store: mustBadgerInMemory(t, nil)}

	_, err := server.SaveGear(context.Background(), apitypes.Gear{})
	if err == nil || !strings.Contains(err.Error(), "empty public key") {
		t.Fatalf("SaveGear invalid err = %v", err)
	}

	_, err = server.SaveGear(context.Background(), apitypes.Gear{
		PublicKey:     "server",
		Role:          apitypes.GearRoleGear,
		Status:        apitypes.GearStatusActive,
		Configuration: apitypes.Configuration{},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid public key") {
		t.Fatalf("SaveGear invalid public key err = %v", err)
	}
}

func TestStoreOpsExists(t *testing.T) {
	server := &Server{Store: mustBadgerInMemory(t, nil)}
	publicKey := giznet.PublicKey{1}

	if exists, err := server.exists(context.Background(), publicKey); err != nil || exists {
		t.Fatalf("exists(missing) = %v, %v", exists, err)
	}

	if _, err := server.SaveGear(context.Background(), apitypes.Gear{
		PublicKey:     publicKey.String(),
		Role:          apitypes.GearRoleGear,
		Status:        apitypes.GearStatusActive,
		Configuration: apitypes.Configuration{},
	}); err != nil {
		t.Fatalf("SaveGear error = %v", err)
	}

	if exists, err := server.exists(context.Background(), publicKey); err != nil || !exists {
		t.Fatalf("exists(peer) = %v, %v", exists, err)
	}
}
