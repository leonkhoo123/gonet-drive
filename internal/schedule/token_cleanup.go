package schedule

import (
	"time"

	"go-file-server/internal/logger"
	"go-file-server/internal/repository"
)

func untilNext(t time.Time, hour, minute int) time.Duration {
	next := time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, t.Location())
	if t.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(t)
}

func StartCleanupScheduler(tokenRepo repository.RefreshTokenRepository) {
	go func() {
		for {
			delay := untilNext(time.Now(), 3, 0)
			logger.L.Debug("token cleanup scheduled", "delay", delay.Round(time.Second))
			time.Sleep(delay)

			deleted, err := tokenRepo.DeleteExpired()
			if err != nil {
				logger.L.Error("token cleanup failed", "err", err)
			} else if deleted > 0 {
				logger.L.Info("token cleanup completed", "removed", deleted)
			}
		}
	}()
}
