package model

import "time"

type RefreshToken struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	TokenHash  string    `json:"-"`
	FamilyID   string    `json:"family_id"`
	DeviceID   string    `json:"device_id"`
	DeviceInfo string    `json:"device_info"`
	IPAddress  string    `json:"ip_address"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsRevoked  bool      `json:"is_revoked"`
	CreatedAt  time.Time `json:"created_at"`
}
