package state

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// ShareCacheEntry holds the cached status of a share link.
type ShareCacheEntry struct {
	Exists  bool
	Blocked bool
}

var (
	// ShareStatusCache caches the DB existence/blocked status of share links.
	// Default 5-minute TTL to limit DB queries. Manually invalidated on
	// block/unblock/delete operations for immediate effect.
	ShareStatusCache = cache.New(5*time.Minute, 10*time.Minute)
)

// GetShareStatus returns (exists, blocked, found) from the cache.
func GetShareStatus(shareID string) (exists, blocked bool, found bool) {
	val, found := ShareStatusCache.Get(shareID)
	if !found {
		return false, false, false
	}
	entry, ok := val.(*ShareCacheEntry)
	if !ok {
		return false, false, false
	}
	return entry.Exists, entry.Blocked, true
}

// SetShareStatus stores the share status in cache with default TTL.
func SetShareStatus(shareID string, exists, blocked bool) {
	ShareStatusCache.Set(shareID, &ShareCacheEntry{
		Exists:  exists,
		Blocked: blocked,
	}, cache.DefaultExpiration)
}

// InvalidateShareStatus removes a share from the cache.
// Call after DB block/unblock/delete to force next middleware check to hit DB.
func InvalidateShareStatus(shareID string) {
	ShareStatusCache.Delete(shareID)
}
