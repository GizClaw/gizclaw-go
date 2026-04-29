package firmware

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
)

func TestStoreHelpersPathsAndAccessors(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	if got := env.srv.depotPath("depot"); got != "depot" {
		t.Fatalf("depotPath() = %q", got)
	}
	if got := env.srv.infoPath("depot"); got != "depot/info.json" {
		t.Fatalf("infoPath() = %q", got)
	}
	if got := env.srv.channelPath("depot", "stable"); got != "depot/stable" {
		t.Fatalf("channelPath() = %q", got)
	}
	if got := env.srv.manifestPath("depot", "stable"); got != "depot/stable/manifest.json" {
		t.Fatalf("manifestPath() = %q", got)
	}
	if got := env.srv.tempPath("depot", "beta"); got != "depot/.tmp-beta" {
		t.Fatalf("tempPath() = %q", got)
	}
	if env.srv.store() != env.store {
		t.Fatal("store() should return configured store")
	}
}

func TestEnsureValidateAndLockDepot(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	if err := env.srv.validateDepot(context.Background(), "missing"); !errors.Is(err, errDepotNotFound) {
		t.Fatalf("validateDepot(missing) = %v", err)
	}
	if err := env.srv.ensureDepot("bad/../name"); err == nil {
		t.Fatal("ensureDepot() expected invalid depot name error")
	}
	if err := env.srv.ensureDepot("depot"); err != nil {
		t.Fatalf("ensureDepot() unexpected error: %v", err)
	}
	env.writeDepotInfo("depot")
	if err := env.srv.validateDepot(context.Background(), "depot"); err != nil {
		t.Fatalf("validateDepot() unexpected error: %v", err)
	}

	env.writeDepotMetadata(apitypes.Depot{Name: "filedepot"})
	env.writeFile("filedepot", "x")
	if err := env.srv.validateDepot(context.Background(), "filedepot"); err == nil {
		t.Fatal("validateDepot() expected non-directory error")
	}

	unlock := env.srv.lockDepot("depot")
	if env.srv.depotMu == nil || env.srv.depotMu["depot"] == nil {
		t.Fatal("lockDepot() should initialize mutex map")
	}
	unlock()
	env.srv.lockDepot("bad/../name")()
}

func TestValidateDepotPropagatesStoreError(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	env.writeDepotInfo("depot")
	store := newMockStore(t)
	store.stat = func(name string) (fs.FileInfo, error) { return nil, errors.New("boom") }
	srv := &Server{Store: store, MetadataStore: env.meta}
	if err := srv.validateDepot(context.Background(), "depot"); err == nil || err.Error() != "boom" {
		t.Fatalf("validateDepot() error = %v", err)
	}
}
