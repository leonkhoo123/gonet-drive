package util

import (
	"fmt"
	"os"
	"path/filepath"
)

// RenameFile renames a file at the given full path to a new name.
// The first argument is the full path of the file to rename.
// The second argument is the new name (not a full path, just the filename).
// The file will be renamed in the same directory.
func RenameFile(fullPath string, newName string) error {
	// Check if the source file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("Source file does not exist: %s", fullPath)
	}

	// Get the directory of the original file
	dir := filepath.Dir(filepath.Clean(fullPath))

	// Construct the new full path
	newPath := filepath.Join(dir, filepath.Base(filepath.Clean(newName)))

	// Check if a file with the new name already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("File name already exists: %s", newPath)
	}

	// Rename the file
	if err := os.Rename(fullPath, newPath); err != nil {
		return fmt.Errorf("Failed to rename file: %w", err)
	}

	return nil
}
