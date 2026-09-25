package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go-file-server/internal/logger"
	"go-file-server/internal/storage"
)

// uploadCompletionFile is the marker written into an upload's temp directory
// once its chunks have been merged into the final file. It makes the final
// chunk idempotent: if the merge succeeded but the response was lost, replaying
// the request returns the same result instead of failing with "Missing chunk N".
const uploadCompletionFile = ".completed.json"

// uploadCompletion records a finished merge so it can be replayed.
type uploadCompletion struct {
	VirtualPath  string    `json:"virtual_path"`  // path returned to the client
	PhysicalPath string    `json:"physical_path"` // path on disk, for existence checks
	Size         int64     `json:"size"`
	CompletedAt  time.Time `json:"completed_at"`
}

// uploadTempDir returns the per-upload scratch directory for an identifier.
func uploadTempDir(root, identifier string) string {
	return filepath.Join(root, ".cloud_reserve", "upload_temp", identifier)
}

func uploadCompletionPath(tempDir string) string {
	return filepath.Join(tempDir, uploadCompletionFile)
}

func writeUploadCompletion(tempDir string, completion uploadCompletion) error {
	data, err := json.Marshal(completion)
	if err != nil {
		return err
	}
	marker := uploadCompletionPath(tempDir)
	tmp := marker + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, marker)
}

func readUploadCompletion(tempDir string) (uploadCompletion, bool) {
	data, err := os.ReadFile(uploadCompletionPath(tempDir))
	if err != nil {
		return uploadCompletion{}, false
	}
	var completion uploadCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return uploadCompletion{}, false
	}
	if completion.PhysicalPath == "" {
		return uploadCompletion{}, false
	}
	return completion, true
}

// completedUpload returns the recorded completion for an identifier when the
// upload already finished and the produced file is still present on disk.
func completedUpload(root, identifier string) (uploadCompletion, bool) {
	completion, ok := readUploadCompletion(uploadTempDir(root, identifier))
	if !ok {
		return uploadCompletion{}, false
	}
	if _, err := os.Stat(completion.PhysicalPath); err != nil {
		return uploadCompletion{}, false
	}
	return completion, true
}

// clearUploadTemp removes an upload's temp directory and adjusts storage usage.
func clearUploadTemp(tempDir string) {
	size := storage.GetPathSize(tempDir)
	_ = os.RemoveAll(tempDir)
	storage.SubtractUsage(size)
}

// --- keyed locks (one per upload identifier) ---

type uploadLock struct {
	mu   sync.Mutex
	refs int
}

var (
	uploadLocksMu sync.Mutex
	uploadLocks   = map[string]*uploadLock{}
)

// lockUpload serializes requests for a single upload identifier and returns the
// unlock function. Entries are reference-counted so the map does not grow
// without bound as identifiers come and go.
func lockUpload(identifier string) func() {
	uploadLocksMu.Lock()
	lock := uploadLocks[identifier]
	if lock == nil {
		lock = &uploadLock{}
		uploadLocks[identifier] = lock
	}
	lock.refs++
	uploadLocksMu.Unlock()

	lock.mu.Lock()

	return func() {
		lock.mu.Unlock()
		uploadLocksMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(uploadLocks, identifier)
		}
		uploadLocksMu.Unlock()
	}
}

// CleanupStaleUploadTemp removes upload temp directories whose modification
// time is older than ttl. This reaps both abandoned in-progress uploads (the
// residue left behind when an upload fails) and completion markers left by
// idempotent replays. It returns the number of directories removed.
func CleanupStaleUploadTemp(root string, ttl time.Duration) (removed int) {
	base := filepath.Join(root, ".cloud_reserve", "upload_temp")
	entries, err := os.ReadDir(base)
	if err != nil {
		return 0
	}

	cutoff := time.Now().Add(-ttl)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		clearUploadTemp(filepath.Join(base, entry.Name()))
		removed++
	}

	if removed > 0 {
		logger.L.Info("upload temp cleanup complete", "removed", removed)
	}
	return removed
}
