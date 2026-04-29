package firmware

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

func TestScanDepotNames(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	env.writeDepotInfo("depot-a", "fw.bin")
	env.writeRelease("depot-b", Beta, "1.0.0", map[string]string{"fw.bin": "firmware"})
	env.writeFile(".hidden/info.json", `{}`)
	env.writeFile("ignored/dev/manifest.json", `{}`)

	names, err := env.srv.scanDepotNames(context.Background())
	if err != nil {
		t.Fatalf("scanDepotNames() unexpected error: %v", err)
	}
	if len(names) != 2 || names[0] != "depot-a" || names[1] != "depot-b" {
		t.Fatalf("scanDepotNames() = %#v", names)
	}
}

func TestScanDepotAndResolveOTA(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	env.writeDepotInfo("depot", "fw.bin")
	env.writeRelease("depot", Stable, "1.2.3", map[string]string{"fw.bin": "firmware"})

	depot, err := env.srv.scanDepot(context.Background(), "depot")
	if err != nil {
		t.Fatalf("scanDepot() unexpected error: %v", err)
	}
	if depot.Name != "depot" || depot.Stable.FirmwareSemver != "1.2.3" {
		t.Fatalf("scanDepot() = %+v", depot)
	}

	ota, err := env.srv.resolveOTA(context.Background(), "depot", Stable)
	if err != nil {
		t.Fatalf("resolveOTA() unexpected error: %v", err)
	}
	if ota.Depot != "depot" || ota.Channel != string(Stable) || ota.FirmwareSemver != "1.2.3" {
		t.Fatalf("resolveOTA() = %+v", ota)
	}
	if _, err := env.srv.resolveOTA(context.Background(), "depot", Beta); !errors.Is(err, errFirmwareNotFound) {
		t.Fatalf("resolveOTA() missing channel error = %v", err)
	}
	if _, err := env.srv.resolveOTA(context.Background(), "missing", Stable); !errors.Is(err, errDepotNotFound) {
		t.Fatalf("resolveOTA() missing depot error = %v", err)
	}
}

func TestScanReleaseErrors(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeDepotInfo("depot")
		if _, err := env.srv.scanRelease(context.Background(), "depot", Stable); !errors.Is(err, errChannelNotFound) {
			t.Fatalf("scanRelease() error = %v", err)
		}
	})

	t.Run("invalid metadata", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		if err := env.meta.Set(context.Background(), depotMetadataKey("depot"), []byte("{")); err != nil {
			t.Fatalf("seed bad metadata: %v", err)
		}
		if _, err := env.srv.scanRelease(context.Background(), "depot", Stable); err == nil {
			t.Fatal("scanRelease() expected metadata error")
		}
	})

	t.Run("channel mismatch", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeFile("depot/stable/fw.bin", "firmware")
		depot := depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "firmware"})
		if err := env.meta.Set(context.Background(), depotMetadataKey("depot"), mustJSON(t, struct {
			Name   string `json:"name"`
			Stable any    `json:"stable"`
		}{Name: "depot", Stable: depot})); err != nil {
			t.Fatalf("seed mismatched metadata: %v", err)
		}
		if _, err := env.srv.scanRelease(context.Background(), "depot", Stable); err == nil {
			t.Fatal("scanRelease() expected channel mismatch")
		}
	})

	t.Run("hash mismatch", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeRelease("depot", Stable, "1.0.0", map[string]string{"fw.bin": "firmware"})
		env.writeFile("depot/stable/fw.bin", "modified")
		if _, err := env.srv.scanRelease(context.Background(), "depot", Stable); err == nil {
			t.Fatal("scanRelease() expected hash mismatch")
		}
	})

	t.Run("read error", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeRelease("depot", Stable, "1.0.0", map[string]string{"fw.bin": "firmware"})
		store := newMockStore(t)
		store.base = env.store
		store.readFile = func(name string) ([]byte, error) { return nil, errors.New("boom") }
		srv := &Server{Store: store, MetadataStore: env.meta}
		if _, err := srv.scanRelease(context.Background(), "depot", Stable); err == nil {
			t.Fatal("scanRelease() expected read error")
		}
	})
}

func TestScanDepotVersionOrderViolation(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	env.writeRelease("depot", Stable, "2.0.0", map[string]string{"fw.bin": "stable"})
	beta := depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "beta"})
	env.writeFile("depot/beta/fw.bin", "beta")
	if err := env.meta.Set(context.Background(), depotMetadataKey("depot"), mustJSON(t, struct {
		Name   string `json:"name"`
		Stable any    `json:"stable"`
		Beta   any    `json:"beta"`
	}{Name: "depot", Stable: depotReleaseForFiles(Stable, "2.0.0", map[string]string{"fw.bin": "stable"}), Beta: beta})); err != nil {
		t.Fatalf("seed version violation metadata: %v", err)
	}

	if _, err := env.srv.scanDepot(context.Background(), "depot"); !errors.Is(err, errVersionOrderViolation) {
		t.Fatalf("scanDepot() error = %v", err)
	}
}

func TestScanDepotNamesMetadataError(t *testing.T) {
	t.Parallel()

	meta := newMockKVStore()
	meta.list = func(context.Context, kv.Key) iter.Seq2[kv.Entry, error] {
		return func(yield func(kv.Entry, error) bool) {
			yield(kv.Entry{}, errors.New("boom"))
		}
	}
	srv := &Server{Store: newMockStore(t), MetadataStore: meta}
	if _, err := srv.scanDepotNames(context.Background()); err == nil || err.Error() != "boom" {
		t.Fatalf("scanDepotNames() error = %v", err)
	}
}

func TestScanDepotNamesEmptyMetadata(t *testing.T) {
	t.Parallel()

	srv := &Server{Store: newMockStore(t), MetadataStore: kv.NewMemory(nil)}
	names, err := srv.scanDepotNames(context.Background())
	if err != nil {
		t.Fatalf("scanDepotNames() unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("scanDepotNames() names = %#v", names)
	}
}
