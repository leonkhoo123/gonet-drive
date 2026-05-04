package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// CopyFiles copies multiple files and/or folders to a destination directory.
// It preserves the directory structure of folders and handles both files and directories.
// Progress is logged with percentage, speed, and estimated time remaining.
//
// Parameters:
//   - tracker: The progress tracker to use
//   - sources: A list of file/folder paths to copy
//   - destDir: The destination directory where items will be copied
//
// Returns:
//   - error: Any error encountered during the copy operation
//
// Example:
//
//	tracker := NewProgressTracker()
//	sources := []string{"/path/to/file1.txt", "/path/to/folder1", "/path/to/file2.txt"}
//	destDir := "/path/to/destination"
//	err := CopyFiles(tracker, sources, destDir)
//
func CopyFiles(tracker *ProgressTracker, sources []string, destDir string, opID string, isSameDir bool, onSizeCalculated func(int64) error) error {
	destDir = filepath.Clean(destDir)
	// Truncate opID if it's too long
	shortID := opID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	tempDirName := ".file_opt_temp_" + shortID
	tempDir := filepath.Join(destDir, tempDirName)

	// Ensure temp directory exists and is hidden
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup temp dir after operation

	log.Println("Calculating total size...")

	// Calculate total size and file count
	for _, source := range sources {
		if err := calculateSize(source, tracker); err != nil {
			return fmt.Errorf("failed to calculate size for '%s': %w", source, err)
		}
	}

	if onSizeCalculated != nil {
		if err := onSizeCalculated(tracker.TotalBytes); err != nil {
			return err
		}
	}

	log.Printf("Starting copy operation: %s across %d files",
		FormatBytes(tracker.TotalBytes),
		tracker.TotalFiles,
	)

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy each source to the temporary destination
	for _, source := range sources {
		if IsCanceled(opID) {
			return fmt.Errorf("operation canceled")
		}

		// Get the base name of the source
		baseName := filepath.Base(source)
		// Ensure unique name in temp dir to avoid collisions between sources
		tempPath := filepath.Join(tempDir, baseName)

		// Check if source exists
		sourceInfo, err := os.Stat(source)
		if err != nil {
			return fmt.Errorf("failed to stat source '%s': %w", source, err)
		}

		// Copy based on whether it's a file or directory
		if sourceInfo.IsDir() {
			if err := copyDir(source, tempPath, tracker, opID); err != nil {
				return fmt.Errorf("failed to copy directory '%s': %w", source, err)
			}
		} else {
			if err := copyFile(source, tempPath, tracker, opID); err != nil {
				return fmt.Errorf("failed to copy file '%s': %w", source, err)
			}
		}
	}

	// Move all items from tempDir to destDir
	tempEntries, err := os.ReadDir(tempDir)
	if err != nil {
		return fmt.Errorf("failed to read temporary directory: %w", err)
	}

	var copiedPaths []string
	var finalErr error

	defer func() {
		if finalErr != nil {
			log.Printf("Copy failed halfway. Cleaning up successfully copied items in destination...")
			for _, p := range copiedPaths {
				os.RemoveAll(p)
			}
		}
	}()

	for _, entry := range tempEntries {
		itemName := entry.Name()
		tempPath := filepath.Join(tempDir, itemName)
		destPath := filepath.Join(destDir, itemName)

		if isSameDir {
			// Same-directory paste: always use unique name (adds (1) since original exists)
			destPath = GetUniqueDestPath(destPath)
		} else if !entry.IsDir() {
			// Cross-directory file paste: use unique name to avoid overwrite
			destPath = GetUniqueDestPath(destPath)
		}

		if entry.IsDir() {
			if err := moveDirWithMerge(tempPath, destPath, tracker, ""); err != nil {
				finalErr = fmt.Errorf("failed to move directory from temp: %w", err)
				return finalErr
			}
		} else {
			if err := moveFile(tempPath, destPath, tracker); err != nil {
				finalErr = fmt.Errorf("failed to move file from temp: %w", err)
				return finalErr
			}
		}
		copiedPaths = append(copiedPaths, destPath)
	}

	// Log final completion
	tracker.FinalLog()

	return nil
}

// calculateSize recursively calculates the total size and file count
func calculateSize(path string, tracker *ProgressTracker) error {
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
			if err := calculateSize(entryPath, tracker); err != nil {
				return err
			}
		}
	} else {
		tracker.TotalBytes += info.Size()
		tracker.TotalFiles++
	}

	return nil
}

// copyFile copies a single file from src to dst with progress tracking
// dst should already be a unique path (use GetUniqueDestPath if needed)
func copyFile(src, dst string, tracker *ProgressTracker, opID string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Get source file info to preserve permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create destination file
	destFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy the contents with progress tracking
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		if IsCanceled(opID) {
			return fmt.Errorf("operation canceled")
		}
		n, err := sourceFile.Read(buf)
		if n > 0 {
			if _, writeErr := destFile.Write(buf[:n]); writeErr != nil {
				if strings.Contains(writeErr.Error(), "no space left on device") {
					return fmt.Errorf("Cloud storage quota exceeded or disk is full")
				}
				return fmt.Errorf("failed to write to destination file: %w", writeErr)
			}
			tracker.Update(int64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read from source file: %w", err)
		}
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	tracker.FileCompleted()

	return nil
}

// copyDir recursively copies a directory from src to dst with progress tracking
// Merges with existing destination directory if it exists
func copyDir(src, dst string, tracker *ProgressTracker, opID string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	// Get source directory info
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}

	// Create or merge with destination directory
	if err := os.MkdirAll(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read all entries in the source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		if IsCanceled(opID) {
			return fmt.Errorf("operation canceled")
		}

		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory (will merge if exists)
			if err := copyDir(sourcePath, destPath, tracker, opID); err != nil {
				return err
			}
		} else {
			// Copy file with auto-rename if conflict
			destPath = GetUniqueDestPath(destPath)
			if err := copyFile(sourcePath, destPath, tracker, opID); err != nil {
				return err
			}
		}
	}

	return nil
}
