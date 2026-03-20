// Package objstore provides an object storage interface with file metadata
// (name, content type, size) and pluggable backends.
//
// The [Store] interface defines the contract for storing and retrieving
// binary objects identified by string keys. The package ships with a
// local-filesystem backend ([NewFS]).
//
// This package follows the same pattern as [kv] and [vecstore]: a generic
// interface with pluggable backends.
package objstore

import (
	"context"
	"errors"
	"io"
	"iter"
)

// Sentinel errors.
var (
	// ErrNotFound is returned when the requested key does not exist.
	ErrNotFound = errors.New("objstore: not found")
)

// Object represents a stored object with its metadata and content stream.
type Object struct {
	// Name is the original file name (e.g. "photo.jpg").
	Name string
	// ContentType is the MIME type (e.g. "image/jpeg").
	ContentType string
	// Size is the content length in bytes.
	Size int64
	// Content is the object body. Callers must close it after use.
	Content io.ReadCloser
}

// ObjectInfo holds object metadata without the content body.
type ObjectInfo struct {
	// Key is the storage key of the object.
	Key string
	// Name is the original file name.
	Name string
	// ContentType is the MIME type.
	ContentType string
	// Size is the content length in bytes.
	Size int64
}

// Store is the interface for object storage with file metadata.
//
// Keys must be non-empty, flat identifiers: no path separators ('/' or '\'),
// no relative-path components ('.' or '..'), and no absolute paths. In other
// words a key is a single filename-like token (e.g. "photo-001", "doc_abc").
//
// All implementations must be safe for concurrent use.
type Store interface {
	// Put stores an object under the given key.
	// If the key already exists, its content and metadata are overwritten.
	// The caller is responsible for closing obj.Content after Put returns.
	Put(ctx context.Context, key string, obj Object) error

	// Get retrieves an object by key. Returns ErrNotFound if not present.
	// The caller must close the returned Object.Content.
	Get(ctx context.Context, key string) (Object, error)

	// Delete removes an object by key. No error if the key does not exist.
	Delete(ctx context.Context, key string) error

	// List iterates over objects whose keys start with the given prefix.
	// The iteration order is lexicographic by key.
	List(ctx context.Context, prefix string) iter.Seq2[ObjectInfo, error]

	// Close releases resources held by the store.
	Close() error
}
