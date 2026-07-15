package model

import "time"

type AudiobookProgress struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	AudioBookName string    `json:"audiobook_name"`
	ProgressTime  float64   `json:"progress_time"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
