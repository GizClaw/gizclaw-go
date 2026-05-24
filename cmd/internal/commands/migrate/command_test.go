package migratecmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMigrateCommandRunsWorkspaceMigration(t *testing.T) {
	previous := migrateWorkspace
	defer func() { migrateWorkspace = previous }()

	var gotWorkspace string
	migrateWorkspace = func(_ context.Context, workspace string) error {
		gotWorkspace = workspace
		return nil
	}

	cmd := NewCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--workspace", "  /tmp/gizclaw  "})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if gotWorkspace != "/tmp/gizclaw" {
		t.Fatalf("workspace = %q, want /tmp/gizclaw", gotWorkspace)
	}
	if !strings.Contains(out.String(), "Migrated workspace /tmp/gizclaw") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestMigrateCommandRequiresWorkspace(t *testing.T) {
	cmd := NewCmd()
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--workspace is required") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestMigrateCommandReturnsMigrationError(t *testing.T) {
	previous := migrateWorkspace
	defer func() { migrateWorkspace = previous }()

	wantErr := errors.New("failed")
	migrateWorkspace = func(context.Context, string) error {
		return wantErr
	}

	cmd := NewCmd()
	cmd.SetArgs([]string{"--workspace", "/tmp/gizclaw"})
	if err := cmd.Execute(); !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
