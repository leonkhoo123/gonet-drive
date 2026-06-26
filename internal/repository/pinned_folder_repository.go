package repository

import (
	"database/sql"

	"go-file-server/internal/model"
)

type PinnedFolderRepository interface {
	Add(username, path string) error
	Remove(username, path string) (int64, error)
	List(username string) ([]model.PinnedFolder, error)
	Reorder(username string, paths []string) error
	Count(username string) (int, error)
	Exists(username, path string) (bool, error)
}

type SQLitePinnedFolderRepo struct {
	DB *sql.DB
}

func NewSQLitePinnedFolderRepo(db *sql.DB) *SQLitePinnedFolderRepo {
	return &SQLitePinnedFolderRepo{DB: db}
}

func (r *SQLitePinnedFolderRepo) Add(username, path string) error {
	var maxPos sql.NullInt64
	err := r.DB.QueryRow(
		"SELECT MAX(position) FROM pinned_folders WHERE username = ?", username,
	).Scan(&maxPos)
	if err != nil {
		return err
	}
	nextPos := 0
	if maxPos.Valid {
		nextPos = int(maxPos.Int64) + 1
	}
	_, err = r.DB.Exec(
		"INSERT OR IGNORE INTO pinned_folders (username, path, position) VALUES (?, ?, ?)",
		username, path, nextPos,
	)
	return err
}

func (r *SQLitePinnedFolderRepo) Remove(username, path string) (int64, error) {
	res, err := r.DB.Exec(
		"DELETE FROM pinned_folders WHERE username = ? AND path = ?",
		username, path,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *SQLitePinnedFolderRepo) List(username string) ([]model.PinnedFolder, error) {
	rows, err := r.DB.Query(
		"SELECT id, username, path, position, created_at FROM pinned_folders WHERE username = ? ORDER BY position ASC",
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []model.PinnedFolder
	for rows.Next() {
		var f model.PinnedFolder
		if err := rows.Scan(&f.ID, &f.Username, &f.Path, &f.Position, &f.CreatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, nil
}

func (r *SQLitePinnedFolderRepo) Reorder(username string, paths []string) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		"UPDATE pinned_folders SET position = ? WHERE username = ? AND path = ?",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, path := range paths {
		if _, err := stmt.Exec(i, username, path); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SQLitePinnedFolderRepo) Count(username string) (int, error) {
	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM pinned_folders WHERE username = ?", username,
	).Scan(&count)
	return count, err
}

func (r *SQLitePinnedFolderRepo) Exists(username, path string) (bool, error) {
	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM pinned_folders WHERE username = ? AND path = ?", username, path,
	).Scan(&count)
	return count > 0, err
}
