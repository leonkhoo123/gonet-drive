package repository

import (
	"database/sql"
	"go-file-server/internal/model"
)

type SharingRepository interface {
	Create(share *model.SharingInfo) error
	GetByID(id string) (*model.SharingInfo, error)
	ListByUsername(username string) ([]model.SharingInfo, error)
	UpdateBlockedStatus(id string, username string, blocked bool) error
	Delete(id string, username string) (int64, error)
}

type SQLiteSharingRepo struct {
	DB *sql.DB
}

func NewSQLiteSharingRepo(db *sql.DB) *SQLiteSharingRepo {
	return &SQLiteSharingRepo{DB: db}
}

func (r *SQLiteSharingRepo) Create(share *model.SharingInfo) error {
	_, err := r.DB.Exec(
		`INSERT INTO sharing_info (id, path, pin_hash, expires_at, blocked, authority, username, description, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		share.ID, share.Path, share.PinHash, share.ExpiresAt, share.Blocked, share.Authority, share.Username, share.Description, share.CreatedAt,
	)
	return err
}

func (r *SQLiteSharingRepo) GetByID(id string) (*model.SharingInfo, error) {
	var s model.SharingInfo
	err := r.DB.QueryRow(
		`SELECT id, path, pin_hash, expires_at, blocked, authority, username, description, created_at 
		 FROM sharing_info WHERE id = ?`, id,
	).Scan(&s.ID, &s.Path, &s.PinHash, &s.ExpiresAt, &s.Blocked, &s.Authority, &s.Username, &s.Description, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SQLiteSharingRepo) ListByUsername(username string) ([]model.SharingInfo, error) {
	rows, err := r.DB.Query(
		`SELECT id, path, expires_at, blocked, authority, username, description, created_at 
		 FROM sharing_info WHERE username = ? ORDER BY created_at DESC`,
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []model.SharingInfo
	for rows.Next() {
		var s model.SharingInfo
		if err := rows.Scan(&s.ID, &s.Path, &s.ExpiresAt, &s.Blocked, &s.Authority, &s.Username, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}
	return shares, nil
}

func (r *SQLiteSharingRepo) UpdateBlockedStatus(id string, username string, blocked bool) error {
	_, err := r.DB.Exec(`UPDATE sharing_info SET blocked = ? WHERE id = ? AND username = ?`, blocked, id, username)
	return err
}

func (r *SQLiteSharingRepo) Delete(id string, username string) (int64, error) {
	res, err := r.DB.Exec(`DELETE FROM sharing_info WHERE id = ? AND username = ?`, id, username)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
