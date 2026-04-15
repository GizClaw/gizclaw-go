package gizclaw

import (
	"strings"
	"testing"

	"github.com/giztoy/giztoy-go/pkg/gizclaw/api/gearservice"
	"github.com/giztoy/giztoy-go/pkg/giznet"
)

func TestClientDialAndServeValidation(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		var client *Client
		if err := client.DialAndServe(giznet.PublicKey{}, "127.0.0.1:1"); err == nil || !strings.Contains(err.Error(), "nil client") {
			t.Fatalf("DialAndServe(nil) err = %v", err)
		}
	})

	t.Run("nil key pair", func(t *testing.T) {
		client := &Client{}
		if err := client.DialAndServe(giznet.PublicKey{}, "127.0.0.1:1"); err == nil || !strings.Contains(err.Error(), "nil key pair") {
			t.Fatalf("DialAndServe(nil key pair) err = %v", err)
		}
	})

	t.Run("empty server addr", func(t *testing.T) {
		keyPair, err := giznet.GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair error = %v", err)
		}
		client := &Client{KeyPair: keyPair}
		if err := client.DialAndServe(giznet.PublicKey{}, ""); err == nil || !strings.Contains(err.Error(), "empty server addr") {
			t.Fatalf("DialAndServe(empty addr) err = %v", err)
		}
	})

	t.Run("already started", func(t *testing.T) {
		keyPair, err := giznet.GenerateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair error = %v", err)
		}
		client := &Client{KeyPair: keyPair, listener: &giznet.Listener{}}
		if err := client.DialAndServe(giznet.PublicKey{}, "127.0.0.1:1"); err == nil || !strings.Contains(err.Error(), "already started") {
			t.Fatalf("DialAndServe(already started) err = %v", err)
		}
	})
}

func TestClientServePeerPublicValidation(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		var client *Client
		if err := client.servePeerPublic(); err == nil || !strings.Contains(err.Error(), "nil client") {
			t.Fatalf("servePeerPublic(nil) err = %v", err)
		}
	})

	t.Run("disconnected client", func(t *testing.T) {
		client := &Client{}
		if err := client.servePeerPublic(); err == nil || !strings.Contains(err.Error(), "not connected") {
			t.Fatalf("servePeerPublic(disconnected) err = %v", err)
		}
	})
}

func TestClientAccessorsAndConversions(t *testing.T) {
	keyPair, err := giznet.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error = %v", err)
	}

	client := &Client{KeyPair: keyPair, serverPK: keyPair.Public}

	if got := client.ServerPublicKey(); got != keyPair.Public {
		t.Fatalf("ServerPublicKey() = %v, want %v", got, keyPair.Public)
	}

	peerClient, err := client.PeerPublicClient()
	if err != nil {
		t.Fatalf("PeerPublicClient() error = %v", err)
	}
	if peerClient == nil {
		t.Fatal("PeerPublicClient() returned nil client")
	}

	if _, err := client.RPCClient(); err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("RPCClient() err = %v", err)
	}

	name := "main"
	device := gearservice.DeviceInfo{
		Sn: func() *string {
			v := "sn-1"
			return &v
		}(),
		Hardware: &gearservice.HardwareInfo{
			Manufacturer:   func() *string { v := "Acme"; return &v }(),
			Model:          func() *string { v := "M1"; return &v }(),
			Depot:          func() *string { v := "demo-main"; return &v }(),
			FirmwareSemver: func() *string { v := "1.2.3"; return &v }(),
			Imeis: &[]gearservice.GearIMEI{{
				Name:   &name,
				Tac:    "12345678",
				Serial: "0000001",
			}},
			Labels: &[]gearservice.GearLabel{{
				Key:   "batch",
				Value: "cn-east",
			}},
		},
	}

	info := gearDeviceToPeerRefreshInfo(device)
	if info.Manufacturer == nil || *info.Manufacturer != "Acme" {
		t.Fatalf("gearDeviceToPeerRefreshInfo() = %+v", info)
	}

	identifiers := gearDeviceToPeerRefreshIdentifiers(device)
	if identifiers.Sn == nil || *identifiers.Sn != "sn-1" {
		t.Fatalf("gearDeviceToPeerRefreshIdentifiers().Sn = %+v", identifiers.Sn)
	}
	if identifiers.Imeis == nil || len(*identifiers.Imeis) != 1 || (*identifiers.Imeis)[0].Tac != "12345678" {
		t.Fatalf("gearDeviceToPeerRefreshIdentifiers().Imeis = %+v", identifiers.Imeis)
	}
	if identifiers.Labels == nil || len(*identifiers.Labels) != 1 || (*identifiers.Labels)[0].Value != "cn-east" {
		t.Fatalf("gearDeviceToPeerRefreshIdentifiers().Labels = %+v", identifiers.Labels)
	}

	version := gearDeviceToPeerRefreshVersion(device)
	if version.Depot == nil || *version.Depot != "demo-main" || version.FirmwareSemver == nil || *version.FirmwareSemver != "1.2.3" {
		t.Fatalf("gearDeviceToPeerRefreshVersion() = %+v", version)
	}

	imei := gearToPeerGearIMEI(gearservice.GearIMEI{Name: &name, Tac: "87654321", Serial: "0000009"})
	if imei.Name == nil || *imei.Name != "main" || imei.Tac != "87654321" || imei.Serial != "0000009" {
		t.Fatalf("gearToPeerGearIMEI() = %+v", imei)
	}

	label := gearToPeerGearLabel(gearservice.GearLabel{Key: "batch", Value: "cn-west"})
	if label.Key != "batch" || label.Value != "cn-west" {
		t.Fatalf("gearToPeerGearLabel() = %+v", label)
	}
}
