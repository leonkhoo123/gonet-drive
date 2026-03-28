package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ResolveDuplicatePath(dir, filename string) string {
	dest := filepath.Join(dir, filename)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}
	log.Printf("File name [%s] duplicate at %s", filename, filepath.Dir(dest))
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s(%d)%s", name, i, ext)
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			log.Printf("Rename file to  %s", filepath.Base(newPath))
			return newPath
		}
	}
}

// getUniqueDestPath generates a unique destination path by adding (1), (2), etc. if file exists
func GetUniqueDestPath(destPath string) string {
	// If destination doesn't exist, return as is
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return destPath
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
			return newPath
		}
	}
}
