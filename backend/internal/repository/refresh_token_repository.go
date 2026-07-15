package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-file-server/internal/model"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, rt *model.RefreshToken) error
	GetByTokenHash(ctx context.Context, hash string) (*model.RefreshToken, error)
	GetActiveSessions(ctx context.Context, username string) ([]model.RefreshToken, error)
	CountActiveSessions(ctx context.Context, username string) (int, error)
	RevokeByID(ctx context.Context, id string) error
	RevokeByFamilyID(ctx context.Context, familyID string) error
	RevokeByUsername(ctx context.Context, username string) error
	RevokeByUsernameAndFamilyID(ctx context.Context, username string, familyID string) (int64, error)
	DeleteExpired(ctx context.Context) (int64, error)
	RotateTx(ctx context.Context, oldID string, newToken *model.RefreshToken) error
}

type SQLiteRefreshTokenRepo struct {
	DB *sql.DB
}

func NewSQLiteRefreshTokenRepo(db *sql.DB) *SQLiteRefreshTokenRepo {
	return &SQLiteRefreshTokenRepo{DB: db}
}

func (r *SQLiteRefreshTokenRepo) Create(ctx context.Context, rt *model.RefreshToken) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, username, token_hash, family_id, device_id, device_info, ip_address, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, rt.ID, rt.Username, rt.TokenHash, rt.FamilyID, rt.DeviceID, rt.DeviceInfo, rt.IPAddress, rt.ExpiresAt)
	return err
}

func (r *SQLiteRefreshTokenRepo) GetByTokenHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, username, family_id, device_id, device_info, ip_address, expires_at, is_revoked 
		FROM refresh_tokens WHERE token_hash = ?
	`, hash).Scan(&rt.ID, &rt.Username, &rt.FamilyID, &rt.DeviceID, &rt.DeviceInfo, &rt.IPAddress, &rt.ExpiresAt, &rt.IsRevoked)
	if err != nil {
		return nil, err
	}
	rt.TokenHash = hash
	return &rt, nil
}

func (r *SQLiteRefreshTokenRepo) GetActiveSessions(ctx context.Context, username string) ([]model.RefreshToken, error) {
	rows, err := r.DB.QueryContext(ctx, `
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

func (r *SQLiteRefreshTokenRepo) RevokeByID(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE refresh_tokens SET is_revoked = 1 WHERE id = ?", id)
	return err
}

func (r *SQLiteRefreshTokenRepo) RevokeByFamilyID(ctx context.Context, familyID string) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE refresh_tokens SET is_revoked = 1 WHERE family_id = ?", familyID)
	return err
}

func (r *SQLiteRefreshTokenRepo) RevokeByUsername(ctx context.Context, username string) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE refresh_tokens SET is_revoked = 1 WHERE username = ?", username)
	return err
}

func (r *SQLiteRefreshTokenRepo) RevokeByUsernameAndFamilyID(ctx context.Context, username string, familyID string) (int64, error) {
	result, err := r.DB.ExecContext(ctx, "UPDATE refresh_tokens SET is_revoked = 1 WHERE username = ? AND family_id = ?", username, familyID)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected, err
}

func (r *SQLiteRefreshTokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.DB.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE is_revoked = 1 OR expires_at <= CURRENT_TIMESTAMP")
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected, err
}

func (r *SQLiteRefreshTokenRepo) CountActiveSessions(ctx context.Context, username string) (int, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT family_id)
		FROM refresh_tokens
		WHERE username = ? AND is_revoked = 0 AND expires_at > CURRENT_TIMESTAMP
	`, username).Scan(&count)
	return count, err
}

func (r *SQLiteRefreshTokenRepo) RotateTx(ctx context.Context, oldID string, newToken *model.RefreshToken) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, "UPDATE refresh_tokens SET is_revoked = 1 WHERE id = ?", oldID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("refresh token %s not found", oldID)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, username, token_hash, family_id, device_id, device_info, ip_address, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, newToken.ID, newToken.Username, newToken.TokenHash, newToken.FamilyID, newToken.DeviceID, newToken.DeviceInfo, newToken.IPAddress, newToken.ExpiresAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}
