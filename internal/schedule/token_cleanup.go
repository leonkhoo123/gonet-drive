package schedule

import (
	"log"
	"time"

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
			log.Printf("Token cleanup scheduled in %s (next run at 03:00)", delay.Round(time.Second))
			time.Sleep(delay)

			deleted, err := tokenRepo.DeleteExpired()
			if err != nil {
				log.Printf("Token cleanup failed: %v", err)
			} else if deleted > 0 {
				log.Printf("Token cleanup: removed %d expired/revoked rows", deleted)
			}
		}
	}()
}
