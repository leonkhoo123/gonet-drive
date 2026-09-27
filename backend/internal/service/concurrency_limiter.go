package service

import (
	"context"

	"go-file-server/internal/logger"
)

// limiter is a counting semaphore that bounds how many external encoder
// processes (ffmpeg) may run at once. Every encoder reserves significant CPU
// and memory, so unbounded concurrency risks exhausting the host (OOM). It
// backs both the thumbnail and the on-the-fly video compression gates, which
// are kept separate so long-lived streams cannot starve thumbnail generation.
type limiter struct {
	ch    chan struct{}
	limit int
	name  string
}

func newLimiter(limit int, name string) *limiter {
	if limit < 1 {
		limit = 1
	}
	return &limiter{
		ch:    make(chan struct{}, limit),
		limit: limit,
		name:  name,
	}
}

// Acquire blocks until a slot is available or the context is cancelled.
// Returns an error if the context is cancelled before acquiring.
func (l *limiter) Acquire(ctx context.Context) error {
	select {
	case l.ch <- struct{}{}:
		logger.L.Debug("encoder semaphore acquired", "name", l.name, "in_use", l.Acquiring(), "limit", l.limit)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire reserves a slot without blocking and reports whether it succeeded.
// Callers that must not queue a long-lived connection (video streams) use this
// and return "busy" immediately when the host is at capacity.
func (l *limiter) TryAcquire() bool {
	select {
	case l.ch <- struct{}{}:
		logger.L.Debug("encoder semaphore acquired", "name", l.name, "in_use", l.Acquiring(), "limit", l.limit)
		return true
	default:
		return false
	}
}

// Release returns a slot to the semaphore. It is safe to call more times than
// Acquire succeeded, so deferred cleanup cannot corrupt the count.
func (l *limiter) Release() {
	select {
	case <-l.ch:
	default:
	}
}

// Available returns the number of slots currently available (for testing).
func (l *limiter) Available() int {
	return l.limit - len(l.ch)
}

// Acquiring returns the number of currently held slots (for testing).
func (l *limiter) Acquiring() int {
	return len(l.ch)
}
