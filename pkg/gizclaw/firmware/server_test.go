package firmware

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/adminservice"
	"github.com/GizClaw/gizclaw-go/pkg/gizclaw/api/apitypes"
	"github.com/GizClaw/gizclaw-go/pkg/store/kv"
)

func TestServerCRUDReleaseRollback(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	s := &Server{Store: kv.NewMemory(nil), Now: func() time.Time { return now }}

	create, err := s.CreateFirmware(ctx, adminservice.CreateFirmwareRequestObject{Body: ptr(firmwareUpsert("devkit", "stable-1", "beta-1", "develop-1", "pending-1"))})
	if err != nil {
		t.Fatalf("CreateFirmware error = %v", err)
	}
	if _, ok := create.(adminservice.CreateFirmware200JSONResponse); !ok {
		t.Fatalf("CreateFirmware response = %T", create)
	}

	released, err := s.ReleaseFirmware(ctx, adminservice.ReleaseFirmwareRequestObject{Name: "devkit"})
	if err != nil {
		t.Fatalf("ReleaseFirmware error = %v", err)
	}
	releasedItem := apitypes.Firmware(released.(adminservice.ReleaseFirmware200JSONResponse))
	if got := slotVersion(releasedItem.Slots.Stable); got != "beta-1" {
		t.Fatalf("released stable = %q", got)
	}
	if got := slotVersion(releasedItem.Slots.Rollback); got != "stable-1" {
		t.Fatalf("released rollback = %q", got)
	}
	if slotVersion(releasedItem.Slots.Pending) != "" {
		t.Fatalf("released pending should be empty: %+v", releasedItem.Slots.Pending)
	}

	rolledBack, err := s.RollbackFirmware(ctx, adminservice.RollbackFirmwareRequestObject{Name: "devkit"})
	if err != nil {
		t.Fatalf("RollbackFirmware error = %v", err)
	}
	rolledBackItem := apitypes.Firmware(rolledBack.(adminservice.RollbackFirmware200JSONResponse))
	if got := slotVersion(rolledBackItem.Slots.Stable); got != "stable-1" {
		t.Fatalf("rolled back stable = %q", got)
	}

	list, err := s.ListFirmwares(ctx, adminservice.ListFirmwaresRequestObject{})
	if err != nil {
		t.Fatalf("ListFirmwares error = %v", err)
	}
	if got := len(adminservice.FirmwareList(list.(adminservice.ListFirmwares200JSONResponse)).Items); got != 1 {
		t.Fatalf("ListFirmwares len = %d", got)
	}
}

func TestServerRejectsOperationLeavingStableEmpty(t *testing.T) {
	ctx := context.Background()
	s := &Server{Store: kv.NewMemory(nil)}
	if _, err := s.CreateFirmware(ctx, adminservice.CreateFirmwareRequestObject{Body: ptr(firmwareUpsert("devkit", "stable-1", "", "", ""))}); err != nil {
		t.Fatalf("CreateFirmware error = %v", err)
	}
	if response, err := s.ReleaseFirmware(ctx, adminservice.ReleaseFirmwareRequestObject{Name: "devkit"}); err != nil {
		t.Fatalf("ReleaseFirmware error = %v", err)
	} else if _, ok := response.(adminservice.ReleaseFirmware409JSONResponse); !ok {
		t.Fatalf("ReleaseFirmware response = %T, want 409", response)
	}
	if response, err := s.RollbackFirmware(ctx, adminservice.RollbackFirmwareRequestObject{Name: "devkit"}); err != nil {
		t.Fatalf("RollbackFirmware error = %v", err)
	} else if _, ok := response.(adminservice.RollbackFirmware409JSONResponse); !ok {
		t.Fatalf("RollbackFirmware response = %T, want 409", response)
	}
}

