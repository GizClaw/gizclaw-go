package voice

import (
	"context"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

func TestServerVoiceCRUDAndFilters(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	srv := &Server{
		Store: kv.NewMemory(nil),
		Now: func() time.Time {
			return now
		},
	}
	body := adminservice.VoiceUpsert{
		Id:     "manual:voice-1",
		Source: apitypes.VoiceSourceManual,
		Provider: apitypes.VoiceProvider{
			Kind: apitypes.VoiceProviderKind("openai-tenant"),
			Name: "main",
		},
	}
	createdResp, err := srv.CreateVoice(ctx, adminservice.CreateVoiceRequestObject{Body: &body})
	if err != nil {
		t.Fatalf("CreateVoice() error = %v", err)
	}
	created, ok := createdResp.(adminservice.CreateVoice200JSONResponse)
	if !ok {
		t.Fatalf("CreateVoice() response = %#v", createdResp)
	}
	if created.CreatedAt != now || created.UpdatedAt != now {
		t.Fatalf("created timestamps = %s/%s, want %s", created.CreatedAt, created.UpdatedAt, now)
	}

	source := adminservice.VoiceSource(apitypes.VoiceSourceManual)
	providerKind := adminservice.VoiceProviderKind("openai-tenant")
	providerName := "main"
	listResp, err := srv.ListVoices(ctx, adminservice.ListVoicesRequestObject{
		Params: adminservice.ListVoicesParams{
			ProviderKind: &providerKind,
			ProviderName: &providerName,
			Source:       &source,
		},
	})
	if err != nil {
		t.Fatalf("ListVoices() error = %v", err)
	}
	listed, ok := listResp.(adminservice.ListVoices200JSONResponse)
	if !ok {
		t.Fatalf("ListVoices() response = %#v", listResp)
	}
	if len(listed.Items) != 1 || listed.Items[0].Id != "manual:voice-1" {
		t.Fatalf("ListVoices() items = %#v", listed.Items)
	}

	description := "updated"
	body.Description = &description
	putResp, err := srv.PutVoice(ctx, adminservice.PutVoiceRequestObject{Id: "manual:voice-1", Body: &body})
	if err != nil {
		t.Fatalf("PutVoice() error = %v", err)
	}
	updated, ok := putResp.(adminservice.PutVoice200JSONResponse)
	if !ok {
		t.Fatalf("PutVoice() response = %#v", putResp)
	}
	if updated.Description == nil || *updated.Description != description {
		t.Fatalf("PutVoice() description = %#v", updated.Description)
	}
	if updated.CreatedAt != now || updated.UpdatedAt != now {
		t.Fatalf("updated timestamps = %s/%s, want %s", updated.CreatedAt, updated.UpdatedAt, now)
	}

	getResp, err := srv.GetVoice(ctx, adminservice.GetVoiceRequestObject{Id: "manual:voice-1"})
	if err != nil {
		t.Fatalf("GetVoice() error = %v", err)
	}
	if _, ok := getResp.(adminservice.GetVoice200JSONResponse); !ok {
		t.Fatalf("GetVoice() response = %#v", getResp)
	}

	deleteResp, err := srv.DeleteVoice(ctx, adminservice.DeleteVoiceRequestObject{Id: "manual:voice-1"})
	if err != nil {
		t.Fatalf("DeleteVoice() error = %v", err)
	}
	if _, ok := deleteResp.(adminservice.DeleteVoice200JSONResponse); !ok {
		t.Fatalf("DeleteVoice() response = %#v", deleteResp)
	}
	missingResp, err := srv.GetVoice(ctx, adminservice.GetVoiceRequestObject{Id: "manual:voice-1"})
	if err != nil {
		t.Fatalf("GetVoice(missing) error = %v", err)
	}
	if _, ok := missingResp.(adminservice.GetVoice404JSONResponse); !ok {
		t.Fatalf("GetVoice(missing) response = %#v", missingResp)
	}
}

func TestProviderDataStringAndLegacyDecode(t *testing.T) {
	voice := apitypes.Voice{
		Provider: apitypes.VoiceProvider{Kind: "provider", Name: "tenant"},
		ProviderData: &apitypes.VoiceProviderData{
			"provider": map[string]string{"voice_id": " voice-1 "},
		},
	}
	if got := ProviderDataString(voice, "voice_id"); got != "voice-1" {
		t.Fatalf("ProviderDataString() = %q, want voice-1", got)
	}

	var decoded apitypes.Voice
	if err := Decode([]byte(`{
		"id": "provider:tenant:voice-1",
		"provider": {"kind": "provider", "name": "tenant"},
		"source": "sync",
		"provider_voice_id": "voice-1",
		"provider_voice_type": "system"
	}`), &decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if ProviderDataString(decoded, "voice_id") != "voice-1" || ProviderDataString(decoded, "voice_type") != "system" {
		t.Fatalf("decoded provider data = %#v", decoded.ProviderData)
	}
}
