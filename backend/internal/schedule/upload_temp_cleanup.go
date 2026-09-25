package schedule

import (
	"time"

	"go-file-server/internal/service"
)

// uploadTempTTL is how long an upload's scratch directory is kept before the
// sweep removes it. It must exceed the longest realistic upload so in-progress
// chunks are never reaped mid-transfer.
const uploadTempTTL = 24 * time.Hour

// StartUploadTempCleanup periodically removes stale upload temp directories:
// residue left behind by failed uploads and completion markers kept for
// idempotent replays. It runs once at startup, then hourly.
func StartUploadTempCleanup(rootPath string) {
	go func() {
		service.CleanupStaleUploadTemp(rootPath, uploadTempTTL)

		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			service.CleanupStaleUploadTemp(rootPath, uploadTempTTL)
		}
	}()
}
