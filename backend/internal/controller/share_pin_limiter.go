package controller

import (
	"sync"
	"time"
)

const (
	maxPINAttemptsPerShare = 15               // max failed attempts before lockout
	pinLockoutDuration     = 15 * time.Minute // lockout duration after max failures
)

type sharePinAttempt struct {
	count    int
	lockout  time.Time
	lastSeen time.Time
}

// SharePINLimiter tracks failed PIN verification attempts per share ID.
// After maxPINAttemptsPerShare failures, the share ID is locked out for pinLockoutDuration.
type SharePINLimiter struct {
	attempts map[string]*sharePinAttempt
	mu       sync.RWMutex
}

// NewSharePINLimiter creates a new per-share-ID PIN attempt limiter.
func NewSharePINLimiter() *SharePINLimiter {
	limiter := &SharePINLimiter{
		attempts: make(map[string]*sharePinAttempt),
	}
	go limiter.cleanupLoop()
	return limiter
}

// IsLockedOut returns true if the share ID is currently locked out due to too many failed attempts.
func (l *SharePINLimiter) IsLockedOut(shareID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	a, exists := l.attempts[shareID]
	if !exists {
		return false
	}
	if !a.lockout.IsZero() && time.Now().Before(a.lockout) {
		return true
	}
	return false
}

// RecordFailure increments the failure count for a share ID and locks it out if the threshold is reached.
func (l *SharePINLimiter) RecordFailure(shareID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	a, exists := l.attempts[shareID]
	if !exists {
		a = &sharePinAttempt{}
		l.attempts[shareID] = a
	}
	a.count++
	a.lastSeen = time.Now()

	if a.count >= maxPINAttemptsPerShare {
		a.lockout = time.Now().Add(pinLockoutDuration)
	}
}

// Reset clears the failure count for a share ID (called on successful verification).
func (l *SharePINLimiter) Reset(shareID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, shareID)
}

// cleanupLoop periodically removes stale entries to prevent memory leaks.
func (l *SharePINLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for id, a := range l.attempts {
			if now.Sub(a.lastSeen) > 30*time.Minute && (a.lockout.IsZero() || now.After(a.lockout)) {
				delete(l.attempts, id)
			}
		}
		l.mu.Unlock()
	}
}
