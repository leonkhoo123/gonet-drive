package repository

import (
	"database/sql"
	"go-file-server/internal/model"
)

type AudioPathRepository interface {
	Create(path *model.AudioPath) error
	GetByUsername(username string) ([]model.AudioPath, error)
	GetEnabledPathsByUsername(username string) ([]string, error)
	UpdateEnabled(id int, username string, isEnabled bool) error
	Delete(id int, username string) error
}

type SQLiteAudioPathRepo struct {
	DB *sql.DB
}

func NewSQLiteAudioPathRepo(db *sql.DB) *SQLiteAudioPathRepo {
	return &SQLiteAudioPathRepo{DB: db}
}

func (r *SQLiteAudioPathRepo) Create(path *model.AudioPath) error {
	_, err := r.DB.Exec(
		"INSERT INTO audio_paths (path, username, is_enabled) VALUES (?, ?, ?)",
		path.Path, path.Username, path.IsEnabled,
	)
	return err
}

func (r *SQLiteAudioPathRepo) GetByUsername(username string) ([]model.AudioPath, error) {
	rows, err := r.DB.Query(
		"SELECT id, path, username, is_enabled, created_at, updated_at FROM audio_paths WHERE username = ?",
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []model.AudioPath
	for rows.Next() {
		var p model.AudioPath
		if err := rows.Scan(&p.ID, &p.Path, &p.Username, &p.IsEnabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func (r *SQLiteAudioPathRepo) GetEnabledPathsByUsername(username string) ([]string, error) {
	rows, err := r.DB.Query("SELECT path FROM audio_paths WHERE username = ? AND is_enabled = 1", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func (r *SQLiteAudioPathRepo) UpdateEnabled(id int, username string, isEnabled bool) error {
	_, err := r.DB.Exec(
		"UPDATE audio_paths SET is_enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND username = ?",
		isEnabled, id, username,
	)
	return err
}

func (r *SQLiteAudioPathRepo) Delete(id int, username string) error {
	_, err := r.DB.Exec("DELETE FROM audio_paths WHERE id = ? AND username = ?", id, username)
	return err
}
