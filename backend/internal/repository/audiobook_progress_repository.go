package repository

import (
	"database/sql"
	"go-file-server/internal/model"
)

type AudiobookProgressRepository interface {
	Upsert(progress *model.AudiobookProgress) error
	GetByUsername(username string) (map[string]float64, error)
}

type SQLiteAudiobookProgressRepo struct {
	DB *sql.DB
}

func NewSQLiteAudiobookProgressRepo(db *sql.DB) *SQLiteAudiobookProgressRepo {
	return &SQLiteAudiobookProgressRepo{DB: db}
}

func (r *SQLiteAudiobookProgressRepo) Upsert(progress *model.AudiobookProgress) error {
	_, err := r.DB.Exec(`
		INSERT INTO audiobook_progress (username, audiobook_name, progress_time)
		VALUES (?, ?, ?)
		ON CONFLICT(username, audiobook_name) 
		DO UPDATE SET progress_time = excluded.progress_time, updated_at = CURRENT_TIMESTAMP
	`, progress.Username, progress.AudioBookName, progress.ProgressTime)
	return err
}

func (r *SQLiteAudiobookProgressRepo) GetByUsername(username string) (map[string]float64, error) {
	rows, err := r.DB.Query("SELECT audiobook_name, progress_time FROM audiobook_progress WHERE username = ?", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	progressMap := make(map[string]float64)
	for rows.Next() {
		var name string
		var time float64
		if err := rows.Scan(&name, &time); err != nil {
			return nil, err
		}
		progressMap[name] = time
	}
	return progressMap, nil
}
