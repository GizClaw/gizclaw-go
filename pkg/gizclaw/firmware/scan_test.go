package firmware

import (
	"errors"
	"io/fs"
	"testing"
)

func TestScanDepotNames(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	env.writeDepotInfo("depot-a", "fw.bin")
	env.writeFile("depot-b/beta/manifest.json", `{"channel":"beta","firmware_semver":"1.0.0"}`)
	env.writeFile(".hidden/info.json", `{}`)
	env.writeFile("ignored/dev/manifest.json", `{}`)

	names, err := env.srv.scanDepotNames()
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

	depot, err := env.srv.scanDepot("depot")
	if err != nil {
		t.Fatalf("scanDepot() unexpected error: %v", err)
	}
	if depot.Name != "depot" || depot.Stable.FirmwareSemver != "1.2.3" {
		t.Fatalf("scanDepot() = %+v", depot)
	}

	ota, err := env.srv.resolveOTA("depot", Stable)
	if err != nil {
		t.Fatalf("resolveOTA() unexpected error: %v", err)
	}
	if ota.Depot != "depot" || ota.Channel != string(Stable) || ota.FirmwareSemver != "1.2.3" {
		t.Fatalf("resolveOTA() = %+v", ota)
	}
	if _, err := env.srv.resolveOTA("depot", Beta); !errors.Is(err, errFirmwareNotFound) {
		t.Fatalf("resolveOTA() missing channel error = %v", err)
	}
	if _, err := env.srv.resolveOTA("missing", Stable); !errors.Is(err, errDepotNotFound) {
		t.Fatalf("resolveOTA() missing depot error = %v", err)
	}
}

func TestScanReleaseErrors(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		if _, err := env.srv.scanRelease("depot", Stable); !errors.Is(err, errChannelNotFound) {
			t.Fatalf("scanRelease() error = %v", err)
		}
	})

	t.Run("invalid manifest", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeFile("depot/stable/manifest.json", `{`)
		if _, err := env.srv.scanRelease("depot", Stable); err == nil {
			t.Fatal("scanRelease() expected parse error")
		}
	})

	t.Run("channel mismatch", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeRelease("depot", Stable, "1.0.0", map[string]string{"fw.bin": "firmware"})
		env.writeJSON("depot/stable/manifest.json", depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "firmware"}))
		if _, err := env.srv.scanRelease("depot", Stable); err == nil {
			t.Fatal("scanRelease() expected channel mismatch")
		}
	})

	t.Run("hash mismatch", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeRelease("depot", Stable, "1.0.0", map[string]string{"fw.bin": "firmware"})
		env.writeFile("depot/stable/fw.bin", "modified")
		if _, err := env.srv.scanRelease("depot", Stable); err == nil {
			t.Fatal("scanRelease() expected hash mismatch")
		}
	})

	t.Run("read error", func(t *testing.T) {
		t.Parallel()
		store := newMockStore(t)
		store.readFile = func(name string) ([]byte, error) { return nil, errors.New("boom") }
		srv := &Server{Store: store}
		if _, err := srv.scanRelease("depot", Stable); err == nil {
			t.Fatal("scanRelease() expected read error")
		}
	})
}

func TestScanDepotVersionOrderViolation(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t)
	env.writeRelease("depot", Stable, "2.0.0", map[string]string{"fw.bin": "stable"})
	env.writeRelease("depot", Beta, "1.0.0", map[string]string{"fw.bin": "beta"})

	if _, err := env.srv.scanDepot("depot"); !errors.Is(err, errVersionOrderViolation) {
		t.Fatalf("scanDepot() error = %v", err)
	}
}

func TestScanDepotNamesWalkError(t *testing.T) {
	t.Parallel()

	store := newMockStore(t)
	store.walkDir = func(root string, fn fs.WalkDirFunc) error { return errors.New("boom") }
	srv := &Server{Store: store}
	if _, err := srv.scanDepotNames(); err == nil || err.Error() != "boom" {
		t.Fatalf("scanDepotNames() error = %v", err)
	}
}

func TestScanDepotNamesCallbackEdgeCases(t *testing.T) {
	t.Parallel()

	store := newMockStore(t)
	store.walkDir = func(root string, fn fs.WalkDirFunc) error {
		if err := fn(".", nil, nil); err != nil {
			return err
		}
		if err := fn("missing", nil, fs.ErrNotExist); err != nil {
			return err
		}
		return nil
	}
	srv := &Server{Store: store}
	names, err := srv.scanDepotNames()
	if err != nil {
		t.Fatalf("scanDepotNames() unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("scanDepotNames() names = %#v", names)
	}
}
