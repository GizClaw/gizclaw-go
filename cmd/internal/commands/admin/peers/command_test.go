package peerscmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func TestSetFirmwareChannelMergesExistingConfig(t *testing.T) {
	original := openPeerConfigClient
	fake := &fakePeerConfigClient{
		getCfg: apitypes.Configuration{
			Certifications: &[]apitypes.GearCertification{{
				Type:      apitypes.GearCertificationType("certification"),
				Authority: apitypes.GearCertificationAuthority("ce"),
				Id:        "ce-001",
			}},
			Firmware: &apitypes.FirmwareConfig{Channel: ptrChannel("beta")},
		},
	}
	openPeerConfigClient = func(string) (peerConfigClient, error) {
		return fake, nil
	}
	defer func() { openPeerConfigClient = original }()

	cmd := NewCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"set-firmware-channel", "device-pk", "stable"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if fake.putCfg.Firmware == nil || fake.putCfg.Firmware.Channel == nil || *fake.putCfg.Firmware.Channel != "stable" {
		t.Fatalf("channel = %+v", fake.putCfg.Firmware)
	}
	if fake.putCfg.Certifications == nil || len(*fake.putCfg.Certifications) != 1 || (*fake.putCfg.Certifications)[0].Id != "ce-001" {
		t.Fatalf("certifications lost: %+v", fake.putCfg.Certifications)
	}

	var got apitypes.Configuration
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if got.Firmware == nil || got.Firmware.Channel == nil || *got.Firmware.Channel != "stable" {
		t.Fatalf("output channel = %+v", got.Firmware)
	}
	if got.Certifications == nil || len(*got.Certifications) != 1 {
		t.Fatalf("output certifications = %+v", got.Certifications)
	}
}

