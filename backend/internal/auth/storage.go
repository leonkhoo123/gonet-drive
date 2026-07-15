package auth

import (
	"context"
	"time"

	gonetauth "github.com/leonkhoo123/gonet-auth"

	"go-file-server/internal/model"
	"go-file-server/internal/repository"

	"github.com/google/uuid"
)

// Compile-time checks that the adapters satisfy the library interfaces.
var (
	_ gonetauth.UserLookup       = (*SQLiteUserStore)(nil)
	_ gonetauth.UserMFAStore     = (*SQLiteUserStore)(nil)
	_ gonetauth.UserLockoutStore = (*SQLiteUserStore)(nil)
	_ gonetauth.TokenStore       = (*SQLiteTokenStore)(nil)
)

// SQLiteUserStore adapts the existing UserRepository to gonetauth user interfaces.
type SQLiteUserStore struct {
	Repo repository.UserRepository
}

func (s *SQLiteUserStore) GetByUsername(ctx context.Context, username string) (*gonetauth.User, error) {
	u, err := s.Repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return toGonetUser(u), nil
}

func (s *SQLiteUserStore) GetByID(ctx context.Context, id string) (*gonetauth.User, error) {
	u, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toGonetUser(u), nil
}

func (s *SQLiteUserStore) Exists(ctx context.Context, username string) (bool, error) {
	return s.Repo.Exists(ctx, username)
}

