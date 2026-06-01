package repository

import (
	"database/sql"
	"time"
)

type ThumbnailRepository interface {
	MarkAllInactive() error
	Upsert(hash, filePath string, isVideo bool) error
	DeleteInactive() ([]string, error)
}

type SQLiteThumbnailRepo struct {
	DB *sql.DB
}

func NewSQLiteThumbnailRepo(db *sql.DB) *SQLiteThumbnailRepo {
	return &SQLiteThumbnailRepo{DB: db}
}

func (r *SQLiteThumbnailRepo) MarkAllInactive() error {
	_, err := r.DB.Exec(`UPDATE thumbnails SET active = 0, updated_at = ?`, time.Now())
	return err
}

func (r *SQLiteThumbnailRepo) Upsert(hash, filePath string, isVideo bool) error {
	isVideoInt := 0
	if isVideo {
		isVideoInt = 1
	}
	_, err := r.DB.Exec(
		`INSERT INTO thumbnails (hash, file_path, is_video, active, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)
		 ON CONFLICT(hash) DO UPDATE SET
		   file_path = excluded.file_path,
		   is_video = excluded.is_video,
		   active = 1,
		   updated_at = excluded.updated_at`,
		hash, filePath, isVideoInt, time.Now(), time.Now(),
	)
	return err
}

func (r *SQLiteThumbnailRepo) DeleteInactive() ([]string, error) {
	rows, err := r.DB.Query(`SELECT hash FROM thumbnails WHERE active = 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}

	if _, err := r.DB.Exec(`DELETE FROM thumbnails WHERE active = 0`); err != nil {
		return nil, err
	}

	return hashes, nil
}
