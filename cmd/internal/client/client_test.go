package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GizClaw/gizclaw-go/cmd/internal/clicontext"
	"github.com/GizClaw/gizclaw-go/pkg/giznet"
)

func TestDialFromContextNoActiveContext(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, _, _, err := DialFromContext("")
	if err == nil {
		t.Fatal("DialFromContext should fail without an active context")
	}
	if !strings.Contains(err.Error(), "no active context") {
		t.Fatalf("DialFromContext error = %v", err)
	}
}

func TestDialFromContextInvalidServerPublicKey(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	store, err := clicontext.DefaultStore()
	if err != nil {
		t.Fatalf("DefaultStore error = %v", err)
	}
	if err := store.Create("local", "127.0.0.1:9820", strings.Repeat("ab", giznet.KeySize)); err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.Root, "local", "config.yaml"), []byte(`
server:
  address: 127.0.0.1:9820
  public-key: not-hex
`), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, _, _, err = DialFromContext("local")
	if err == nil {
		t.Fatal("DialFromContext should fail on invalid server public key")
	}
	if !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("DialFromContext error = %v", err)
	}
}

func TestDialFromContextMissingNamedContext(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, _, _, err := DialFromContext("missing")
	if err == nil {
		t.Fatal("DialFromContext should fail for a missing named context")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("DialFromContext error = %v", err)
	}
}
