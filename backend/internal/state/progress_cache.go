package state

import (
	"sync"
	"time"

	"go-file-server/internal/util"

	"github.com/patrickmn/go-cache"
)

var (
	// GlobalProgressCache stores progress trackers with a default expiration of 1 hour
	// and checks for expired items every 10 minutes
	GlobalProgressCache = cache.New(1*time.Hour, 10*time.Minute)

	// operationOwners tracks which username owns each operation ID
	operationOwners   = make(map[string]string)
	operationOwnersMu sync.RWMutex
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

// SetOperationOwner records which username owns an operation ID.
func SetOperationOwner(opID, username string) {
	operationOwnersMu.Lock()
	defer operationOwnersMu.Unlock()
	operationOwners[opID] = username
}

// GetOperationOwner returns the username that owns the operation ID, or "" if unknown.
func GetOperationOwner(opID string) string {
	operationOwnersMu.RLock()
	defer operationOwnersMu.RUnlock()
	return operationOwners[opID]
}

// ClearOperationOwner removes the ownership record for an operation ID.
func ClearOperationOwner(opID string) {
	operationOwnersMu.Lock()
	defer operationOwnersMu.Unlock()
	delete(operationOwners, opID)
}
