package static

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"
)

// Handler serves embedded UI assets and falls back to index.html for
// client-side routes so BrowserRouter deep links work.
func Handler(uiFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(uiFS))
	etags := make(map[string]string)
	var etagsMu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean != "" {
			if _, err := fs.Stat(uiFS, clean); err == nil {
				if handleCacheHeaders(w, r, uiFS, etags, &etagsMu, clean) {
					return
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		r2 := r.Clone(r.Context())
		r2.URL = r.URL
		u := *r.URL
		u.Path = "/"
		r2.URL = &u
		fileServer.ServeHTTP(w, r2)
	})
}

func handleCacheHeaders(w http.ResponseWriter, r *http.Request, uiFS fs.FS, etags map[string]string, etagsMu *sync.Mutex, name string) bool {
	if name == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
		return false
	}
	etagsMu.Lock()
	etag, ok := etags[name]
	etagsMu.Unlock()
	if !ok {
		data, err := fs.ReadFile(uiFS, name)
		if err != nil {
			w.Header().Set("Cache-Control", "no-cache")
			return false
		}
		sum := sha256.Sum256(data)
		etag = `"` + hex.EncodeToString(sum[:]) + `"`
		etagsMu.Lock()
		etags[name] = etag
		etagsMu.Unlock()
	}
	w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}
