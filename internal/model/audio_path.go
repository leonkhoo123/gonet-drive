package model

import "time"

type AudioPath struct {
	ID        int       `json:"id"`
	Path      string    `json:"path"`
	Username  string    `json:"username"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