func (s *SQLiteUserStore) Create(ctx context.Context, user *gonetauth.User) error {
	now := time.Now()
	return s.Repo.Create(ctx, &model.User{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
		MFAMandatory: user.MFAMandatory,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (s *SQLiteUserStore) Delete(ctx context.Context, id string) error {
	return s.Repo.Delete(ctx, id)
}

func (s *SQLiteUserStore) ListUsers(ctx context.Context) ([]gonetauth.User, error) {
	users, err := s.Repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]gonetauth.User, 0, len(users))
	for _, u := range users {
		result = append(result, *toGonetUser(u))
	}
	return result, nil
}

func (s *SQLiteUserStore) HasAdmin(ctx context.Context) (bool, error) {
	return s.Repo.HasAdmin(ctx)
}

func (s *SQLiteUserStore) CountAdmins(ctx context.Context) (int, error) {
	return s.Repo.CountAdmins(ctx)
}

func (s *SQLiteUserStore) UpdateMFASecret(ctx context.Context, username, secret string) error {
	return s.Repo.UpdateMFASecret(ctx, username, secret)
}

func (s *SQLiteUserStore) EnableMFA(ctx context.Context, username string) error {
	return s.Repo.EnableMFA(ctx, username)
}

func (s *SQLiteUserStore) DisableMFA(ctx context.Context, username string) error {
	return s.Repo.DisableMFA(ctx, username)
}

func (s *SQLiteUserStore) IncrementTokenVersion(ctx context.Context, username string) error {
	return s.Repo.IncrementTokenVersion(ctx, username)
}

func (s *SQLiteUserStore) IncrementTokenVersionByID(ctx context.Context, id string) error {
	return s.Repo.IncrementTokenVersionByID(ctx, id)
}

func (s *SQLiteUserStore) IncrementFailedAttempts(ctx context.Context, username string, lockedUntil *time.Time) (int, error) {
	return s.Repo.IncrementFailedAttempts(ctx, username, lockedUntil)
}

func (s *SQLiteUserStore) ResetFailedAttempts(ctx context.Context, username string) error {
	return s.Repo.ResetFailedAttempts(ctx, username)
}

func (s *SQLiteUserStore) SaveRecoveryCodes(ctx context.Context, username string, hashedCodes []string) error {
	return s.Repo.SaveRecoveryCodes(ctx, username, hashedCodes)
}

func (s *SQLiteUserStore) GetRecoveryCodes(ctx context.Context, username string) ([]string, error) {
	return s.Repo.GetRecoveryCodes(ctx, username)
}

func (s *SQLiteUserStore) ConsumeRecoveryCode(ctx context.Context, username string, codeHash string) error {
	return s.Repo.ConsumeRecoveryCode(ctx, username, codeHash)
}

func toGonetUser(u *model.User) *gonetauth.User {
	if u == nil {
		return nil
	}
	var mfaSecret string
	if u.MFASecret != nil {
		mfaSecret = *u.MFASecret
	}
	return &gonetauth.User{
		ID:             u.ID,
		Username:       u.Username,
		PasswordHash:   u.PasswordHash,
		Role:           u.Role,
		MFASecret:      mfaSecret,
		MFAEnabled:     u.MFAEnabled,
		MFAMandatory:   u.MFAMandatory,
		TokenVersion:   u.TokenVersion,
		FailedAttempts: u.FailedAttempts,
		LockedUntil:    u.LockedUntil,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

// SQLiteTokenStore adapts RefreshTokenRepository to gonetauth.TokenStore.
type SQLiteTokenStore struct {
	Repo repository.RefreshTokenRepository
}

func (s *SQLiteTokenStore) Create(ctx context.Context, token *gonetauth.RefreshToken) error {
	return s.Repo.Create(ctx, toModelRefreshToken(token))
}

func (s *SQLiteTokenStore) GetByTokenHash(ctx context.Context, hash string) (*gonetauth.RefreshToken, error) {
	rt, err := s.Repo.GetByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return toGonetRefreshToken(rt), nil
}

func (s *SQLiteTokenStore) GetActiveSessions(ctx context.Context, username string) ([]gonetauth.SessionInfo, error) {
	sessions, err := s.Repo.GetActiveSessions(ctx, username)
	if err != nil {
		return nil, err
	}
	var infos []gonetauth.SessionInfo
	for _, sess := range sessions {
		infos = append(infos, gonetauth.SessionInfo{
			FamilyID:   sess.FamilyID,
			DeviceID:   sess.DeviceID,
			DeviceInfo: sess.DeviceInfo,
			IPAddress:  sess.IPAddress,
			CreatedAt:  sess.CreatedAt.Format(time.RFC3339),
			ExpiresAt:  sess.ExpiresAt.Format(time.RFC3339),
		})
	}
	return infos, nil
}

func (s *SQLiteTokenStore) CountActiveSessions(ctx context.Context, username string) (int, error) {
	return s.Repo.CountActiveSessions(ctx, username)
}

func (s *SQLiteTokenStore) RevokeByID(ctx context.Context, id string) error {
	return s.Repo.RevokeByID(ctx, id)
}

func (s *SQLiteTokenStore) RevokeByFamilyID(ctx context.Context, familyID string) error {
	return s.Repo.RevokeByFamilyID(ctx, familyID)
}

func (s *SQLiteTokenStore) RevokeByUsername(ctx context.Context, username string) error {
	return s.Repo.RevokeByUsername(ctx, username)
}

func (s *SQLiteTokenStore) RevokeByUsernameAndFamilyID(ctx context.Context, username, familyID string) (int64, error) {
	return s.Repo.RevokeByUsernameAndFamilyID(ctx, username, familyID)
}

func (s *SQLiteTokenStore) DeleteExpired(ctx context.Context) (int64, error) {
	return s.Repo.DeleteExpired(ctx)
}

func (s *SQLiteTokenStore) RotateTx(ctx context.Context, oldID string, newToken *gonetauth.RefreshToken) error {
	return s.Repo.RotateTx(ctx, oldID, toModelRefreshToken(newToken))
}

func toModelRefreshToken(t *gonetauth.RefreshToken) *model.RefreshToken {
	id := t.ID
	if id == "" {
		id = uuid.New().String()
	}
	return &model.RefreshToken{
		ID:         id,
		Username:   t.Username,
		TokenHash:  t.TokenHash,
		FamilyID:   t.FamilyID,
		DeviceID:   t.DeviceID,
		DeviceInfo: t.DeviceInfo,
		IPAddress:  t.IPAddress,
		ExpiresAt:  t.ExpiresAt,
	}
}

func toGonetRefreshToken(rt *model.RefreshToken) *gonetauth.RefreshToken {
	if rt == nil {
		return nil
	}
	return &gonetauth.RefreshToken{
		ID:         rt.ID,
		Username:   rt.Username,
		TokenHash:  rt.TokenHash,
		FamilyID:   rt.FamilyID,
		DeviceID:   rt.DeviceID,
		DeviceInfo: rt.DeviceInfo,
		IPAddress:  rt.IPAddress,
		ExpiresAt:  rt.ExpiresAt,
		IsRevoked:  rt.IsRevoked,
		CreatedAt:  rt.CreatedAt,
	}
}
