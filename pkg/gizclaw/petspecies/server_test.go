package petspecies

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
	"github.com/GizClaw/gizclaw-go/pkg/store/objectstore"
)

func TestServerPutUploadDownloadAndList(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	srv := &Server{
		Store:  kv.NewMemory(nil),
		Assets: objectstore.Dir(t.TempDir()),
		Now: func() time.Time {
			now = now.Add(time.Second)
			return now
		},
	}
	item, err := srv.Put(ctx, "rabbit", apitypes.PetSpeciesSpec{Name: "Rabbit"})
	if err != nil {
		t.Fatalf("Put error = %v", err)
	}
	if item.Id != "rabbit" || item.Name != "Rabbit" || item.ZpetPath != "" {
		t.Fatalf("Put item = %#v", item)
	}
	data := []byte(`{"magic":"zpet","version":1,"id":"rabbit","canvas":[120,96],"format":"indexed_rle_u8_v1","clips":[{"id":"default"},{"id":"feed"}]}` + "\nPAYLOAD")
	item, err = srv.UploadZpet(ctx, "rabbit", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("UploadZpet error = %v", err)
	}
	if item.ZpetPath != "rabbit.zpet" || item.ZpetMetadata.SpeciesId != "rabbit" || item.ZpetMetadata.CanvasWidth != 120 || len(item.ZpetMetadata.ClipIds) != 2 {
		t.Fatalf("UploadZpet item = %#v", item)
	}
	r, err := srv.DownloadZpet(ctx, "rabbit")
	if err != nil {
		t.Fatalf("DownloadZpet error = %v", err)
	}
	defer r.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll error = %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("download bytes = %q, want %q", got, data)
	}

	if _, err := srv.Put(ctx, "cat", apitypes.PetSpeciesSpec{Name: "Cat"}); err != nil {
		t.Fatalf("Put cat error = %v", err)
	}
	items, hasNext, next, err := srv.List(ctx, "", 1)
	if err != nil {
		t.Fatalf("List page 1 error = %v", err)
	}
	if len(items) != 1 || !hasNext || next == nil {
		t.Fatalf("List page 1 = items %#v hasNext %v next %v", items, hasNext, next)
	}
	items, hasNext, _, err = srv.List(ctx, *next, 1)
	if err != nil {
		t.Fatalf("List page 2 error = %v", err)
	}
	if len(items) != 1 || hasNext {
		t.Fatalf("List page 2 = items %#v hasNext %v", items, hasNext)
	}
}

func TestParseZpetMetadataRejectsInvalidFiles(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(""),
		[]byte(`{"magic":"nope","version":1,"id":"rabbit","canvas":[1,2],"format":"x"}`),
		[]byte(`{"magic":"zpet","version":1,"canvas":[1,2],"format":"x"}`),
		[]byte(`not-json`),
	} {
		if _, err := ParseZpetMetadata(data); err == nil {
			t.Fatalf("ParseZpetMetadata(%q) error = nil, want error", data)
		}
	}
}

func TestServerGetUpdateDeleteAndConfigurationErrors(t *testing.T) {
	ctx := context.Background()
	srv := &Server{Store: kv.NewMemory(nil), Assets: objectstore.Dir(t.TempDir())}
	if _, err := srv.Put(ctx, " ", apitypes.PetSpeciesSpec{Name: "bad"}); err == nil {
		t.Fatalf("Put blank id error = nil, want error")
	}
	if _, err := srv.Put(ctx, "fox", apitypes.PetSpeciesSpec{}); err == nil {
		t.Fatalf("Put blank name error = nil, want error")
	}
	item, err := srv.Put(ctx, "fox", apitypes.PetSpeciesSpec{Name: "Fox", ZpetPath: stringPtr("custom/fox.zpet")})
	if err != nil {
		t.Fatalf("Put fox error = %v", err)
	}
	data := []byte(`{"magic":"zpet","version":2,"id":"fox","canvas":[64,32],"format":"mono","clips":[{"id":"idle"}]}` + "\nFOX")
	item, err = srv.UploadZpet(ctx, "fox", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("UploadZpet fox error = %v", err)
	}
	if item.ZpetPath != "custom/fox.zpet" || item.ZpetMetadata.Version != 2 {
		t.Fatalf("UploadZpet fox item = %#v", item)
	}
	got, err := srv.Get(ctx, "fox")
	if err != nil {
		t.Fatalf("Get fox error = %v", err)
	}
	if got.Name != "Fox" || got.ZpetPath != "custom/fox.zpet" {
		t.Fatalf("Get fox = %#v", got)
	}
	updated, err := srv.Put(ctx, "fox", apitypes.PetSpeciesSpec{Name: "Red Fox"})
	if err != nil {
		t.Fatalf("Put update fox error = %v", err)
	}
	if updated.Name != "Red Fox" || updated.ZpetPath != "custom/fox.zpet" || updated.ZpetMetadata.SpeciesId != "fox" {
		t.Fatalf("Put update fox = %#v", updated)
	}
	deleted, err := srv.Delete(ctx, "fox")
	if err != nil {
		t.Fatalf("Delete fox error = %v", err)
	}
	if deleted.Id != "fox" {
		t.Fatalf("Delete fox = %#v", deleted)
	}
	if _, err := srv.Get(ctx, "fox"); err == nil {
		t.Fatalf("Get deleted fox error = nil, want error")
	}
	if _, err := srv.DownloadZpet(ctx, "fox"); err == nil {
		t.Fatalf("Download deleted fox error = nil, want error")
	}

	if _, _, _, err := (&Server{}).List(ctx, "", 0); err == nil {
		t.Fatalf("List without store error = nil, want error")
	}
	if _, err := (&Server{Store: kv.NewMemory(nil)}).UploadZpet(ctx, "fox", bytes.NewReader(data)); err == nil {
		t.Fatalf("UploadZpet without assets error = nil, want error")
	}
}

func stringPtr(value string) *string {
	return &value
}
