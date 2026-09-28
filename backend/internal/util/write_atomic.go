package util

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path via a hidden temp file in the same
// directory followed by an atomic rename, so a reader never observes a
// partially written file. Parent directories are created as needed and the
// final file gets perm (the temp is chmod'ed before the rename).
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if _, statErr := os.Stat(tmpName); statErr == nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
