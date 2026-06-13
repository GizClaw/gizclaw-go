package static

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestHandlerCachesStaticAssetsWithETag(t *testing.T) {
	uiFS := fstest.MapFS{
		"index.html": {Data: []byte("index")},
		"app.js":     {Data: []byte("app")},
	}
	handler := Handler(fs.FS(uiFS))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app.js status = %d, want %d", rec.Code, http.StatusOK)
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("GET /app.js missing ETag")
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=0, must-revalidate" {
		t.Fatalf("GET /app.js Cache-Control = %q, want revalidate cache", got)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	req.Header.Set("If-None-Match", etag)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotModified {
		t.Fatalf("GET /app.js with matching ETag status = %d, want %d", rec.Code, http.StatusNotModified)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing-route", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /missing-route status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("GET /missing-route Cache-Control = %q, want no-cache", got)
	}
}
