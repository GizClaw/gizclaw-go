package firmware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	root := t.TempDir()
	store := NewStore(root)
	return NewManager(store, "/firmwares")
}

func seedDepot(t *testing.T, m *Manager, depot string, payload []byte) {
	t.Helper()
	if err := m.uploader.PutInfo(depot, DepotInfo{
		Files: []DepotInfoFile{{Path: "firmware.bin"}},
	}); err != nil {
		t.Fatalf("PutInfo: %v", err)
	}
	for _, tc := range []struct {
		ver string
		ch  Channel
	}{
		{"1.0.0", ChannelStable},
		{"1.1.0", ChannelBeta},
		{"1.2.0", ChannelTesting},
		{"0.9.0", ChannelRollback},
	} {
		tar := buildReleaseTar(t, mustRelease(t, tc.ver, tc.ch, payload), map[string][]byte{"firmware.bin": payload})
		if _, err := m.uploader.UploadTar(depot, tc.ch, bytes.NewReader(tar)); err != nil {
			t.Fatalf("UploadTar %s: %v", tc.ch, err)
		}
	}
}

func TestAdminHandler_ListDepots(t *testing.T) {
	m := newTestManager(t)
	seedDepot(t, m, "demo", []byte("fw"))
	handler := m.AdminHandler()

	req := httptest.NewRequest(http.MethodGet, "/firmwares", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/firmwares", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for wrong method, got %d", rr.Code)
	}
}

func TestAdminHandler_GetDepot(t *testing.T) {
	m := newTestManager(t)
	seedDepot(t, m, "demo", []byte("fw"))
	handler := m.AdminHandler()

	req := httptest.NewRequest(http.MethodGet, "/firmwares/demo", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/firmwares/missing", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing depot, got %d", rr.Code)
	}
}

func TestAdminHandler_PutDepotInfo(t *testing.T) {
	m := newTestManager(t)
	handler := m.AdminHandler()

	body, _ := json.Marshal(DepotInfo{Files: []DepotInfoFile{{Path: "firmware.bin"}}})
	req := httptest.NewRequest(http.MethodPut, "/firmwares/new-depot", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/firmwares/new-depot", bytes.NewReader([]byte("{")))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", rr.Code)
	}

	payload := []byte("fw")
	tar := buildReleaseTar(t, mustRelease(t, "1.0.0", ChannelStable, payload), map[string][]byte{"firmware.bin": payload})
	if _, err := m.uploader.UploadTar("new-depot", ChannelStable, bytes.NewReader(tar)); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPut, "/firmwares/new-depot", bytes.NewReader([]byte(`{"files":[{"path":"other.bin"}]}`)))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 for info mismatch, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/firmwares/new-depot", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for unsupported method, got %d", rr.Code)
	}
}

func TestAdminHandler_GetChannel(t *testing.T) {
	m := newTestManager(t)
	seedDepot(t, m, "demo", []byte("fw"))
	handler := m.AdminHandler()

	req := httptest.NewRequest(http.MethodGet, "/firmwares/demo/stable", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/firmwares/demo/unknown", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown channel, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/firmwares/demo/stable", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for unsupported method, got %d", rr.Code)
	}
}

func TestAdminHandler_UploadChannel(t *testing.T) {
	m := newTestManager(t)
	handler := m.AdminHandler()

	if err := m.uploader.PutInfo("demo", DepotInfo{
		Files: []DepotInfoFile{{Path: "firmware.bin"}},
	}); err != nil {
		t.Fatal(err)
	}

	payload := []byte("fw")
	tar := buildReleaseTar(t, mustRelease(t, "1.0.0", ChannelStable, payload), map[string][]byte{"firmware.bin": payload})
	req := httptest.NewRequest(http.MethodPut, "/firmwares/demo/stable", bytes.NewReader(tar))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/firmwares/demo/stable", bytes.NewReader([]byte("bad")))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 for bad upload, got %d", rr.Code)
	}
}

func TestAdminHandler_ReleaseAndRollback(t *testing.T) {
	m := newTestManager(t)
	seedDepot(t, m, "demo", []byte("fw"))
	handler := m.AdminHandler()

	req := httptest.NewRequest(http.MethodPut, "/firmwares/demo:release", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("release status = %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/firmwares/demo:rollback", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("rollback status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminHandler_SwitchErrors(t *testing.T) {
	m := newTestManager(t)
	handler := m.AdminHandler()

	req := httptest.NewRequest(http.MethodPut, "/firmwares/missing:release", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/firmwares/missing:rollback", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := m.uploader.PutInfo("no-beta", DepotInfo{
		Files: []DepotInfoFile{{Path: "firmware.bin"}},
	}); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPut, "/firmwares/no-beta:release", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/firmwares/no-beta:rollback", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminHandler_InternalErrors(t *testing.T) {
	m := newTestManager(t)
	seedDepot(t, m, "demo", []byte("fw"))
	handler := m.AdminHandler()

	if err := os.WriteFile(m.store.ManifestPath("demo", ChannelBeta), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/firmwares/demo:release", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}

	if err := os.WriteFile(m.store.ManifestPath("demo", ChannelRollback), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPut, "/firmwares/demo:rollback", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminHandler_UnmatchedRoutes(t *testing.T) {
	m := newTestManager(t)
	handler := m.AdminHandler()

	req := httptest.NewRequest(http.MethodGet, "/other-prefix/foo", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched route, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/firmwares/a/b/c", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for too many segments, got %d", rr.Code)
	}
}
