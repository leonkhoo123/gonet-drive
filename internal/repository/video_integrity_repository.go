package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type VideoIntegrityRepository interface {
	DeleteAll() error
	Upsert(hash, filePath, issueType, mimeCodec string) error
	GetCorruptHashes(hashes []string) (map[string]bool, error)
	Count() (int, error)
	All() ([]VideoIntegrityEntry, error)
	LastScanTime() (*time.Time, error)
}

type VideoIntegrityEntry struct {
	Hash            string    `json:"hash"`
	FilePath        string    `json:"file_path"`
	IssueType       string    `json:"issue_type"`
	MimeCodecString string    `json:"mime_codec_string"`
	DetectedAt      time.Time `json:"detected_at"`
	LastCheckedAt   time.Time `json:"last_checked_at"`
}

type SQLiteVideoIntegrityRepo struct {
	DB *sql.DB
}

func NewSQLiteVideoIntegrityRepo(db *sql.DB) *SQLiteVideoIntegrityRepo {
	return &SQLiteVideoIntegrityRepo{DB: db}
}

func (r *SQLiteVideoIntegrityRepo) DeleteAll() error {
	_, err := r.DB.Exec(`DELETE FROM video_integrity`)
	return err
}

func (r *SQLiteVideoIntegrityRepo) Upsert(hash, filePath, issueType, mimeCodec string) error {
	now := time.Now()
	_, err := r.DB.Exec(
		`INSERT INTO video_integrity (hash, file_path, issue_type, mime_codec_string, detected_at, last_checked_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(hash) DO UPDATE SET
		   file_path = excluded.file_path,
		   issue_type = excluded.issue_type,
		   mime_codec_string = excluded.mime_codec_string,
		   last_checked_at = excluded.last_checked_at`,
		hash, filePath, issueType, mimeCodec, now, now,
	)
	return err
}

func (r *SQLiteVideoIntegrityRepo) GetCorruptHashes(hashes []string) (map[string]bool, error) {
	if len(hashes) == 0 {
		return map[string]bool{}, nil
	}

	placeholders := make([]string, len(hashes))
	args := make([]interface{}, len(hashes))
	for i, h := range hashes {
		placeholders[i] = "?"
		args[i] = h
	}

	query := fmt.Sprintf(`SELECT hash FROM video_integrity WHERE hash IN (%s)`, strings.Join(placeholders, ","))
	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	corruptSet := make(map[string]bool, len(hashes))
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		corruptSet[h] = true
	}
	return corruptSet, rows.Err()
}

func (r *SQLiteVideoIntegrityRepo) Count() (int, error) {
	var count int
	err := r.DB.QueryRow(`SELECT COUNT(*) FROM video_integrity`).Scan(&count)
	return count, err
}

func (r *SQLiteVideoIntegrityRepo) All() ([]VideoIntegrityEntry, error) {
	rows, err := r.DB.Query(`SELECT hash, file_path, issue_type, mime_codec_string, detected_at, last_checked_at FROM video_integrity`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []VideoIntegrityEntry
	for rows.Next() {
		var e VideoIntegrityEntry
		if err := rows.Scan(&e.Hash, &e.FilePath, &e.IssueType, &e.MimeCodecString, &e.DetectedAt, &e.LastCheckedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *SQLiteVideoIntegrityRepo) LastScanTime() (*time.Time, error) {
	var ts sql.NullInt64
	err := r.DB.QueryRow(`SELECT CAST(strftime('%s', MAX(last_checked_at)) AS INTEGER) FROM video_integrity`).Scan(&ts)
	if err != nil {
		return nil, err
	}
	if !ts.Valid {
		return nil, nil
	}
	t := time.Unix(ts.Int64, 0)
	return &t, nil
}
