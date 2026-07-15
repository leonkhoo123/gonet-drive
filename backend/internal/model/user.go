package model

import "time"

type User struct {
	ID             string     `json:"id"`
	Username       string     `json:"username"`
	PasswordHash   string     `json:"-"`
	Role           string     `json:"role"`
	MFASecret      *string    `json:"-"`
	MFAEnabled     bool       `json:"mfa_enabled"`
	MFAMandatory   bool       `json:"mfa_mandatory"`
	StorageQuota   *int64     `json:"storage_quota"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	FailedAttempts int        `json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until"`
	TokenVersion   int        `json:"token_version"`
}
