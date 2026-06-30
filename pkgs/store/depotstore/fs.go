package depotstore

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Dir stores depots on the local filesystem rooted at the given directory.
type Dir string

var _ Store = Dir("")

func (d Dir) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(d.abs(name))
}

func (d Dir) Open(name string) (fs.File, error) {
	return os.Open(d.abs(name))
}

func (d Dir) WriteFile(name string, data []byte) error {
	full := d.abs(name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}

func (d Dir) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(d.abs(name))
}

func (d Dir) MkdirAll(name string) error {
	return os.MkdirAll(d.abs(name), 0o755)
}

func (d Dir) Rename(oldName, newName string) error {
	oldPath := d.abs(oldName)
	newPath := d.abs(newName)
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

func (d Dir) RemoveAll(name string) error {
	return os.RemoveAll(d.abs(name))
}

func (d Dir) abs(name string) string {
	if name == "." || name == "" {
		return string(d)
	}
	return filepath.Join(string(d), filepath.FromSlash(strings.TrimPrefix(name, "./")))
}
