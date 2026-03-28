package repository

import (
	"database/sql"
	"go-file-server/internal/model"
)

type UserRepository interface {
	GetByUsername(username string) (*model.User, error)
	GetByID(id string) (*model.User, error)
	ListAll() ([]*model.User, error)
	Create(user *model.User) error
	Update(user *model.User) error
	Delete(id string) error
	IncrementTokenVersion(username string) error
	IncrementTokenVersionByID(id string) error
	Exists(username string) (bool, error)
	UpdateMFASecret(username string, secret string) error
	EnableMFA(username string) error
}

type SQLiteUserRepo struct {
	DB *sql.DB
}

func NewSQLiteUserRepo(db *sql.DB) *SQLiteUserRepo {
	return &SQLiteUserRepo{DB: db}
}

func (r *SQLiteUserRepo) GetByUsername(username string) (*model.User, error) {
	var u model.User
	err := r.DB.QueryRow(
		`SELECT id, username, password_hash, role, mfa_secret, mfa_enabled, mfa_mandatory, storage_quota, created_at, updated_at, failed_attempts, locked_until, token_version 
		 FROM users WHERE username = ?`, username,
	).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MFASecret, &u.MFAEnabled, &u.MFAMandatory,
		&u.StorageQuota, &u.CreatedAt, &u.UpdatedAt, &u.FailedAttempts, &u.LockedUntil, &u.TokenVersion,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLiteUserRepo) GetByID(id string) (*model.User, error) {
	var u model.User
	err := r.DB.QueryRow(
		`SELECT id, username, password_hash, role, mfa_secret, mfa_enabled, mfa_mandatory, storage_quota, created_at, updated_at, failed_attempts, locked_until, token_version 
		 FROM users WHERE id = ?`, id,
	).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MFASecret, &u.MFAEnabled, &u.MFAMandatory,
		&u.StorageQuota, &u.CreatedAt, &u.UpdatedAt, &u.FailedAttempts, &u.LockedUntil, &u.TokenVersion,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *SQLiteUserRepo) ListAll() ([]*model.User, error) {
	rows, err := r.DB.Query(`SELECT id, username, password_hash, role, mfa_secret, mfa_enabled, mfa_mandatory, storage_quota, created_at, updated_at, failed_attempts, locked_until, token_version FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MFASecret, &u.MFAEnabled, &u.MFAMandatory,
			&u.StorageQuota, &u.CreatedAt, &u.UpdatedAt, &u.FailedAttempts, &u.LockedUntil, &u.TokenVersion,
		); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *SQLiteUserRepo) Create(user *model.User) error {
	_, err := r.DB.Exec(
		`INSERT INTO users (id, username, password_hash, role, mfa_secret, mfa_enabled, mfa_mandatory, storage_quota, created_at, updated_at, failed_attempts, locked_until, token_version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, user.Role, user.MFASecret, user.MFAEnabled, user.MFAMandatory,
		user.StorageQuota, user.CreatedAt, user.UpdatedAt, user.FailedAttempts, user.LockedUntil, user.TokenVersion,
	)
	return err
}

func (r *SQLiteUserRepo) Update(user *model.User) error {
	_, err := r.DB.Exec(
		`UPDATE users SET username = ?, password_hash = ?, role = ?, mfa_secret = ?, mfa_enabled = ?, mfa_mandatory = ?, storage_quota = ?, updated_at = ?, failed_attempts = ?, locked_until = ?, token_version = ?
		 WHERE id = ?`,
		user.Username, user.PasswordHash, user.Role, user.MFASecret, user.MFAEnabled, user.MFAMandatory, user.StorageQuota,
		user.UpdatedAt, user.FailedAttempts, user.LockedUntil, user.TokenVersion, user.ID,
	)
	return err
}

func (r *SQLiteUserRepo) Delete(id string) error {
	_, err := r.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *SQLiteUserRepo) IncrementTokenVersion(username string) error {
	_, err := r.DB.Exec(`UPDATE users SET token_version = token_version + 1 WHERE username = ?`, username)
	return err
}

func (r *SQLiteUserRepo) IncrementTokenVersionByID(id string) error {
	_, err := r.DB.Exec(`UPDATE users SET token_version = token_version + 1 WHERE id = ?`, id)
	return err
}

func (r *SQLiteUserRepo) Exists(username string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`, username).Scan(&exists)
	return exists, err
}

func (r *SQLiteUserRepo) UpdateMFASecret(username string, secret string) error {
	_, err := r.DB.Exec("UPDATE users SET mfa_secret = ? WHERE username = ?", secret, username)
	return err
}

func (r *SQLiteUserRepo) EnableMFA(username string) error {
	_, err := r.DB.Exec("UPDATE users SET mfa_enabled = 1 WHERE username = ?", username)
	return err
}
