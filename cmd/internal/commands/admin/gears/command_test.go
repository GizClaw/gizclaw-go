package gearscmd

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	apitypes "github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func TestPutConfigMergesExistingConfig(t *testing.T) {
	original := openGearConfigClient
	fake := &fakeGearConfigClient{
		getCfg: apitypes.Configuration{
			Certifications: &[]apitypes.GearCertification{{
				Type:      apitypes.GearCertificationType("certification"),
				Authority: apitypes.GearCertificationAuthority("ce"),
				Id:        "ce-001",
			}},
			Firmware: &apitypes.FirmwareConfig{Channel: ptrChannel("beta")},
		},
	}
	openGearConfigClient = func(string) (gearConfigClient, error) {
		return fake, nil
	}
	defer func() { openGearConfigClient = original }()

	cmd := NewCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"put-config", "device-pk", "stable"})
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

type fakeGearConfigClient struct {
	getCfg apitypes.Configuration
	putCfg apitypes.Configuration
}

func (f *fakeGearConfigClient) GetGearConfig(context.Context, string) (apitypes.Configuration, error) {
	return f.getCfg, nil
}

func (f *fakeGearConfigClient) PutGearConfig(_ context.Context, _ string, cfg apitypes.Configuration) (apitypes.Configuration, error) {
	f.putCfg = cfg
	return cfg, nil
}

func (f *fakeGearConfigClient) Close() error { return nil }

func ptrChannel(value string) *apitypes.GearFirmwareChannel {
	channel := apitypes.GearFirmwareChannel(value)
	return &channel
}
