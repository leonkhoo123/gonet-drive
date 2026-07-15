package state

import (
	"time"

	"go-file-server/internal/util"

	"github.com/patrickmn/go-cache"
)

var (
	// GlobalProgressCache stores progress trackers with a default expiration of 1 hour
	// and checks for expired items every 10 minutes
	GlobalProgressCache = cache.New(1*time.Hour, 10*time.Minute)
)

// SetProgress stores a progress tracker in the cache
func SetProgress(id string, pt *util.ProgressTracker) {
	GlobalProgressCache.Set(id, pt, cache.DefaultExpiration)
}

// GetProgress retrieves a progress tracker from the cache
func GetProgress(id string) (*util.ProgressTracker, bool) {
	if x, found := GlobalProgressCache.Get(id); found {
		if pt, ok := x.(*util.ProgressTracker); ok {
			return pt, true
		}
	}
	return nil, false
}
