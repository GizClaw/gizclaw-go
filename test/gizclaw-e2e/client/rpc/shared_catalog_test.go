//go:build gizclaw_e2e

package rpc_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/rpcapi"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/gizcli"
	clitest "github.com/GizClaw/gizclaw-go/test/gizclaw-e2e/cmd"
)

type sharedSetupRPCHarness struct {
	ctx  context.Context
	h    *clitest.Harness
	peer *gizcli.Client
}

func newSharedSetupRPCHarness(t *testing.T) *sharedSetupRPCHarness {
	t.Helper()

	h := clitest.NewSetupHarness(t, "client-rpc-shared-catalog")
	configHome := getenvDefault("GIZCLAW_E2E_CLIENT_CONFIG_HOME", filepath.Join(h.RepoRoot, "test", "gizclaw-e2e", "testdata", "gizclaw-config-home"))
	contextName := getenvDefault("GIZCLAW_E2E_CLIENT_CONTEXT", "e2e-client")
	h.SetContextAlias("e2e-client", configHome, contextName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	peer := h.ConnectClientFromContext("e2e-client")
	t.Cleanup(func() { peer.Close() })
	return &sharedSetupRPCHarness{ctx: ctx, h: h, peer: peer}
}

func TestSharedSetupRPCCatalogPagination(t *testing.T) {
	env := newSharedSetupRPCHarness(t)

	workflowNames := collectWorkflowNames(t, env.ctx, env.peer, 25)
	requireName(t, workflowNames, "e2e-rpc-workflow")
	requirePrefixCount(t, workflowNames, "e2e-rpc-workflow-", 120)

	workspaceNames := collectWorkspaceNames(t, env.ctx, env.peer, 25)
	requireName(t, workspaceNames, "e2e-rpc-workspace")
	requireName(t, workspaceNames, "e2e-rpc-history-workspace")
	requirePrefixCount(t, workspaceNames, "e2e-rpc-workspace-", 120)

	modelIDs := collectModelIDs(t, env.ctx, env.peer, 25)
	requireName(t, modelIDs, "e2e-rpc-model")
	requirePrefixCount(t, modelIDs, "e2e-rpc-model-", 80)

	credentialNames := collectCredentialNames(t, env.ctx, env.peer, 25)
	requireName(t, credentialNames, "e2e-rpc-credential")
	requirePrefixCount(t, credentialNames, "e2e-rpc-credential-", 50)

	firmwareNames := collectFirmwareNames(t, env.ctx, env.peer, 25)
	requireName(t, firmwareNames, "e2e-rpc-firmware")
	requirePrefixCount(t, firmwareNames, "e2e-rpc-firmware-", 80)
}

func TestSharedSetupRPCFirmwareDownloadFixture(t *testing.T) {
	env := newSharedSetupRPCHarness(t)

	got, err := env.peer.GetFirmware(env.ctx, "shared.firmware.get", rpcapi.FirmwareGetRequest{FirmwareId: "e2e-rpc-firmware"})
	if err != nil {
		t.Fatalf("firmware.get e2e-rpc-firmware: %v", err)
	}
	if got.Slots.Stable.Artifacts == nil || len(*got.Slots.Stable.Artifacts) != 1 {
		t.Fatalf("firmware stable artifacts = %#v", got.Slots.Stable.Artifacts)
	}
	artifact := (*got.Slots.Stable.Artifacts)[0]
	if artifact.Name != "main" || artifact.Path == nil || strings.TrimSpace(*artifact.Path) == "" {
		t.Fatalf("firmware artifact = %#v", artifact)
	}

	var out bytes.Buffer
	download, err := env.peer.DownloadFirmware(env.ctx, "shared.firmware.download", rpcapi.FirmwareDownloadRequest{
		FirmwareId:   "e2e-rpc-firmware",
		Channel:      rpcapi.FirmwareChannelNameStable,
		ArtifactName: "main",
	}, &out)
	if err != nil {
		t.Fatalf("firmware.download e2e-rpc-firmware: %v", err)
	}
	if download.Bytes != int64(out.Len()) {
		t.Fatalf("firmware.download bytes = %d, payload len = %d", download.Bytes, out.Len())
	}
	assertTarContains(t, out.Bytes(), "MANIFEST.txt", "gizclaw e2e rpc firmware")
}

func TestSharedSetupRPCHistoryFixture(t *testing.T) {
	env := newSharedSetupRPCHarness(t)

	limit := 2
	history, err := env.peer.ListWorkspaceHistory(env.ctx, "shared.history.list", rpcapi.WorkspaceHistoryListRequest{
		WorkspaceName: "e2e-rpc-history-workspace",
		Limit:         &limit,
	})
	if err != nil {
		t.Fatalf("workspace history list: %v", err)
	}
	if len(history.Items) != 2 || !history.HasNext || history.NextCursor == nil {
		t.Fatalf("workspace history first page = %#v", history)
	}
	first := history.Items[0]
	if first.Type != rpcapi.PeerRunHistoryEntryTypeGear || first.GearId == nil || !first.ReplayAvailable {
		t.Fatalf("workspace history first entry = %#v", first)
	}
	if !strings.HasPrefix(first.Text, "rpc shared history round") {
		t.Fatalf("workspace history first text = %q", first.Text)
	}

	next, err := env.peer.ListWorkspaceHistory(env.ctx, "shared.history.next", rpcapi.WorkspaceHistoryListRequest{
		WorkspaceName: "e2e-rpc-history-workspace",
		Limit:         &limit,
		Cursor:        history.NextCursor,
	})
	if err != nil {
		t.Fatalf("workspace history next page: %v", err)
	}
	if len(next.Items) == 0 {
		t.Fatalf("workspace history next page = %#v", next)
	}

	got, err := env.peer.GetWorkspaceHistory(env.ctx, "shared.history.get", rpcapi.WorkspaceHistoryGetRequest{
		WorkspaceName: "e2e-rpc-history-workspace",
		HistoryId:     first.Id,
	})
	if err != nil {
		t.Fatalf("workspace history get %q: %v", first.Id, err)
	}
	if got.Id != first.Id || got.Text != first.Text {
		t.Fatalf("workspace history get = %#v, want id/text from %#v", got, first)
	}

	var audio bytes.Buffer
	audioResp, err := env.peer.GetWorkspaceHistoryAudio(env.ctx, "shared.history.audio", rpcapi.WorkspaceHistoryAudioGetRequest{
		WorkspaceName: "e2e-rpc-history-workspace",
		HistoryId:     first.Id,
	}, &audio)
	if err != nil {
		t.Fatalf("workspace history audio get %q: %v", first.Id, err)
	}
	if audioResp.Bytes != int64(audio.Len()) || !strings.Contains(audio.String(), "rpc-history-opus-payload") {
		t.Fatalf("workspace history audio bytes = %d payload=%q metadata=%#v", audio.Len(), audio.String(), audioResp)
	}
}

func TestSharedSetupRPCMutationFixtures(t *testing.T) {
	env := newSharedSetupRPCHarness(t)

	_, _ = env.peer.DeleteWorkflow(env.ctx, "shared.workflow.delete.preclean", rpcapi.WorkflowDeleteRequest{Name: "e2e-rpc-mut-workflow"})
	createdWorkflow, err := env.peer.CreateWorkflow(env.ctx, "shared.workflow.create", rpcWorkflow("e2e-rpc-mut-workflow", "shared setup mutation workflow"))
	if err != nil {
		t.Fatalf("workflow.create e2e-rpc-mut-workflow: %v", err)
	}
	if createdWorkflow.Metadata.Name != "e2e-rpc-mut-workflow" {
		t.Fatalf("workflow.create = %#v", createdWorkflow)
	}
	if _, err := env.peer.DeleteWorkflow(env.ctx, "shared.workflow.delete", rpcapi.WorkflowDeleteRequest{Name: "e2e-rpc-mut-workflow"}); err != nil {
		t.Fatalf("workflow.delete e2e-rpc-mut-workflow: %v", err)
	}

	_, _ = env.peer.DeleteModel(env.ctx, "shared.model.delete.preclean", rpcapi.ModelDeleteRequest{Id: "e2e-rpc-mut-model"})
	createdModel, err := env.peer.CreateModel(env.ctx, "shared.model.create", rpcModel("e2e-rpc-mut-model", "e2e-rpc-provider"))
	if err != nil {
		t.Fatalf("model.create e2e-rpc-mut-model: %v", err)
	}
	if createdModel.Id != "e2e-rpc-mut-model" {
		t.Fatalf("model.create = %#v", createdModel)
	}
	if _, err := env.peer.DeleteModel(env.ctx, "shared.model.delete", rpcapi.ModelDeleteRequest{Id: "e2e-rpc-mut-model"}); err != nil {
		t.Fatalf("model.delete e2e-rpc-mut-model: %v", err)
	}

	_, _ = env.peer.DeleteCredential(env.ctx, "shared.credential.delete.preclean", rpcapi.CredentialDeleteRequest{Name: "e2e-rpc-mut-credential"})
	createdCredential, err := env.peer.CreateCredential(env.ctx, "shared.credential.create", rpcCredential("e2e-rpc-mut-credential", "sk-e2e-rpc-mut"))
	if err != nil {
		t.Fatalf("credential.create e2e-rpc-mut-credential: %v", err)
	}
	if createdCredential.Name != "e2e-rpc-mut-credential" {
		t.Fatalf("credential.create = %#v", createdCredential)
	}
	if _, err := env.peer.DeleteCredential(env.ctx, "shared.credential.delete", rpcapi.CredentialDeleteRequest{Name: "e2e-rpc-mut-credential"}); err != nil {
		t.Fatalf("credential.delete e2e-rpc-mut-credential: %v", err)
	}
}

func collectWorkflowNames(t *testing.T, ctx context.Context, peer *gizcli.Client, limit int) map[string]bool {
	t.Helper()

	names := map[string]bool{}
	var cursor *string
	for page := 0; page < 100; page++ {
		list, err := peer.ListWorkflows(ctx, "shared.workflow.list", rpcapi.WorkflowListRequest{Cursor: cursor, Limit: &limit})
		if err != nil {
			t.Fatalf("workflow.list page %d: %v", page, err)
		}
		for _, item := range list.Items {
			names[item.Metadata.Name] = true
		}
		if !list.HasNext {
			return names
		}
		if list.NextCursor == nil || *list.NextCursor == "" {
			t.Fatalf("workflow.list page %d has_next without next cursor: %#v", page, list)
		}
		cursor = list.NextCursor
	}
	t.Fatal("workflow.list pagination did not terminate")
	return names
}

func collectWorkspaceNames(t *testing.T, ctx context.Context, peer *gizcli.Client, limit int) map[string]bool {
	t.Helper()

	names := map[string]bool{}
	var cursor *string
	for page := 0; page < 100; page++ {
		list, err := peer.ListWorkspaces(ctx, "shared.workspace.list", rpcapi.WorkspaceListRequest{Cursor: cursor, Limit: &limit})
		if err != nil {
			t.Fatalf("workspace.list page %d: %v", page, err)
		}
		for _, item := range list.Items {
			names[item.Name] = true
		}
		if !list.HasNext {
			return names
		}
		if list.NextCursor == nil || *list.NextCursor == "" {
			t.Fatalf("workspace.list page %d has_next without next cursor: %#v", page, list)
		}
		cursor = list.NextCursor
	}
	t.Fatal("workspace.list pagination did not terminate")
	return names
}

func collectModelIDs(t *testing.T, ctx context.Context, peer *gizcli.Client, limit int) map[string]bool {
	t.Helper()

	names := map[string]bool{}
	var cursor *string
	for page := 0; page < 100; page++ {
		list, err := peer.ListModels(ctx, "shared.model.list", rpcapi.ModelListRequest{Cursor: cursor, Limit: &limit})
		if err != nil {
			t.Fatalf("model.list page %d: %v", page, err)
		}
		for _, item := range list.Items {
			names[item.Id] = true
		}
		if !list.HasNext {
			return names
		}
		if list.NextCursor == nil || *list.NextCursor == "" {
			t.Fatalf("model.list page %d has_next without next cursor: %#v", page, list)
		}
		cursor = list.NextCursor
	}
	t.Fatal("model.list pagination did not terminate")
	return names
}

func collectCredentialNames(t *testing.T, ctx context.Context, peer *gizcli.Client, limit int) map[string]bool {
	t.Helper()

	names := map[string]bool{}
	var cursor *string
	for page := 0; page < 100; page++ {
		list, err := peer.ListCredentials(ctx, "shared.credential.list", rpcapi.CredentialListRequest{Cursor: cursor, Limit: &limit})
		if err != nil {
			t.Fatalf("credential.list page %d: %v", page, err)
		}
		for _, item := range list.Items {
			names[item.Name] = true
		}
		if !list.HasNext {
			return names
		}
		if list.NextCursor == nil || *list.NextCursor == "" {
			t.Fatalf("credential.list page %d has_next without next cursor: %#v", page, list)
		}
		cursor = list.NextCursor
	}
	t.Fatal("credential.list pagination did not terminate")
	return names
}

func collectFirmwareNames(t *testing.T, ctx context.Context, peer *gizcli.Client, limit int) map[string]bool {
	t.Helper()

	names := map[string]bool{}
	var cursor *string
	for page := 0; page < 100; page++ {
		list, err := peer.ListFirmwares(ctx, "shared.firmware.list", rpcapi.FirmwareListRequest{Cursor: cursor, Limit: &limit})
		if err != nil {
			t.Fatalf("firmware.list page %d: %v", page, err)
		}
		for _, item := range list.Items {
			names[item.Name] = true
		}
		if !list.HasNext {
			return names
		}
		if list.NextCursor == nil || *list.NextCursor == "" {
			t.Fatalf("firmware.list page %d has_next without next cursor: %#v", page, list)
		}
		cursor = list.NextCursor
	}
	t.Fatal("firmware.list pagination did not terminate")
	return names
}

func requireName(t *testing.T, names map[string]bool, name string) {
	t.Helper()
	if !names[name] {
		t.Fatalf("missing %q in names map with %d entries", name, len(names))
	}
}

func requirePrefixCount(t *testing.T, names map[string]bool, prefix string, want int) {
	t.Helper()
	got := 0
	for name := range names {
		if strings.HasPrefix(name, prefix) {
			got++
		}
	}
	if got < want {
		t.Fatalf("prefix %q count = %d, want at least %d", prefix, got, want)
	}
}

func assertTarContains(t *testing.T, data []byte, name string, want string) {
	t.Helper()

	tr := tar.NewReader(bytes.NewReader(data))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar: %v", err)
		}
		if header.Name != name {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read tar member %q: %v", name, err)
		}
		if !strings.Contains(string(body), want) {
			t.Fatalf("tar member %q missing %q: %s", name, want, string(body))
		}
		return
	}
	t.Fatalf("tar member %q not found", name)
}

func getenvDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