func TestServerPutGetDeleteFirmware(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	nextTime := createdAt
	s := &Server{
		Store: kv.NewMemory(nil),
		Now: func() time.Time {
			out := nextTime
			nextTime = updatedAt
			return out
		},
	}

	put, err := s.PutFirmware(ctx, adminservice.PutFirmwareRequestObject{
		Name: "devkit",
		Body: ptr(firmwareUpsertWithArtifact("devkit", "1.0.0")),
	})
	if err != nil {
		t.Fatalf("PutFirmware error = %v", err)
	}
	putItem := apitypes.Firmware(put.(adminservice.PutFirmware200JSONResponse))
	if putItem.CreatedAt != createdAt || putItem.UpdatedAt != createdAt {
		t.Fatalf("first put timestamps = %s/%s, want %s", putItem.CreatedAt, putItem.UpdatedAt, createdAt)
	}

	update := firmwareUpsertWithArtifact("devkit", "1.1.0")
	description := " updated firmware "
	update.Description = &description
	updated, err := s.PutFirmware(ctx, adminservice.PutFirmwareRequestObject{Name: "devkit", Body: ptr(update)})
	if err != nil {
		t.Fatalf("PutFirmware update error = %v", err)
	}
	updatedItem := apitypes.Firmware(updated.(adminservice.PutFirmware200JSONResponse))
	if updatedItem.CreatedAt != createdAt || updatedItem.UpdatedAt != updatedAt {
		t.Fatalf("updated timestamps = %s/%s, want %s/%s", updatedItem.CreatedAt, updatedItem.UpdatedAt, createdAt, updatedAt)
	}
	if updatedItem.Description == nil || *updatedItem.Description != "updated firmware" {
		t.Fatalf("updated description = %v", updatedItem.Description)
	}

	got, err := s.GetFirmware(ctx, adminservice.GetFirmwareRequestObject{Name: "devkit"})
	if err != nil {
		t.Fatalf("GetFirmware error = %v", err)
	}
	if item := apitypes.Firmware(got.(adminservice.GetFirmware200JSONResponse)); slotVersion(item.Slots.Stable) != "1.1.0" {
		t.Fatalf("GetFirmware stable = %+v", item.Slots.Stable)
	}

	deleted, err := s.DeleteFirmware(ctx, adminservice.DeleteFirmwareRequestObject{Name: "devkit"})
	if err != nil {
		t.Fatalf("DeleteFirmware error = %v", err)
	}
	if item := apitypes.Firmware(deleted.(adminservice.DeleteFirmware200JSONResponse)); item.Name != "devkit" {
		t.Fatalf("DeleteFirmware item = %+v", item)
	}
	if response, err := s.GetFirmware(ctx, adminservice.GetFirmwareRequestObject{Name: "devkit"}); err != nil {
		t.Fatalf("GetFirmware after delete error = %v", err)
	} else if _, ok := response.(adminservice.GetFirmware404JSONResponse); !ok {
		t.Fatalf("GetFirmware after delete response = %T, want 404", response)
	}
}

func TestServerCreateAndPutValidation(t *testing.T) {
	ctx := context.Background()
	s := &Server{Store: kv.NewMemory(nil)}

	if response, err := s.CreateFirmware(ctx, adminservice.CreateFirmwareRequestObject{}); err != nil {
		t.Fatalf("CreateFirmware nil body error = %v", err)
	} else if _, ok := response.(adminservice.CreateFirmware400JSONResponse); !ok {
		t.Fatalf("CreateFirmware nil body response = %T, want 400", response)
	}
	if response, err := s.CreateFirmware(ctx, adminservice.CreateFirmwareRequestObject{Body: ptr(firmwareUpsert("", "", "", "", ""))}); err != nil {
		t.Fatalf("CreateFirmware empty name error = %v", err)
	} else if _, ok := response.(adminservice.CreateFirmware400JSONResponse); !ok {
		t.Fatalf("CreateFirmware empty name response = %T, want 400", response)
	}
	if response, err := s.PutFirmware(ctx, adminservice.PutFirmwareRequestObject{Name: "devkit", Body: ptr(firmwareUpsert("other", "", "", "", ""))}); err != nil {
		t.Fatalf("PutFirmware name mismatch error = %v", err)
	} else if _, ok := response.(adminservice.PutFirmware400JSONResponse); !ok {
		t.Fatalf("PutFirmware name mismatch response = %T, want 400", response)
	}
	if _, err := s.CreateFirmware(ctx, adminservice.CreateFirmwareRequestObject{Body: ptr(firmwareUpsertWithArtifact("devkit", "1.0.0"))}); err != nil {
		t.Fatalf("CreateFirmware first error = %v", err)
	}
	if response, err := s.CreateFirmware(ctx, adminservice.CreateFirmwareRequestObject{Body: ptr(firmwareUpsertWithArtifact("devkit", "1.0.0"))}); err != nil {
		t.Fatalf("CreateFirmware duplicate error = %v", err)
	} else if _, ok := response.(adminservice.CreateFirmware409JSONResponse); !ok {
		t.Fatalf("CreateFirmware duplicate response = %T, want 409", response)
	}
}

func TestServerRejectsInvalidArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		artifact apitypes.FirmwareArtifact
		want     string
	}{
		{name: "missing name", artifact: apitypes.FirmwareArtifact{Kind: apitypes.FirmwareArtifactKindApp, Url: "https://example.test/app.bin"}, want: "name is required"},
		{name: "missing kind", artifact: apitypes.FirmwareArtifact{Name: "main", Url: "https://example.test/app.bin"}, want: "kind is required"},
		{name: "bad kind", artifact: apitypes.FirmwareArtifact{Name: "main", Kind: apitypes.FirmwareArtifactKind("other"), Url: "https://example.test/app.bin"}, want: "unsupported kind"},
		{name: "missing url", artifact: apitypes.FirmwareArtifact{Name: "main", Kind: apitypes.FirmwareArtifactKindApp}, want: "url is required"},
		{name: "negative size", artifact: apitypes.FirmwareArtifact{Name: "main", Kind: apitypes.FirmwareArtifactKindApp, Url: "https://example.test/app.bin", Size: int64Ptr(-1)}, want: "size must be non-negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := firmwareUpsertWithArtifact("devkit", "1.0.0")
			req.Slots.Stable.Artifacts = &[]apitypes.FirmwareArtifact{tt.artifact}
			_, err := normalizeFirmwareUpsert(req, "")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("normalizeFirmwareUpsert error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestServerListFirmwaresPagination(t *testing.T) {
	ctx := context.Background()
	s := &Server{Store: kv.NewMemory(nil)}
	for _, name := range []string{"devkit", "p4_func_ev", "waveshare"} {
		if _, err := s.CreateFirmware(ctx, adminservice.CreateFirmwareRequestObject{Body: ptr(firmwareUpsertWithArtifact(name, "1.0.0"))}); err != nil {
			t.Fatalf("CreateFirmware(%s) error = %v", name, err)
		}
	}

	limit := int32(2)
	first, err := s.ListFirmwares(ctx, adminservice.ListFirmwaresRequestObject{Params: adminservice.ListFirmwaresParams{Limit: &limit}})
	if err != nil {
		t.Fatalf("ListFirmwares first error = %v", err)
	}
	firstPage := adminservice.FirmwareList(first.(adminservice.ListFirmwares200JSONResponse))
	if len(firstPage.Items) != 2 || !firstPage.HasNext || firstPage.NextCursor == nil {
		t.Fatalf("first page = %+v", firstPage)
	}

	second, err := s.ListFirmwares(ctx, adminservice.ListFirmwaresRequestObject{Params: adminservice.ListFirmwaresParams{Cursor: firstPage.NextCursor, Limit: &limit}})
	if err != nil {
		t.Fatalf("ListFirmwares second error = %v", err)
	}
	secondPage := adminservice.FirmwareList(second.(adminservice.ListFirmwares200JSONResponse))
	if len(secondPage.Items) != 1 || secondPage.HasNext || secondPage.NextCursor != nil {
		t.Fatalf("second page = %+v", secondPage)
	}
}

func TestServerStoreNotConfigured(t *testing.T) {
	ctx := context.Background()
	s := &Server{}
	if response, err := s.ListFirmwares(ctx, adminservice.ListFirmwaresRequestObject{}); err != nil {
		t.Fatalf("ListFirmwares error = %v", err)
	} else if _, ok := response.(adminservice.ListFirmwares500JSONResponse); !ok {
		t.Fatalf("ListFirmwares response = %T, want 500", response)
	}
	if response, err := s.GetFirmware(ctx, adminservice.GetFirmwareRequestObject{Name: "devkit"}); err != nil {
		t.Fatalf("GetFirmware error = %v", err)
	} else if _, ok := response.(adminservice.GetFirmware500JSONResponse); !ok {
		t.Fatalf("GetFirmware response = %T, want 500", response)
	}
}

func firmwareUpsert(name, stable, beta, develop, pending string) adminservice.FirmwareUpsert {
	return adminservice.FirmwareUpsert{
		Name: name,
		Slots: apitypes.FirmwareSlots{
			Stable:  firmwareSlot(stable),
			Beta:    firmwareSlot(beta),
			Develop: firmwareSlot(develop),
			Pending: firmwareSlot(pending),
		},
	}
}

func firmwareUpsertWithArtifact(name, stable string) adminservice.FirmwareUpsert {
	req := firmwareUpsert(name, stable, "", "", "")
	sha256 := strings.Repeat("a", 64)
	size := int64(42)
	req.Slots.Stable.Artifacts = &[]apitypes.FirmwareArtifact{{
		Name:   "main",
		Kind:   apitypes.FirmwareArtifactKindApp,
		Url:    "https://firmware.example/" + name + "/" + stable + "/app.bin",
		Sha256: &sha256,
		Size:   &size,
	}}
	return req
}

func firmwareSlot(version string) apitypes.FirmwareSlot {
	if version == "" {
		return apitypes.FirmwareSlot{}
	}
	return apitypes.FirmwareSlot{Version: &version}
}

func slotVersion(slot apitypes.FirmwareSlot) string {
	if slot.Version == nil {
		return ""
	}
	return *slot.Version
}

func ptr[T any](value T) *T {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