func TestPutConfigUsesFilePayload(t *testing.T) {
	original := openPeerConfigClient
	fake := &fakePeerConfigClient{}
	openPeerConfigClient = func(string) (peerConfigClient, error) {
		return fake, nil
	}
	defer func() { openPeerConfigClient = original }()

	file := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"firmware":{"channel":"beta"}}`)
	if err := os.WriteFile(file, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := NewCmd()
	cmd.SetArgs([]string{"put-config", "device-pk", "--file", file})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if fake.putCfg.Firmware == nil || fake.putCfg.Firmware.Channel == nil || *fake.putCfg.Firmware.Channel != "beta" {
		t.Fatalf("put config = %+v", fake.putCfg)
	}
}

func TestPeerCommandsReturnContextErrors(t *testing.T) {
	cases := [][]string{
		{"list"},
		{"get", "device-pk"},
		{"resolve-sn", "sn-001"},
		{"resolve-imei", "12345678", "000001"},
		{"approve", "device-pk", "gear"},
		{"block", "device-pk"},
		{"info", "device-pk"},
		{"config", "device-pk"},
		{"runtime", "device-pk"},
		{"ota", "device-pk"},
		{"list-by-label", "batch", "test"},
		{"list-by-certification", "certification", "ce", "ce-001"},
		{"list-by-firmware", "demo", "stable"},
		{"delete", "device-pk"},
		{"refresh", "device-pk"},
	}
	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			cmd := NewCmd()
			cmd.SetArgs(append(args, "--context", "__missing_context__"))
			if err := cmd.Execute(); err == nil {
				t.Fatal("Execute error = nil")
			}
		})
	}
}

func TestPeerCommandsUseClientOperations(t *testing.T) {
	restore := stubPeerCommandClients(t)
	defer restore()

	cases := [][]string{
		{"list"},
		{"get", "device-pk"},
		{"resolve-sn", "sn-001"},
		{"resolve-imei", "12345678", "000001"},
		{"approve", "device-pk", "gear"},
		{"block", "device-pk"},
		{"info", "device-pk"},
		{"config", "device-pk"},
		{"runtime", "device-pk"},
		{"ota", "device-pk"},
		{"list-by-label", "batch", "test"},
		{"list-by-certification", "certification", "ce", "ce-001"},
		{"list-by-firmware", "demo", "stable"},
		{"delete", "device-pk"},
		{"refresh", "device-pk"},
	}
	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			cmd := NewCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if out.Len() == 0 {
				t.Fatal("command produced no output")
			}
		})
	}
}

func stubPeerCommandClients(t *testing.T) func() {
	t.Helper()
	originalConnect := connectFromContext
	originalList := listPeers
	originalGet := getPeer
	originalResolveSN := resolvePeerBySN
	originalResolveIMEI := resolvePeerByIMEI
	originalApprove := approvePeer
	originalBlock := blockPeer
	originalInfo := getPeerInfo
	originalConfig := getPeerConfig
	originalPutConfig := putPeerConfig
	originalRuntime := getPeerRuntime
	originalOTA := getPeerOTA
	originalListLabel := listPeersByLabel
	originalListCertification := listPeersByCertification
	originalListFirmware := listPeersByFirmware
	originalDelete := deletePeer
	originalRefresh := refreshPeer

	registration := apitypes.Registration{
		PublicKey: "device-pk",
		Role:      apitypes.GearRoleGear,
		Status:    apitypes.GearStatusActive,
	}
	connectFromContext = func(string) (*gizclaw.Client, error) { return &gizclaw.Client{}, nil }
	listPeers = func(context.Context, *gizclaw.Client) ([]apitypes.Registration, error) {
		return []apitypes.Registration{registration}, nil
	}
	getPeer = func(context.Context, *gizclaw.Client, string) (apitypes.Registration, error) {
		return registration, nil
	}
	resolvePeerBySN = func(context.Context, *gizclaw.Client, string) (string, error) { return "device-pk", nil }
	resolvePeerByIMEI = func(context.Context, *gizclaw.Client, string, string) (string, error) {
		return "device-pk", nil
	}
	approvePeer = func(context.Context, *gizclaw.Client, string, apitypes.GearRole) (apitypes.Registration, error) {
		return registration, nil
	}
	blockPeer = func(context.Context, *gizclaw.Client, string) (apitypes.Registration, error) {
		return registration, nil
	}
	getPeerInfo = func(context.Context, *gizclaw.Client, string) (apitypes.DeviceInfo, error) {
		return apitypes.DeviceInfo{}, nil
	}
	getPeerConfig = func(context.Context, *gizclaw.Client, string) (apitypes.Configuration, error) {
		return apitypes.Configuration{}, nil
	}
	putPeerConfig = func(_ context.Context, _ *gizclaw.Client, _ string, cfg apitypes.Configuration) (apitypes.Configuration, error) {
		return cfg, nil
	}
	getPeerRuntime = func(context.Context, *gizclaw.Client, string) (apitypes.Runtime, error) {
		online := true
		return apitypes.Runtime{Online: online}, nil
	}
	getPeerOTA = func(context.Context, *gizclaw.Client, string) (apitypes.OTASummary, error) {
		return apitypes.OTASummary{}, nil
	}
	listPeersByLabel = func(context.Context, *gizclaw.Client, string, string) ([]apitypes.Registration, error) {
		return []apitypes.Registration{registration}, nil
	}
	listPeersByCertification = func(context.Context, *gizclaw.Client, apitypes.GearCertificationType, apitypes.GearCertificationAuthority, string) ([]apitypes.Registration, error) {
		return []apitypes.Registration{registration}, nil
	}
	listPeersByFirmware = func(context.Context, *gizclaw.Client, string, apitypes.GearFirmwareChannel) ([]apitypes.Registration, error) {
		return []apitypes.Registration{registration}, nil
	}
	deletePeer = func(context.Context, *gizclaw.Client, string) (apitypes.Registration, error) {
		return registration, nil
	}
	refreshPeer = func(context.Context, *gizclaw.Client, string) (adminservice.RefreshResult, error) {
		return adminservice.RefreshResult{Gear: apitypes.Gear{PublicKey: "device-pk"}}, nil
	}

	return func() {
		connectFromContext = originalConnect
		listPeers = originalList
		getPeer = originalGet
		resolvePeerBySN = originalResolveSN
		resolvePeerByIMEI = originalResolveIMEI
		approvePeer = originalApprove
		blockPeer = originalBlock
		getPeerInfo = originalInfo
		getPeerConfig = originalConfig
		putPeerConfig = originalPutConfig
		getPeerRuntime = originalRuntime
		getPeerOTA = originalOTA
		listPeersByLabel = originalListLabel
		listPeersByCertification = originalListCertification
		listPeersByFirmware = originalListFirmware
		deletePeer = originalDelete
		refreshPeer = originalRefresh
	}
}

type fakePeerConfigClient struct {
	getCfg apitypes.Configuration
	putCfg apitypes.Configuration
}

func (f *fakePeerConfigClient) GetPeerConfig(context.Context, string) (apitypes.Configuration, error) {
	return f.getCfg, nil
}

func (f *fakePeerConfigClient) PutPeerConfig(_ context.Context, _ string, cfg apitypes.Configuration) (apitypes.Configuration, error) {
	f.putCfg = cfg
	return cfg, nil
}

func (f *fakePeerConfigClient) Close() error { return nil }

func ptrChannel(value string) *apitypes.GearFirmwareChannel {
	channel := apitypes.GearFirmwareChannel(value)
	return &channel
}
