package firmware

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

func TestExtractTar(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "firmware"}))},
			tarEntry{Name: "fw.bin", Data: []byte("firmware")},
		)
		release, err := extractTar(env.store, "depot/.tmp-beta", Beta, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("extractTar() unexpected error: %v", err)
		}
		if release.FirmwareSemver != "1.0.0" {
			t.Fatalf("extractTar() release = %+v", release)
		}
	})

	t.Run("missing manifest", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t, tarEntry{Name: "fw.bin", Data: []byte("firmware")})
		if _, err := extractTar(env.store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected missing manifest")
		}
	})

	t.Run("illegal entry", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t, tarEntry{Name: "dir", Typeflag: tar.TypeDir})
		if _, err := extractTar(env.store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected illegal entry error")
		}
	})

	t.Run("manifest channel mismatch", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Stable, "1.0.0", map[string]string{"fw.bin": "firmware"}))},
			tarEntry{Name: "fw.bin", Data: []byte("firmware")},
		)
		if _, err := extractTar(env.store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected channel mismatch")
		}
	})

	t.Run("invalid manifest json", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t, tarEntry{Name: "manifest.json", Data: []byte("{")})
		if _, err := extractTar(env.store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected invalid manifest error")
		}
	})

	t.Run("invalid path entry", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t, tarEntry{Name: "../bad.bin", Data: []byte("x")})
		if _, err := extractTar(env.store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected invalid path error")
		}
	})

	t.Run("duplicate entry", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "firmware"}))},
			tarEntry{Name: "fw.bin", Data: []byte("one")},
			tarEntry{Name: "fw.bin", Data: []byte("two")},
		)
		if _, err := extractTar(env.store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected duplicate entry error")
		}
	})

	t.Run("missing manifest file", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "firmware"}))},
		)
		if _, err := extractTar(env.store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected missing manifest file")
		}
	})

	t.Run("extra tar files", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "firmware"}))},
			tarEntry{Name: "fw.bin", Data: []byte("firmware")},
			tarEntry{Name: "extra.bin", Data: []byte("extra")},
		)
		if _, err := extractTar(env.store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected tar mismatch")
		}
	})

	t.Run("write error", func(t *testing.T) {
		t.Parallel()
		store := newMockStore(t)
		store.writeFile = func(name string, data []byte) error { return errors.New("boom") }
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "firmware"}))},
			tarEntry{Name: "fw.bin", Data: []byte("firmware")},
		)
		if _, err := extractTar(store, "tmp", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("extractTar() expected write error")
		}
	})
}

