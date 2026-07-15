package ws

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var OpMetaCache = cache.New(12*time.Hour, 30*time.Minute)

func SetOpMeta(opID string, msg OperationMessage) {
	OpMetaCache.Set(opID, msg, cache.DefaultExpiration)
}

func GetOpMeta(opID string) (OperationMessage, bool) {
	if x, found := OpMetaCache.Get(opID); found {
		if msg, ok := x.(OperationMessage); ok {
			return msg, true
		}
	}
	return OperationMessage{}, false
}

func BroadcastAndCache(msg OperationMessage) {
	Broadcast(msg)
	if msg.OpId != "" {
		SetOpMeta(msg.OpId, msg)
	}
}

// BroadcastAndCacheToUser sends a message only to the specified user's WebSocket connections
// and caches it for late-joining clients.
func BroadcastAndCacheToUser(username string, msg OperationMessage) {
	BroadcastToUser(username, msg)
	if msg.OpId != "" {
		SetOpMeta(msg.OpId, msg)
	}
}
