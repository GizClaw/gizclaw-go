package adminvoices_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

func TestAdminVoicesUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "506-admin-voices")
	h.StartServerFromFixture("server_config.yaml")
	h.CreateContext("admin-a").MustSucceed(t)
	h.RegisterContext("admin-a", "admin_default", "--sn", "admin-sn").MustSucceed(t)

	seedVoices(t, h)

	list := h.RunCLI("admin", "voices", "list", "--context", "admin-a")
	list.MustSucceed(t)
	for _, want := range []string{`"id":"voice-1"`, `"id":"voice-2"`} {
		if !strings.Contains(list.Stdout, want) {
			t.Fatalf("voices list missing %q:\n%s", want, list.Stdout)
		}
	}

	filtered := h.RunCLI("admin", "voices", "list", "--provider-name", "main-cn", "--context", "admin-a")
	filtered.MustSucceed(t)
	if !strings.Contains(filtered.Stdout, `"id":"voice-1"`) || strings.Contains(filtered.Stdout, `"id":"voice-2"`) {
		t.Fatalf("voices filtered list returned unexpected items:\n%s", filtered.Stdout)
	}

	get := h.RunCLI("admin", "voices", "get", "voice-1", "--context", "admin-a")
	get.MustSucceed(t)
	if !strings.Contains(get.Stdout, `"name":"Voice One"`) {
		t.Fatalf("voices get missing name:\n%s", get.Stdout)
	}
}

func seedVoices(t *testing.T, h *clitest.Harness) {
	t.Helper()

	c := h.ConnectClientFromContext("admin-a")
	defer c.Close()
	api, err := c.ServerAdminClient()
	if err != nil {
		t.Fatalf("create admin client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, req := range []adminservice.VoiceUpsert{
		{
			Id:     "voice-1",
			Source: apitypes.VoiceSourceManual,
			Provider: apitypes.VoiceProvider{
				Kind: "minimax-tenant",
				Name: "main-cn",
			},
			Name:        ptr("Voice One"),
			Description: ptr("primary"),
		},
		{
			Id:     "voice-2",
			Source: apitypes.VoiceSourceManual,
			Provider: apitypes.VoiceProvider{
				Kind: "minimax-tenant",
				Name: "other-cn",
			},
			Name: ptr("Voice Two"),
		},
	} {
		resp, err := api.CreateVoiceWithResponse(ctx, req)
		if err != nil {
			t.Fatalf("seed voice %q: %v", req.Id, err)
		}
		if resp.JSON200 == nil {
			t.Fatalf("seed voice %q got status %d: %s", req.Id, resp.StatusCode(), strings.TrimSpace(string(resp.Body)))
		}
	}
}

func ptr(value string) *string {
	return &value
}
