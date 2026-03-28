package model

import "time"

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
