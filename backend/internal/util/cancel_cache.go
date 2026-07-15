package util

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var (
	// GlobalCancelCache stores canceled operation IDs
	// We use a short expiration because operations typically don't last forever.
	GlobalCancelCache = cache.New(1*time.Hour, 10*time.Minute)
)

// SetCancelOperation marks an operation as canceled
func SetCancelOperation(opID string) {
	GlobalCancelCache.Set(opID, true, cache.DefaultExpiration)
}

// IsCanceled checks if an operation has been canceled
func IsCanceled(opID string) bool {
	if opID == "" {
		return false
	}
	_, found := GlobalCancelCache.Get(opID)
	return found
}
