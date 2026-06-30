package depotstore

import "io/fs"

// Store provides the file operations needed by depot-based firmware storage.
//
// Paths are relative slash-separated names under the store root.
type Store interface {
	Open(name string) (fs.File, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte) error
	Stat(name string) (fs.FileInfo, error)
	MkdirAll(name string) error
	Rename(oldName, newName string) error
	RemoveAll(name string) error
}
