// Package filesystem defines a mutable filesystem interface for persistent storage.
//
// Concrete implementations (local disk, S3, etc.) live in separate packages
// and are wired at the application level.
package filesystem

import (
	"io"
	"io/fs"
)

// FS provides hierarchical filesystem access for persistent storage.
//
// Implementations must be safe for concurrent use.
type FS interface {
	fs.FS

	// ReadDir reads the named directory
	// and returns a list of directory entries sorted by filename.
	ReadDir(name string) ([]fs.DirEntry, error)

	// Stat returns file metadata for the named path.
	// Returns an error wrapping fs.ErrNotExist if the path does not exist.
	Stat(name string) (fs.FileInfo, error)

	// Sub returns an FS corresponding to the subtree rooted at dir.
	Sub(dir string) (fs.FS, error)

	// Glob returns the names of all files matching pattern,
	// providing an implementation of the top-level
	// Glob function.
	Glob(pattern string) ([]string, error)

	// Create creates or truncates a named file for writing.
	// Implementations should ensure atomicity where possible (e.g. write
	// to a temporary file and rename on Close).
	Create(name string) (io.WriteCloser, error)

	// Remove removes a named file. Returns nil if the file does not exist.
	Remove(name string) error

	// MkdirAll creates a directory and any missing parents.
	// Returns nil if the directory already exists.
	MkdirAll(name string) error

	// Rename atomically renames oldName to newName where supported.
	Rename(oldName, newName string) error

	// RemoveAll removes the named path and any children below it.
	// Returns nil if the path does not exist.
	RemoveAll(name string) error
}
