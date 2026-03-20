package objstore_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haivivi/giztoy/go/pkg/objstore"
)

func newTestStore(t *testing.T) objstore.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := objstore.NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func makeObject(name, contentType, body string) objstore.Object {
	return objstore.Object{
		Name:        name,
		ContentType: contentType,
		Size:        int64(len(body)),
		Content:     io.NopCloser(strings.NewReader(body)),
	}
}

func readObject(t *testing.T, obj objstore.Object) string {
	t.Helper()
	defer obj.Content.Close()
	data, err := io.ReadAll(obj.Content)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	return string(data)
}

func TestPutGetDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Get non-existent key.
	_, err := s.Get(ctx, "missing")
	if !errors.Is(err, objstore.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Put and Get.
	obj := makeObject("hello.txt", "text/plain", "hello world")
	if err := s.Put(ctx, "doc1", obj); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := s.Get(ctx, "doc1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "hello.txt" {
		t.Errorf("Name = %q, want %q", got.Name, "hello.txt")
	}
	if got.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, want %q", got.ContentType, "text/plain")
	}
	body := readObject(t, got)
	if body != "hello world" {
		t.Errorf("body = %q, want %q", body, "hello world")
	}

	// Overwrite.
	obj2 := makeObject("updated.txt", "text/html", "<h1>hi</h1>")
	if err := s.Put(ctx, "doc1", obj2); err != nil {
		t.Fatalf("Put overwrite: %v", err)
	}
	got2, err := s.Get(ctx, "doc1")
	if err != nil {
		t.Fatalf("Get after overwrite: %v", err)
	}
	if got2.Name != "updated.txt" {
		t.Errorf("Name after overwrite = %q, want %q", got2.Name, "updated.txt")
	}
	body2 := readObject(t, got2)
	if body2 != "<h1>hi</h1>" {
		t.Errorf("body after overwrite = %q, want %q", body2, "<h1>hi</h1>")
	}

	// Delete.
	if err := s.Delete(ctx, "doc1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = s.Get(ctx, "doc1")
	if !errors.Is(err, objstore.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	// Delete non-existent key should not error.
	if err := s.Delete(ctx, "no-such-key"); err != nil {
		t.Fatalf("Delete non-existent: %v", err)
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	objects := map[string]objstore.Object{
		"alpha":   makeObject("a.txt", "text/plain", "aaa"),
		"beta":    makeObject("b.bin", "application/octet-stream", "bbb"),
		"alpha2":  makeObject("a2.txt", "text/plain", "aaa2"),
		"charlie": makeObject("c.jpg", "image/jpeg", "ccc"),
	}
	for key, obj := range objects {
		if err := s.Put(ctx, key, obj); err != nil {
			t.Fatalf("Put %s: %v", key, err)
		}
	}

	// List all (empty prefix).
	var allKeys []string
	for info, err := range s.List(ctx, "") {
		if err != nil {
			t.Fatalf("List all: %v", err)
		}
		allKeys = append(allKeys, info.Key)
	}
	if len(allKeys) != 4 {
		t.Fatalf("List all: got %d entries, want 4: %v", len(allKeys), allKeys)
	}

	// List with prefix "alpha" — should match "alpha" and "alpha2".
	var prefixed []string
	for info, err := range s.List(ctx, "alpha") {
		if err != nil {
			t.Fatalf("List alpha: %v", err)
		}
		prefixed = append(prefixed, info.Key)
	}
	if len(prefixed) != 2 {
		t.Fatalf("List alpha: got %d entries, want 2: %v", len(prefixed), prefixed)
	}

	// Verify lexicographic order.
	if prefixed[0] != "alpha" || prefixed[1] != "alpha2" {
		t.Errorf("List alpha order = %v, want [alpha, alpha2]", prefixed)
	}

	// List with prefix that matches nothing.
	var empty []string
	for info, err := range s.List(ctx, "zzz") {
		if err != nil {
			t.Fatalf("List zzz: %v", err)
		}
		empty = append(empty, info.Key)
	}
	if len(empty) != 0 {
		t.Fatalf("List zzz: got %d entries, want 0", len(empty))
	}
}

func TestListMetadata(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.Put(ctx, "img", makeObject("photo.png", "image/png", "pixels")); err != nil {
		t.Fatal(err)
	}

	for info, err := range s.List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		if info.Key != "img" {
			t.Errorf("Key = %q, want %q", info.Key, "img")
		}
		if info.Name != "photo.png" {
			t.Errorf("Name = %q, want %q", info.Name, "photo.png")
		}
		if info.ContentType != "image/png" {
			t.Errorf("ContentType = %q, want %q", info.ContentType, "image/png")
		}
		if info.Size != 6 {
			t.Errorf("Size = %d, want 6", info.Size)
		}
	}
}

func TestPutSizeComputed(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	obj := objstore.Object{
		Name:        "test.dat",
		ContentType: "application/octet-stream",
		Size:        0,
		Content:     io.NopCloser(bytes.NewReader([]byte("12345"))),
	}
	if err := s.Put(ctx, "sized", obj); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get(ctx, "sized")
	if err != nil {
		t.Fatal(err)
	}
	defer got.Content.Close()
	if got.Size != 5 {
		t.Errorf("Size = %d, want 5", got.Size)
	}
}

func TestInvalidKeys(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	cases := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"dot-dot", ".."},
		{"absolute", "/etc/passwd"},
		{"traversal", "foo/../../etc"},
		{"slash", "a/b"},
		{"backslash", `a\b`},
		{"dot", "."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Put(ctx, tc.key, makeObject("x", "x", "x"))
			if err == nil {
				t.Error("Put: expected error for invalid key")
			}
			_, err = s.Get(ctx, tc.key)
			if err == nil {
				t.Error("Get: expected error for invalid key")
			}
			err = s.Delete(ctx, tc.key)
			if err == nil {
				t.Error("Delete: expected error for invalid key")
			}
		})
	}
}

