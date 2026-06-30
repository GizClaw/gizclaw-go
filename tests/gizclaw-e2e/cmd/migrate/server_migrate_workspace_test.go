//go:build gizclaw_e2e

package migrate_test

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	clitest "github.com/GizClaw/gizclaw-go/tests/gizclaw-e2e/cmd"
	_ "modernc.org/sqlite"
)

func TestServerMigrateWorkspaceUserStory(t *testing.T) {
	h := clitest.NewHarness(t, "204-server-migrate-workspace")
	h.ServerAddr = "127.0.0.1:0"
	h.PrepareServerWorkspaceFromFixture("server_config.yaml")

	for range 2 {
		result := h.RunCLI("migrate", "--workspace", h.ServerWorkspace)
		result.MustSucceed(t)
		if !strings.Contains(result.Stdout, "Migrated workspace ") {
			t.Fatalf("migrate output = %q", result.Stdout)
		}
	}

	db, err := sql.Open("sqlite", filepath.Join(h.ServerWorkspace, "data", "acl.sqlite"))
	if err != nil {
		t.Fatalf("open migrated sqlite db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO acl_views (name, created_at, updated_at) VALUES ('default', 'now', 'now')`); err != nil {
		t.Fatalf("insert acl view after migrate: %v", err)
	}
}
