//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/services/ai/workspace"
	"github.com/GizClaw/gizclaw-go/pkg/store/objectstore"
)

const historyWorkspaceName = "e2e-rpc-history-workspace"

func main() {
	repoRoot, err := findRepoRoot()
	must(err)
	resourcesDir := filepath.Join(repoRoot, "test", "gizclaw-e2e", "testdata", "resources")
	peerID := mustDefaultClientPeer(resourcesDir)

	objects := objectstore.Dir(filepath.Join(repoRoot, "test", "gizclaw-e2e", "testdata", "server-workspace", "data", "agenthost"))
	must(objects.DeletePrefix(workspace.ObjectPrefix(historyWorkspaceName) + "/history"))
	store := workspace.NewHistoryStore(objects, historyWorkspaceName)
	base := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	for i, text := range []string{
		"rpc shared history round one",
		"rpc shared history round two",
		"rpc shared history round three",
	} {
		_, err := store.Append(context.Background(), workspace.AppendHistoryRequest{
			Type:      "gear",
			GearID:    peerID,
			Name:      "transcript",
			Text:      text,
			CreatedAt: base.Add(time.Duration(i) * time.Second),
			Asset: &workspace.AppendHistoryAsset{
				MIMEType: "audio/opus",
				Data:     []byte(fmt.Sprintf("rpc-history-opus-payload-%d\n", i+1)),
			},
		})
		must(err)
	}
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", fmt.Errorf("repo root with go.mod not found")
		}
		wd = parent
	}
}

func mustDefaultClientPeer(resourcesDir string) string {
	data, err := os.ReadFile(filepath.Join(resourcesDir, "031-client-peer-config.json"))
	must(err)
	var doc struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
	}
	must(json.Unmarshal(data, &doc))
	if doc.Metadata.Name == "" {
		panic("031-client-peer-config.json metadata.name is empty")
	}
	return doc.Metadata.Name
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
