//go:build gizclaw_e2e

package admin_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
)

func TestAdminAPIWorkspaceHistoryListGetAndDownloadAudio(t *testing.T) {
	env := newAdminAPIHarness(t)
	workspaceName := "e2e-rpc-history-workspace"
	order := adminservice.Asc
	limit := 2

	history, err := env.api.ListWorkspaceHistoryWithResponse(env.ctx, workspaceName, &adminservice.ListWorkspaceHistoryParams{
		Limit: &limit,
		Order: &order,
	})
	if err != nil {
		t.Fatalf("list workspace history: %v", err)
	}
	requireStatusOK(t, history, history.Body)
	if history.JSON200 == nil || len(history.JSON200.Items) != 2 || !history.JSON200.HasNext || history.JSON200.NextCursor == nil {
		t.Fatalf("workspace history list = %#v", history.JSON200)
	}
	first := history.JSON200.Items[0]
	if first.Id == "" || first.Text == "" || !first.ReplayAvailable || first.GearId == nil {
		t.Fatalf("first workspace history = %#v", first)
	}
	if !strings.HasPrefix(first.Text, "rpc shared history round") {
		t.Fatalf("first workspace history text = %q", first.Text)
	}

	next, err := env.api.ListWorkspaceHistoryWithResponse(env.ctx, workspaceName, &adminservice.ListWorkspaceHistoryParams{
		Limit:  &limit,
		Cursor: history.JSON200.NextCursor,
		Order:  &order,
	})
	if err != nil {
		t.Fatalf("list workspace history next page: %v", err)
	}
	requireStatusOK(t, next, next.Body)
	if next.JSON200 == nil || len(next.JSON200.Items) == 0 {
		t.Fatalf("workspace history next page = %#v", next.JSON200)
	}

	get, err := env.api.GetWorkspaceHistoryWithResponse(env.ctx, workspaceName, first.Id)
	if err != nil {
		t.Fatalf("get workspace history: %v", err)
	}
	requireStatusOK(t, get, get.Body)
	if get.JSON200 == nil || get.JSON200.Id != first.Id || get.JSON200.Text != first.Text {
		t.Fatalf("workspace history get = %#v, want %#v", get.JSON200, first)
	}

	audio, err := env.api.DownloadWorkspaceHistoryAudioWithResponse(env.ctx, workspaceName, first.Id)
	if err != nil {
		t.Fatalf("download workspace history audio: %v", err)
	}
	requireStatusOK(t, audio, audio.Body)
	if got := audio.HTTPResponse.Header.Get("Content-Type"); got != "audio/ogg" {
		t.Fatalf("history audio content-type = %q, want audio/ogg", got)
	}
	if !bytes.Contains(audio.Body, []byte("rpc-history-opus-payload")) {
		t.Fatalf("history audio payload = %q", string(audio.Body))
	}
}
