package util

import (
	"fmt"
	"os"
	"path/filepath"

	"go-file-server/internal/logger"
)

// DeleteFilesPermanent permanently deletes files and folders from the filesystem.
// Progress is tracked by file count.
func DeleteFilesPermanent(tracker *ProgressTracker, sources []string, opID string, onDeleted func(int64)) error {
	logger.L.Debug("counting files to permanently delete")

	// Count total files
	for _, source := range sources {
		if err := countFiles(source, tracker); err != nil {
			logger.L.Warn("failed to count files for deletion", "path", source, "err", err)
		}
	}

	logger.L.Info("starting permanent delete operation", "files", tracker.TotalFiles)

	var finalErr error

	for _, source := range sources {
		if IsCanceled(opID) {
			finalErr = fmt.Errorf("operation canceled")
			return finalErr
		}
		if err := removePath(source, tracker, opID, onDeleted); err != nil {
			finalErr = fmt.Errorf("failed to permanently delete '%s': %w", source, err)
			logger.L.Error("failed to permanently delete", "path", source, "err", err)
			// Continue attempting to delete other files even if one fails
		}
	}

	tracker.FinalLog()
	return finalErr
}

// removePath recursively deletes a file or directory and updates the tracker.
func removePath(path string, tracker *ProgressTracker, opID string, onDeleted func(int64)) error {
	if IsCanceled(opID) {
		return fmt.Errorf("operation canceled")
	}

	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if IsCanceled(opID) {
				return fmt.Errorf("operation canceled")
			}
			entryPath := filepath.Join(path, entry.Name())
			if err := removePath(entryPath, tracker, opID, onDeleted); err != nil {
				return err
			}
		}

		// Remove the empty directory
		if err := os.Remove(path); err != nil {
			return err
		}
	} else {
		size := info.Size()
		// Remove the file or symlink
		if err := os.Remove(path); err != nil {
			return err
		}
		if onDeleted != nil {
			onDeleted(size)
		}
		tracker.FileCompleted()
	}

	return nil
}