func TestUploadTar(t *testing.T) {
	t.Parallel()

	t.Run("invalid channel", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		if _, err := env.srv.uploadTar(context.Background(), "depot", Channel("dev"), bytes.NewReader(nil)); err == nil {
			t.Fatal("uploadTar() expected invalid channel error")
		}
	})

	t.Run("success replace existing", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeDepotInfo("depot", "fw.bin")
		env.writeRelease("depot", Beta, "1.0.0", map[string]string{"fw.bin": "old"})

		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.1.0", map[string]string{"fw.bin": "new"}))},
			tarEntry{Name: "fw.bin", Data: []byte("new")},
		)
		release, err := env.srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("uploadTar() unexpected error: %v", err)
		}
		if release.FirmwareSemver != "1.1.0" {
			t.Fatalf("uploadTar() release = %+v", release)
		}
		if got := string(env.readFile("depot/beta/fw.bin")); got != "new" {
			t.Fatalf("uploaded file = %q", got)
		}
	})

	t.Run("info mismatch", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeDepotInfo("depot", "other.bin")
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "new"}))},
			tarEntry{Name: "fw.bin", Data: []byte("new")},
		)
		if _, err := env.srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("uploadTar() expected info mismatch")
		}
	})

	t.Run("mkdir temp error", func(t *testing.T) {
		t.Parallel()
		store := newMockStore(t)
		store.mkdirAll = func(name string) error {
			if name == "depot/.tmp-beta" || name == "depot" {
				return errors.New("boom")
			}
			return store.base.MkdirAll(name)
		}
		srv := &Server{Store: store}
		data := buildTar(t)
		if _, err := srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("uploadTar() expected mkdir error")
		}
	})

	t.Run("metadata read error", func(t *testing.T) {
		t.Parallel()
		store := newMockStore(t)
		meta := newMockKVStore()
		meta.get = func(context.Context, kv.Key) ([]byte, error) { return nil, errors.New("boom") }
		srv := &Server{Store: store, MetadataStore: meta}
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "new"}))},
			tarEntry{Name: "fw.bin", Data: []byte("new")},
		)
		if _, err := srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("uploadTar() expected metadata read error")
		}
	})

	t.Run("metadata parse error", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		if err := env.meta.Set(context.Background(), depotMetadataKey("depot"), []byte("{")); err != nil {
			t.Fatalf("seed bad metadata: %v", err)
		}
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "new"}))},
			tarEntry{Name: "fw.bin", Data: []byte("new")},
		)
		if _, err := env.srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("uploadTar() expected metadata parse error")
		}
	})

	t.Run("existing rename to swap error", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeRelease("depot", Beta, "1.0.0", map[string]string{"fw.bin": "old"})
		store := newMockStore(t)
		store.base = env.store
		baseRename := store.base.Rename
		store.rename = func(oldName, newName string) error {
			if oldName == "depot/beta" && newName == "depot/beta.old" {
				return errors.New("boom")
			}
			return baseRename(oldName, newName)
		}
		srv := &Server{Store: store, MetadataStore: env.meta}
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.1.0", map[string]string{"fw.bin": "new"}))},
			tarEntry{Name: "fw.bin", Data: []byte("new")},
		)
		if _, err := srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("uploadTar() expected swap rename error")
		}
	})

	t.Run("rename error", func(t *testing.T) {
		t.Parallel()
		store := newMockStore(t)
		baseRename := store.base.Rename
		store.rename = func(oldName, newName string) error {
			if oldName == "depot/.tmp-beta" && newName == "depot/beta" {
				return errors.New("boom")
			}
			return baseRename(oldName, newName)
		}
		srv := &Server{Store: store, MetadataStore: kv.NewMemory(nil)}
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "new"}))},
			tarEntry{Name: "fw.bin", Data: []byte("new")},
		)
		if _, err := srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("uploadTar() expected rename error")
		}
	})

	t.Run("rename error restores previous release", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		env.writeRelease("depot", Beta, "1.0.0", map[string]string{"fw.bin": "old"})
		store := newMockStore(t)
		store.base = env.store
		baseRename := store.base.Rename
		store.rename = func(oldName, newName string) error {
			if oldName == "depot/.tmp-beta" && newName == "depot/beta" {
				return errors.New("boom")
			}
			return baseRename(oldName, newName)
		}
		srv := &Server{Store: store, MetadataStore: env.meta}
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.1.0", map[string]string{"fw.bin": "new"}))},
			tarEntry{Name: "fw.bin", Data: []byte("new")},
		)
		if _, err := srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("uploadTar() expected rename error")
		}
		if got := string(env.readFile("depot/beta/fw.bin")); got != "old" {
			t.Fatalf("restored file = %q", got)
		}
	})

	t.Run("metadata write error after rename", func(t *testing.T) {
		t.Parallel()
		store := newMockStore(t)
		base := store.base
		baseRename := base.Rename
		store.rename = func(oldName, newName string) error {
			return baseRename(oldName, newName)
		}
		meta := newMockKVStore()
		meta.set = func(context.Context, kv.Key, []byte) error { return errors.New("boom") }
		srv := &Server{Store: store, MetadataStore: meta}
		data := buildTar(t,
			tarEntry{Name: "manifest.json", Data: mustJSON(t, depotReleaseForFiles(Beta, "1.0.0", map[string]string{"fw.bin": "new"}))},
			tarEntry{Name: "fw.bin", Data: []byte("new")},
		)
		if _, err := srv.uploadTar(context.Background(), "depot", Beta, bytes.NewReader(data)); err == nil {
			t.Fatal("uploadTar() expected metadata write error")
		}
	})
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}
