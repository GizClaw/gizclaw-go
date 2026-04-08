package firmware

import (
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := NewStore(filepath.Join(root, "depots"))
	m := NewManager(store, "/firmwares")
	if m == nil {
		t.Fatal("nil manager")
	}
	if m.store != store || m.scanner == nil || m.uploader == nil || m.switcher == nil || m.ota == nil || m.adminMux == nil {
		t.Fatalf("unexpected fields: %+v", m)
	}
}
