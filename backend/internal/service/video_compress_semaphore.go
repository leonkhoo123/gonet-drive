package service

import (
	"sync"

	"go-file-server/internal/config"
	"go-file-server/internal/logger"
)

// videoCompressSemaphore bounds the number of concurrent on-the-fly transcode
// streams. It is deliberately separate from the thumbnail semaphore: a stream
// is long-lived, so a paused player holding a slot for minutes must not starve
// thumbnail generation.
type videoCompressSemaphore = limiter

var globalVideoCompressSemaphore *videoCompressSemaphore
var videoCompressOnce sync.Once

// GetVideoCompressSemaphore returns the process-wide transcode limiter,
// initialised from VIDEO_COMPRESS_MAX_CONCURRENT (default 2, minimum 1).
func GetVideoCompressSemaphore() *videoCompressSemaphore {
	videoCompressOnce.Do(func() {
		limit := 1
		if cfg := config.AppConfig; cfg != nil {
			limit = cfg.Server.VideoCompressMaxConcurrent
		}
		globalVideoCompressSemaphore = newLimiter(limit, "video-compress")
		logger.L.Debug("video compress semaphore initialized", "limit", globalVideoCompressSemaphore.limit)
	})
	return globalVideoCompressSemaphore
}
