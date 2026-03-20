package objstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// fileMeta is the on-disk JSON structure stored alongside each object.
type fileMeta struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// FS is a local-filesystem-backed object store. Each key maps to a
// subdirectory under root containing a "data" file and a "meta.json" file.
//
// FS is safe for concurrent use. However, the Object.Content returned by Get
// is read outside the internal lock; a concurrent Put to the same key may
// cause the reader to observe partial content. Callers that need
// read-after-write consistency on the same key should synchronize externally.
type FS struct {
	root string
	mu   sync.RWMutex
}

// NewFS creates a filesystem-backed Store rooted at the given directory.
// The directory is created if it does not exist.
func NewFS(root string) (*FS, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("objstore: resolve root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("objstore: create root: %w", err)
	}
	return &FS{root: abs}, nil
}

// compile-time check
var _ Store = (*FS)(nil)

func (fs *FS) Put(_ context.Context, key string, obj Object) error {
	if err := validateKey(key); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := fs.keyDir(key)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("objstore: mkdir %q: %w", dir, err)
	}

	dataPath := filepath.Join(dir, "data")
	tmpDataPath := dataPath + ".tmp"
	f, err := os.Create(tmpDataPath)
	if err != nil {
		return fmt.Errorf("objstore: create data: %w", err)
	}
	n, copyErr := io.Copy(f, obj.Content)
	if closeErr := f.Close(); closeErr != nil && copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		os.Remove(tmpDataPath)
		return fmt.Errorf("objstore: write data: %w", copyErr)
	}

	meta := fileMeta{
		Name:        obj.Name,
		ContentType: obj.ContentType,
		Size:        n,
	}
	metaData, err := json.Marshal(meta)
	if err != nil {
		os.Remove(tmpDataPath)
		return fmt.Errorf("objstore: marshal meta: %w", err)
	}
	metaPath := filepath.Join(dir, "meta.json")
	tmpMetaPath := metaPath + ".tmp"
	if err := os.WriteFile(tmpMetaPath, metaData, 0o644); err != nil {
		os.Remove(tmpDataPath)
		return fmt.Errorf("objstore: write meta: %w", err)
	}

	// Rename data first, then meta. On same-filesystem renames are atomic,
	// so an existing object is never left in a half-written state.
	//
	// Known limitation: if the data rename succeeds but the meta rename
	// fails, the on-disk state will have new data with stale (or missing)
	// meta.json. This window is extremely narrow on a local filesystem
	// (same-device rename rarely fails independently) and is accepted as
	// a trade-off to avoid a more complex journaling scheme.
	if err := os.Rename(tmpDataPath, dataPath); err != nil {
		os.Remove(tmpDataPath)
		os.Remove(tmpMetaPath)
		return fmt.Errorf("objstore: commit data: %w", err)
	}
	if err := os.Rename(tmpMetaPath, metaPath); err != nil {
		os.Remove(tmpMetaPath)
		return fmt.Errorf("objstore: commit meta: %w", err)
	}
	return nil
}

func (fs *FS) Get(_ context.Context, key string) (Object, error) {
	if err := validateKey(key); err != nil {
		return Object{}, err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	dir := fs.keyDir(key)
	meta, err := fs.readMeta(dir)
	if err != nil {
		return Object{}, err
	}

	f, err := os.Open(filepath.Join(dir, "data"))
	if err != nil {
		if os.IsNotExist(err) {
			return Object{}, ErrNotFound
		}
		return Object{}, fmt.Errorf("objstore: open data: %w", err)
	}
	return Object{
		Name:        meta.Name,
		ContentType: meta.ContentType,
		Size:        meta.Size,
		Content:     f,
	}, nil
}

func (fs *FS) Delete(_ context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := fs.keyDir(key)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("objstore: delete %q: %w", key, err)
	}
	return nil
}

func (fs *FS) List(_ context.Context, prefix string) iter.Seq2[ObjectInfo, error] {
	return func(yield func(ObjectInfo, error) bool) {
		fs.mu.RLock()
		defer fs.mu.RUnlock()

		entries, err := os.ReadDir(fs.root)
		if err != nil {
			if os.IsNotExist(err) {
				return
			}
			yield(ObjectInfo{}, fmt.Errorf("objstore: read root: %w", err))
			return
		}

		keys := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if prefix != "" && !strings.HasPrefix(name, prefix) {
				continue
			}
			keys = append(keys, name)
		}
		sort.Strings(keys)

		for _, key := range keys {
			dir := fs.keyDir(key)
			meta, err := fs.readMeta(dir)
			if err != nil {
				if !yield(ObjectInfo{}, err) {
					return
				}
				continue
			}
			if !yield(ObjectInfo{
				Key:         key,
				Name:        meta.Name,
				ContentType: meta.ContentType,
				Size:        meta.Size,
			}, nil) {
				return
			}
		}
	}
}

func (fs *FS) Close() error { return nil }

// keyDir returns the directory path for a given key.
func (fs *FS) keyDir(key string) string {
	return filepath.Join(fs.root, key)
}

// readMeta reads and decodes the meta.json file from a key directory.
func (fs *FS) readMeta(dir string) (fileMeta, error) {
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return fileMeta{}, ErrNotFound
		}
		return fileMeta{}, fmt.Errorf("objstore: read meta: %w", err)
	}
	var meta fileMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return fileMeta{}, fmt.Errorf("objstore: decode meta: %w", err)
	}
	return meta, nil
}

// validateKey rejects empty keys and path-traversal attempts.
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("objstore: empty key")
	}
	if strings.Contains(key, `\`) {
		return fmt.Errorf("objstore: invalid key %q", key)
	}
	if filepath.IsAbs(key) || strings.HasPrefix(key, "/") {
		return fmt.Errorf("objstore: invalid key %q", key)
	}
	if cleaned := filepath.Clean(key); cleaned != key {
		return fmt.Errorf("objstore: invalid key %q", key)
	}
	if key == "." || strings.Contains(key, "..") {
		return fmt.Errorf("objstore: invalid key %q", key)
	}
	if strings.Contains(key, "/") {
		return fmt.Errorf("objstore: invalid key %q", key)
	}
	return nil
}
