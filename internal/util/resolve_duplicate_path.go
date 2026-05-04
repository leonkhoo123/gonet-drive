package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ResolveDuplicatePath adds a numeric suffix to a filename if it already exists in the given directory.
func ResolveDuplicatePath(dir, filename string) (string, error) {
	if strings.Contains(dir, "..") || strings.Contains(filename, "..") {
		return "", fmt.Errorf("invalid path")
	}
	dest := filepath.Join(dir, filepath.Base(filepath.Clean(filename)))
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest, nil
	}
	log.Printf("File name [%s] duplicate at %s", filename, filepath.Dir(dest))
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filepath.Base(filename), ext)
	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s(%d)%s", name, i, ext)
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			log.Printf("Rename file to  %s", filepath.Base(newPath))
			return newPath, nil
		}
	}
}

// GetUniqueDestPath generates a unique destination path by adding (1), (2), etc. if file exists
func GetUniqueDestPath(destPath string) (string, error) {
	if strings.Contains(destPath, "..") {
		return "", fmt.Errorf("invalid path")
	}
	destPath = filepath.Clean(destPath)
	// If destination doesn't exist, return as is
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return destPath, nil
	}

	// Extract directory, filename, and extension
	dir := filepath.Dir(destPath)
	base := filepath.Base(destPath)
	ext := filepath.Ext(base)
	nameWithoutExt := base[:len(base)-len(ext)]

	// Try adding (1), (2), (3), etc. until we find a unique name
	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s (%d)%s", nameWithoutExt, i, ext)
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath, nil
		}
	}
}
