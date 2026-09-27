package service

import (
	"sync"

	"go-file-server/internal/config"
	"go-file-server/internal/logger"
)

// thumbnailSemaphore is a limiter dedicated to thumbnail generation. It remains
// a named type so call sites and tests read clearly, while sharing the generic
// slot accounting in concurrency_limiter.go.
type thumbnailSemaphore = limiter

var globalThumbnailSemaphore *thumbnailSemaphore
var semaphoreOnce sync.Once

func GetThumbnailSemaphore() *thumbnailSemaphore {
	semaphoreOnce.Do(func() {
		cfg := config.AppConfig
		if cfg == nil {
			globalThumbnailSemaphore = newThumbnailSemaphore(1)
		} else {
			globalThumbnailSemaphore = newThumbnailSemaphore(cfg.Server.ThumbnailMaxConcurrent)
		}
		logger.L.Debug("thumbnail semaphore initialized", "limit", globalThumbnailSemaphore.limit)
	})
	return globalThumbnailSemaphore
}

func newThumbnailSemaphore(limit int) *thumbnailSemaphore {
	return newLimiter(limit, "thumbnail")
}
