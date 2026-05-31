package service

import (
	"context"
	"sync"

	"go-file-server/internal/config"
)

type thumbnailSemaphore struct {
	ch    chan struct{}
	mu    sync.Mutex
	limit int
}

var globalThumbnailSemaphore *thumbnailSemaphore
var semaphoreOnce sync.Once

func getThumbnailSemaphore() *thumbnailSemaphore {
	cfg := config.AppConfig
	if cfg == nil {
		return newThumbnailSemaphore(2)
	}
	return newThumbnailSemaphore(cfg.Server.ThumbnailMaxConcurrent)
}

func newThumbnailSemaphore(limit int) *thumbnailSemaphore {
	if limit < 1 {
		limit = 1
	}
	return &thumbnailSemaphore{
		ch:    make(chan struct{}, limit),
		limit: limit,
	}
}

// Acquire blocks until a slot is available or the context is cancelled.
// Returns an error if the context is cancelled before acquiring.
func (s *thumbnailSemaphore) Acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release returns a slot to the semaphore.
func (s *thumbnailSemaphore) Release() {
	select {
	case <-s.ch:
	default:
	}
}

// Available returns the number of slots currently available (for testing).
func (s *thumbnailSemaphore) Available() int {
	return s.limit - len(s.ch)
}

// Acquiring returns the number of currently held slots (for testing).
func (s *thumbnailSemaphore) Acquiring() int {
	return len(s.ch)
}
