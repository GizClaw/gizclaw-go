package depotstore

import (
	"path/filepath"
	"testing"
)

func TestDirStoreOperations(t *testing.T) {
	root := t.TempDir()
	store := Dir(root)

	if err := store.WriteFile("demo/manifest.json", []byte("manifest")); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	data, err := store.ReadFile("demo/manifest.json")
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if string(data) != "manifest" {
		t.Fatalf("ReadFile data = %q", data)
	}

	file, err := store.Open("demo/manifest.json")
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	_ = file.Close()

	info, err := store.Stat("demo/manifest.json")
	if err != nil {
		t.Fatalf("Stat error = %v", err)
	}
	if info.Name() != "manifest.json" {
		t.Fatalf("Stat name = %q", info.Name())
	}

	if err := store.MkdirAll("demo/testing"); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := store.WriteFile("demo/testing/fw.bin", []byte("fw")); err != nil {
		t.Fatalf("WriteFile nested error = %v", err)
	}

	if err := store.Rename("demo/testing/fw.bin", "demo/stable/fw.bin"); err != nil {
		t.Fatalf("Rename error = %v", err)
	}
	if _, err := store.Stat("demo/stable/fw.bin"); err != nil {
		t.Fatalf("Stat renamed file error = %v", err)
	}

	if got := store.abs("./demo/manifest.json"); got != filepath.Join(root, "demo", "manifest.json") {
		t.Fatalf("abs(./demo/manifest.json) = %q", got)
	}
	if got := store.abs(""); got != root {
		t.Fatalf("abs(\"\") = %q, want %q", got, root)
	}

	if err := store.RemoveAll("demo/stable"); err != nil {
		t.Fatalf("RemoveAll error = %v", err)
	}
	if _, err := store.Stat("demo/stable/fw.bin"); err == nil {
		t.Fatal("Stat should fail after RemoveAll")
	}
}