func TestListEarlyBreak(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	for _, k := range []string{"a", "b", "c"} {
		if err := s.Put(ctx, k, makeObject(k+".txt", "text/plain", k)); err != nil {
			t.Fatal(err)
		}
	}

	var count int
	for _, err := range s.List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		count++
		if count == 1 {
			break
		}
	}
	if count != 1 {
		t.Errorf("expected 1 item after break, got %d", count)
	}
}

func TestGetCorruptedMeta(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := objstore.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Put(ctx, "bad", makeObject("x", "x", "data")); err != nil {
		t.Fatal(err)
	}

	// Corrupt the meta.json.
	metaPath := filepath.Join(dir, "bad", "meta.json")
	if err := os.WriteFile(metaPath, []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = s.Get(ctx, "bad")
	if err == nil {
		t.Fatal("expected error for corrupted meta")
	}
	if errors.Is(err, objstore.ErrNotFound) {
		t.Fatal("expected non-ErrNotFound error for corrupted meta")
	}
}

func TestListCorruptedMeta(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := objstore.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Put(ctx, "item", makeObject("x", "x", "data")); err != nil {
		t.Fatal(err)
	}

	// Corrupt the meta.json.
	metaPath := filepath.Join(dir, "item", "meta.json")
	if err := os.WriteFile(metaPath, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	var sawErr bool
	for _, err := range s.List(ctx, "") {
		if err != nil {
			sawErr = true
		}
	}
	if !sawErr {
		t.Fatal("expected error during List with corrupted meta")
	}
}

func TestNewFSCreatesDir(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "deep", "nested", "dir")
	s, err := objstore.NewFS(nested)
	if err != nil {
		t.Fatalf("NewFS nested: %v", err)
	}
	s.Close()

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory to be created")
	}
}

func TestClose(t *testing.T) {
	s := newTestStore(t)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPutReadOnlyRoot(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	root := filepath.Join(base, "store")
	s, err := objstore.NewFS(root)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(root, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(root, 0o755) })

	err = s.Put(ctx, "key", makeObject("f", "t", "body"))
	if err == nil {
		t.Fatal("expected error when root is read-only")
	}
}

func TestNewFSReadOnlyParent(t *testing.T) {
	base := t.TempDir()
	readOnly := filepath.Join(base, "locked")
	if err := os.Mkdir(readOnly, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(readOnly, 0o755) })

	_, err := objstore.NewFS(filepath.Join(readOnly, "sub"))
	if err == nil {
		t.Fatal("expected error when parent is read-only")
	}
}

