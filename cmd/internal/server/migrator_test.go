package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GizClaw/gizclaw-go/cmd/internal/storage"
	"github.com/GizClaw/gizclaw-go/cmd/internal/stores"
)

func TestNewMigratorRunsACLMigration(t *testing.T) {
	migrator, err := NewMigrator(validLayeredConfig(t.TempDir()))
	if err != nil {
		t.Fatalf("NewMigrator() error = %v", err)
	}
	defer migrator.Close()

	for range 2 {
		if err := migrator.Migrate(context.Background()); err != nil {
			t.Fatalf("Migrate() error = %v", err)
		}
	}
	if _, err := migrator.ACL.DB.ExecContext(context.Background(), `INSERT INTO acl_views (name, created_at, updated_at) VALUES ('default', 'now', 'now')`); err != nil {
		t.Fatalf("insert acl view after migration: %v", err)
	}
}

func TestCmdMigratorCloseHandlesNilState(t *testing.T) {
	var nilMigrator *CmdMigrator
	if err := nilMigrator.Close(); err != nil {
		t.Fatalf("nil Close() error = %v", err)
	}
	if err := (&CmdMigrator{}).Close(); err != nil {
		t.Fatalf("empty Close() error = %v", err)
	}
}

func TestMigrateWorkspaceRunsACLMigrationFromWorkspaceConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, workspaceConfigFile), []byte(`
listen: "127.0.0.1:0"
storage:
  acl-db:
    kind: sql
    sqlite:
      dir: data/acl.sqlite
stores:
  acl:
    kind: sql
    storage: acl-db
acl:
  store: acl
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := MigrateWorkspace(context.Background(), root); err != nil {
		t.Fatalf("MigrateWorkspace() error = %v", err)
	}

	migrator, err := NewMigrator(Config{
		Storage: map[string]storage.Config{
			"acl-db": {Kind: storage.KindSQL, SQLite: &storage.SQLConfig{Dir: filepath.Join(root, "data", "acl.sqlite")}},
		},
		Stores: map[string]stores.Config{
			"acl": {Kind: stores.KindSQL, Storage: "acl-db"},
		},
		ACL: ACLConfig{Store: "acl"},
	})
	if err != nil {
		t.Fatalf("NewMigrator() after workspace migration error = %v", err)
	}
	defer migrator.Close()
	if _, err := migrator.ACL.DB.ExecContext(context.Background(), `INSERT INTO acl_views (name, created_at, updated_at) VALUES ('workspace', 'now', 'now')`); err != nil {
		t.Fatalf("insert acl view after workspace migration: %v", err)
	}
}

func TestNewMigratorRequiresACLStore(t *testing.T) {
	if _, err := NewMigrator(Config{}); err == nil {
		t.Fatal("NewMigrator() error = nil")
	}
}
