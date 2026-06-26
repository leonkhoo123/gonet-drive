package model

import "time"

type PinnedFolder struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Path      string    `json:"path"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}
