package state

import (
	"sync/atomic"
	"time"
)

var lastAPIRequest atomic.Int64

func RecordAPIRequest() {
	lastAPIRequest.Store(time.Now().UnixMilli())
}

func IsIdle(threshold time.Duration) bool {
	last := lastAPIRequest.Load()
	if last == 0 {
		return true
	}
	return time.Since(time.UnixMilli(last)) > threshold
}

func ResetLastAPIRequest() {
	lastAPIRequest.Store(0)
}
