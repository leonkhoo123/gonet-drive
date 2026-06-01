package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"go-file-server/internal/logger"
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

		logger.L.Info("storage scan started", "path", rootPath)

		err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				logger.L.Warn("storage scan path access error", "path", path, "err", err)
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
			logger.L.Error("storage scan failed", "err", err)
			return
		}

		usedBytes.Store(totalSize)
		duration := time.Since(startTime)
		logger.L.Info("storage scan complete", "duration", duration.String(), "total_bytes", totalSize)
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
