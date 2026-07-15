package model

import "time"

type Thumbnail struct {
	ID        int64     `json:"id"`
	Hash      string    `json:"hash"`
	FilePath  string    `json:"file_path"`
	IsVideo   bool      `json:"is_video"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
