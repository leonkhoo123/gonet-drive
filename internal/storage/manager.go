package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

var (
	// usedBytes tracks the current total size of files in the managed directory.
	// It uses an atomic variable for thread-safe operations.
	usedBytes atomic.Int64

	// isScanning indicates if the background scan is currently running.
	isScanning atomic.Bool
)

// InitStorageManager starts a background goroutine to scan the root directory
// and calculate the total size of all files. It sets the result in usedBytes.
func InitStorageManager(rootPath string) {
	if !isScanning.CompareAndSwap(false, true) {
		// Already scanning
		return
	}

	go func() {
		defer isScanning.Store(false)
		startTime := time.Now()
		var totalSize int64

		log.Printf("StorageManager: Starting background scan of %s...", rootPath)

		err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// We might encounter permission errors or files deleted during the scan.
				// Log the error but continue scanning.
				log.Printf("StorageManager: Error accessing path %s: %v", path, err)
				return nil
			}

			if !d.IsDir() {
				info, err := d.Info()
				if err == nil {
					totalSize += info.Size()
				}
			}
			return nil
		})

		if err != nil {
			log.Printf("StorageManager: Background scan failed: %v", err)
			return
		}

		usedBytes.Store(totalSize)
		duration := time.Since(startTime)
		log.Printf("StorageManager: Background scan complete in %v. Total used storage: %d bytes (%.2f GB)",
			duration, totalSize, float64(totalSize)/(1024*1024*1024))
	}()
}

// GetPathSize calculates the total size of files within a given directory or file path.
func GetPathSize(path string) int64 {
	var totalSize int64
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})
	return totalSize
}

// AddUsage adds the specified number of bytes to the current usage.
func AddUsage(bytes int64) {
	if bytes > 0 {
		usedBytes.Add(bytes)
	}
}

// SubtractUsage subtracts the specified number of bytes from the current usage.
func SubtractUsage(bytes int64) {
	if bytes > 0 {
		// Ensure we don't go below 0, though in a perfect system we shouldn't.
		for {
			current := usedBytes.Load()
			newVal := current - bytes
			if newVal < 0 {
				newVal = 0
			}
			if usedBytes.CompareAndSwap(current, newVal) {
				break
			}
		}
	}
}

// GetUsage returns the current used storage in bytes.
func GetUsage() int64 {
	return usedBytes.Load()
}

// CheckLimit checks if adding incomingBytes will exceed the limit.
// If limit is 0 or less, it's considered unlimited.
func CheckLimit(limit int64, incomingBytes int64) bool {
	if limit <= 0 {
		return true // Unlimited
	}
	return GetUsage()+incomingBytes <= limit
}

// HasSufficientStorage is a helper that returns an error if storage is insufficient.
func HasSufficientStorage(limit int64, incomingBytes int64) error {
	if !CheckLimit(limit, incomingBytes) {
		return fmt.Errorf("Cloud storage quota exceeded or disk is full")
	}
	return nil
}
