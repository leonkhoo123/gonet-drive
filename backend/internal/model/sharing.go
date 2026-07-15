package model

import "time"

// NeverExpires is a sentinel value representing a share that never expires.
var NeverExpires = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

type SharingInfo struct {
	ID          string    `json:"id"`
	Path        string    `json:"path"`
	PinHash     string    `json:"-"` // Don't expose hash in JSON
	ExpiresAt   time.Time `json:"expires_at"`
	Blocked     bool      `json:"blocked"`
	Authority   string    `json:"authority"` // 'view' or 'modify'
	Username    string    `json:"username"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	IsDir       bool      `json:"is_dir"` // Computed property, not in DB
}

// IsNeverExpires returns true if this share is set to never expire.
func (s *SharingInfo) IsNeverExpires() bool {
	return s.ExpiresAt.Equal(NeverExpires)
}
