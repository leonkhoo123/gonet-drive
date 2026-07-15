package model

import "time"

type VideoIntegrity struct {
	Hash            string    `json:"hash"`
	FilePath        string    `json:"file_path"`
	IssueType       string    `json:"issue_type"`
	MimeCodecString string    `json:"mime_codec_string"`
	DetectedAt      time.Time `json:"detected_at"`
	LastCheckedAt   time.Time `json:"last_checked_at"`
}