func TestGetDataMissingMetaPresent(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := objstore.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Put(ctx, "partial", makeObject("x", "x", "data")); err != nil {
		t.Fatal(err)
	}

	// Remove the data file but keep meta.json.
	if err := os.Remove(filepath.Join(dir, "partial", "data")); err != nil {
		t.Fatal(err)
	}

	_, err = s.Get(ctx, "partial")
	if !errors.Is(err, objstore.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPutWriteError(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	obj := objstore.Object{
		Name:        "fail.txt",
		ContentType: "text/plain",
		Size:        0,
		Content:     io.NopCloser(&errReader{err: errors.New("read boom")}),
	}
	err := s.Put(ctx, "failkey", obj)
	if err == nil {
		t.Fatal("expected error from failing reader")
	}
}

type errReader struct{ err error }

func (r *errReader) Read([]byte) (int, error) { return 0, r.err }

func TestPutOverwriteAtomicity(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	orig := makeObject("orig.txt", "text/plain", "original content")
	if err := s.Put(ctx, "atom", orig); err != nil {
		t.Fatal(err)
	}

	// Attempt an overwrite with a reader that fails mid-stream.
	bad := objstore.Object{
		Name:        "bad.txt",
		ContentType: "text/html",
		Content:     io.NopCloser(&errReader{err: errors.New("boom")}),
	}
	if err := s.Put(ctx, "atom", bad); err == nil {
		t.Fatal("expected error from failing reader")
	}

	// The original object must still be intact.
	got, err := s.Get(ctx, "atom")
	if err != nil {
		t.Fatalf("Get after failed overwrite: %v", err)
	}
	body := readObject(t, got)
	if got.Name != "orig.txt" {
		t.Errorf("Name = %q, want %q", got.Name, "orig.txt")
	}
	if got.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, want %q", got.ContentType, "text/plain")
	}
	if body != "original content" {
		t.Errorf("body = %q, want %q", body, "original content")
	}
}

func TestListUnreadableRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "store")
	s, err := objstore.NewFS(root)
	if err != nil {
		t.Fatal(err)
	}

	// Make root non-readable so ReadDir fails with EACCES, not ENOENT.
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(root, 0o755) })

	ctx := context.Background()
	var sawErr bool
	for _, err := range s.List(ctx, "") {
		if err != nil {
			sawErr = true
		}
	}
	if !sawErr {
		t.Fatal("expected error when root is unreadable")
	}
}

func TestPutRenameDataError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := objstore.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Place a non-empty directory at the "data" path so that
	// os.Rename(file, dir) fails with EISDIR / ENOTDIR.
	keyDir := filepath.Join(dir, "rk")
	if err := os.MkdirAll(filepath.Join(keyDir, "data", "blocker"), 0o755); err != nil {
		t.Fatal(err)
	}

	err = s.Put(ctx, "rk", makeObject("f", "t", "body"))
	if err == nil {
		t.Fatal("expected error when data target is a directory")
	}
}

func TestPutRenameMetaError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := objstore.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Place a non-empty directory at the "meta.json" path so that data
	// rename succeeds but the meta rename fails.
	keyDir := filepath.Join(dir, "rk")
	if err := os.MkdirAll(filepath.Join(keyDir, "meta.json", "blocker"), 0o755); err != nil {
		t.Fatal(err)
	}

	err = s.Put(ctx, "rk", makeObject("f", "t", "body"))
	if err == nil {
		t.Fatal("expected error when meta.json target is a directory")
	}
}

func TestListDeletedRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "store")
	s, err := objstore.NewFS(root)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	var count int
	for _, err := range s.List(ctx, "") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 items, got %d", count)
	}
}

func TestPutCreateDataError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := objstore.NewFS(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Put(ctx, "rk", makeObject("f", "t", "body")); err != nil {
		t.Fatal(err)
	}

	// Make the key directory read-only so temp file creation fails.
	keyDir := filepath.Join(dir, "rk")
	if err := os.Chmod(keyDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(keyDir, 0o755) })

	err = s.Put(ctx, "rk", makeObject("f2", "t2", "new"))
	if err == nil {
		t.Fatal("expected error when key dir is read-only")
	}
}
