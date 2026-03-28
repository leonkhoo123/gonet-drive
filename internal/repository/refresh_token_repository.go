package repository

import (
	"database/sql"
	"go-file-server/internal/model"
)

type RefreshTokenRepository interface {
	Create(rt *model.RefreshToken) error
	GetByTokenHash(hash string) (*model.RefreshToken, error)
	GetActiveSessions(username string) ([]model.RefreshToken, error)
	RevokeByID(id string) error
	RevokeByFamilyID(familyID string) error
	RevokeByUsername(username string) error
	RevokeByUsernameAndFamilyID(username string, familyID string) error
}

type SQLiteRefreshTokenRepo struct {
	DB *sql.DB
}

func NewSQLiteRefreshTokenRepo(db *sql.DB) *SQLiteRefreshTokenRepo {
	return &SQLiteRefreshTokenRepo{DB: db}
}

func (r *SQLiteRefreshTokenRepo) Create(rt *model.RefreshToken) error {
	_, err := r.DB.Exec(`
		INSERT INTO refresh_tokens (id, username, token_hash, family_id, device_id, device_info, ip_address, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, rt.ID, rt.Username, rt.TokenHash, rt.FamilyID, rt.DeviceID, rt.DeviceInfo, rt.IPAddress, rt.ExpiresAt)
	return err
}

func (r *SQLiteRefreshTokenRepo) GetByTokenHash(hash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.DB.QueryRow(`
		SELECT id, username, family_id, device_id, device_info, ip_address, expires_at, is_revoked 
		FROM refresh_tokens WHERE token_hash = ?
	`, hash).Scan(&rt.ID, &rt.Username, &rt.FamilyID, &rt.DeviceID, &rt.DeviceInfo, &rt.IPAddress, &rt.ExpiresAt, &rt.IsRevoked)
	if err != nil {
		return nil, err
	}
	rt.TokenHash = hash
	return &rt, nil
}

func (r *SQLiteRefreshTokenRepo) GetActiveSessions(username string) ([]model.RefreshToken, error) {
	rows, err := r.DB.Query(`
		SELECT family_id, device_id, device_info, ip_address, created_at, expires_at
		FROM refresh_tokens
		WHERE username = ? AND is_revoked = 0 AND expires_at > CURRENT_TIMESTAMP
		GROUP BY family_id
		ORDER BY MAX(created_at) DESC
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []model.RefreshToken
	for rows.Next() {
		var rt model.RefreshToken
		if err := rows.Scan(&rt.FamilyID, &rt.DeviceID, &rt.DeviceInfo, &rt.IPAddress, &rt.CreatedAt, &rt.ExpiresAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, rt)
	}
	return sessions, nil
}

func (r *SQLiteRefreshTokenRepo) RevokeByID(id string) error {
	_, err := r.DB.Exec("UPDATE refresh_tokens SET is_revoked = 1 WHERE id = ?", id)
	return err
}

func (r *SQLiteRefreshTokenRepo) RevokeByFamilyID(familyID string) error {
	_, err := r.DB.Exec("UPDATE refresh_tokens SET is_revoked = 1 WHERE family_id = ?", familyID)
	return err
}

func (r *SQLiteRefreshTokenRepo) RevokeByUsername(username string) error {
	_, err := r.DB.Exec("UPDATE refresh_tokens SET is_revoked = 1 WHERE username = ?", username)
	return err
}

func (r *SQLiteRefreshTokenRepo) RevokeByUsernameAndFamilyID(username string, familyID string) error {
	_, err := r.DB.Exec("UPDATE refresh_tokens SET is_revoked = 1 WHERE username = ? AND family_id = ?", username, familyID)
	return err
}
