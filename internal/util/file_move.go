package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// MoveFiles moves multiple files and/or folders to a destination directory.
// It uses os.Rename for efficient moving within the same filesystem.
// Progress is logged with percentage and file count.
//
// Parameters:
//   - tracker: The progress tracker to use
//   - sources: A list of file/folder paths to move
//   - destDir: The destination directory where items will be moved
//
// Returns:
//   - error: Any error encountered during the move operation
//
// Example:
//
//	tracker := NewProgressTracker()
//	sources := []string{"/path/to/file1.txt", "/path/to/folder1", "/path/to/file2.txt"}
//	destDir := "/path/to/destination"
//	err := MoveFiles(tracker, sources, destDir)
//
func MoveFiles(tracker *ProgressTracker, sources []string, destDir string, opID string) error {
	destDir = filepath.Clean(destDir)
	// Tracker is passed in now

	log.Println("Counting files to move...")

	// Count total files
	for _, source := range sources {
		if err := countFiles(source, tracker); err != nil {
			return fmt.Errorf("failed to count files for '%s': %w", source, err)
		}
	}

	log.Printf("Starting move operation: %d files to move", tracker.TotalFiles)

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	var movedItems []struct{ src, dst string }
	var finalErr error

	defer func() {
		if finalErr != nil {
			log.Printf("Move failed halfway. Reverting %d successfully moved items...", len(movedItems))
			for i := len(movedItems) - 1; i >= 0; i-- {
				item := movedItems[i]
				// We attempt to rename it back. If it's a merged directory, this may only partially work,
				// but for most items, this will revert the move properly.
				if err := os.Rename(item.dst, item.src); err != nil {
					log.Printf("Failed to revert move from %s to %s: %v", item.dst, item.src, err)
				}
			}
		}
	}()

	// Move each source to the destination
	for _, source := range sources {
		if IsCanceled(opID) {
			finalErr = fmt.Errorf("operation canceled")
			return finalErr
		}

		// Detect if this is a same-directory move
		// We use filepath.Clean to ensure consistent path comparison
		if filepath.Dir(source) == filepath.Clean(destDir) {
			finalErr = fmt.Errorf("cannot move '%s' to the same directory", filepath.Base(source))
			return finalErr
		}

		// Get the base name of the source
		baseName := filepath.Base(source)
		destPath := filepath.Join(destDir, baseName)

		// Check if source exists
		sourceInfo, err := os.Stat(source)
		if err != nil {
			finalErr = fmt.Errorf("failed to stat source '%s': %w", source, err)
			return finalErr
		}

		// Move based on whether it's a file or directory
		if sourceInfo.IsDir() {
			if err := moveDirWithMerge(source, destPath, tracker, opID); err != nil {
				finalErr = fmt.Errorf("failed to move directory '%s': %w", source, err)
				return finalErr
			}
		} else {
			// For files, auto-rename if conflict exists (always, as per requirements)
			destPath = GetUniqueDestPath(destPath)
			if err := moveFile(source, destPath, tracker); err != nil {
				finalErr = fmt.Errorf("failed to move file '%s': %w", source, err)
				return finalErr
			}
		}

		movedItems = append(movedItems, struct{ src, dst string }{source, destPath})
	}

	// Log final completion
	tracker.FinalLog()

	return nil
}

// countFiles recursively counts the total number of files
func countFiles(path string, tracker *ProgressTracker) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())
			if err := countFiles(entryPath, tracker); err != nil {
				return err
			}
		}
	} else {
		tracker.TotalFiles++
	}

	return nil
}

// moveFile moves a single file from src to dst
func moveFile(src, dst string, tracker *ProgressTracker) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	// Perform the actual move using os.Rename
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to rename '%s' to '%s': %w", src, dst, err)
	}

	// Mark file as completed
	tracker.FileCompleted()

	return nil
}

// moveDirWithMerge moves a directory, merging with destination if it exists
func moveDirWithMerge(src, dst string, tracker *ProgressTracker, opID string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	if IsCanceled(opID) {
		return fmt.Errorf("operation canceled")
	}

	// Check if destination directory exists
	destInfo, err := os.Stat(dst)
	if err == nil && destInfo.IsDir() {
		// Destination exists and is a directory - merge contents
		return mergeMoveDir(src, dst, tracker, opID)
	} else if err == nil {
		// Destination exists but is a file - this is a conflict
		// Auto-rename the destination path
		dst = GetUniqueDestPath(dst)
	}

	// Destination doesn't exist or we got a unique name - try simple rename first
	if err := os.Rename(src, dst); err == nil {
		// Success! Mark all files as moved
		return markFilesAsMoved(dst, tracker)
	}

	// Rename failed (possibly cross-device), fall back to merge move
	return mergeMoveDir(src, dst, tracker, opID)
}

// mergeMoveDir recursively moves directory contents, merging with existing destination
func mergeMoveDir(src, dst string, tracker *ProgressTracker, opID string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	// Ensure destination directory exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read all entries in the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Move each entry
	for _, entry := range entries {
		if IsCanceled(opID) {
			return fmt.Errorf("operation canceled")
		}

		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively move subdirectory with merge
			if err := moveDirWithMerge(sourcePath, destPath, tracker, opID); err != nil {
				return err
			}
		} else {
			// Move file with auto-rename if conflict
			destPath = GetUniqueDestPath(destPath)
			if err := moveFile(sourcePath, destPath, tracker); err != nil {
				return err
			}
		}
	}

	// Remove the now-empty source directory
	if err := os.Remove(src); err != nil {
		// Log but don't fail if we can't remove the source directory
		log.Printf("Warning: failed to remove source directory '%s': %v", src, err)
	}

	return nil
}

// markFilesAsMoved recursively marks all files in a directory as moved
func markFilesAsMoved(path string, tracker *ProgressTracker) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			// Recursively mark files in subdirectory
			if err := markFilesAsMoved(entryPath, tracker); err != nil {
				return err
			}
		} else {
			// Mark file as completed
			tracker.FileCompleted()
		}
	}

	return nil
}
