package controller

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// MemoryRateLimiterStore implements gonetauth.RateLimiterStore using in-memory token bucket.
type MemoryRateLimiterStore struct {
	ips      map[string]*rate.Limiter
	lastSeen map[string]time.Time
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
	maxIdle  time.Duration
}

// NewMemoryRateLimiterStore creates a new in-memory rate limiter store.
func NewMemoryRateLimiterStore(rps rate.Limit, burst int) *MemoryRateLimiterStore {
	return &MemoryRateLimiterStore{
		ips:      make(map[string]*rate.Limiter),
		lastSeen: make(map[string]time.Time),
		rps:      rps,
		burst:    burst,
		maxIdle:  10 * time.Minute,
	}
}

// Allow checks if a request from the given IP is allowed.
func (s *MemoryRateLimiterStore) Allow(ip string) bool {
	// Fast path: read-lock to check if limiter exists
	s.mu.RLock()
	limiter, exists := s.ips[ip]
	s.mu.RUnlock()

	if exists {
		s.mu.Lock()
		s.lastSeen[ip] = time.Now()
		allowed := limiter.Allow()
		s.mu.Unlock()
		return allowed
	}

	// Slow path: write-lock to create new limiter
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	limiter, exists = s.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(s.rps, s.burst)
		s.ips[ip] = limiter
	}
	s.lastSeen[ip] = time.Now()
	return limiter.Allow()
}

// Cleanup removes stale entries older than maxIdle.
func (s *MemoryRateLimiterStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for ip, lastSeen := range s.lastSeen {
		if time.Since(lastSeen) > s.maxIdle {
			delete(s.ips, ip)
			delete(s.lastSeen, ip)
		}
	}
}

// StartCleanupScheduler starts a background goroutine that periodically cleans up stale entries.
func (s *MemoryRateLimiterStore) StartCleanupScheduler(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.Cleanup()
		}
	}()
}
