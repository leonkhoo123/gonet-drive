package auth

import (
	"context"
	"strconv"
	"time"

	gonetauth "github.com/leonkhoo123/gonet-auth"

	"go-file-server/internal/model"
	"go-file-server/internal/repository"

	"github.com/google/uuid"
)

// SQLiteUserStore adapts the existing UserRepository to gonetauth.UserStore.
type SQLiteUserStore struct {
	Repo repository.UserRepository
}

func (s *SQLiteUserStore) GetByUsername(_ context.Context, username string) (*gonetauth.User, error) {
	u, err := s.Repo.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	return toGonetUser(u), nil
}

func (s *SQLiteUserStore) GetByID(_ context.Context, id int64) (*gonetauth.User, error) {
	u, err := s.Repo.GetByID(strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	return toGonetUser(u), nil
}

func (s *SQLiteUserStore) Exists(username string) (bool, error) {
	return s.Repo.Exists(username)
}

func (s *SQLiteUserStore) UpdateMFASecret(username, secret string) error {
	return s.Repo.UpdateMFASecret(username, secret)
}

func (s *SQLiteUserStore) EnableMFA(username string) error {
	return s.Repo.EnableMFA(username)
}

func (s *SQLiteUserStore) IncrementTokenVersion(username string) error {
	return s.Repo.IncrementTokenVersion(username)
}

func (s *SQLiteUserStore) IncrementTokenVersionByID(id int64) error {
	return s.Repo.IncrementTokenVersionByID(strconv.FormatInt(id, 10))
}

func toGonetUser(u *model.User) *gonetauth.User {
	if u == nil {
		return nil
	}
	id, _ := strconv.ParseInt(u.ID, 10, 64)
	var mfaSecret string
	if u.MFASecret != nil {
		mfaSecret = *u.MFASecret
	}
	return &gonetauth.User{
		ID:             id,
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

func (s *SQLiteTokenStore) Create(_ context.Context, token *gonetauth.RefreshToken) error {
	return s.Repo.Create(toModelRefreshToken(token))
}

func (s *SQLiteTokenStore) GetByTokenHash(hash string) (*gonetauth.RefreshToken, error) {
	rt, err := s.Repo.GetByTokenHash(hash)
	if err != nil {
		return nil, err
	}
	return toGonetRefreshToken(rt), nil
}

func (s *SQLiteTokenStore) GetActiveSessions(username string) ([]gonetauth.SessionInfo, error) {
	sessions, err := s.Repo.GetActiveSessions(username)
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

func (s *SQLiteTokenStore) RevokeByID(id int64) error {
	return s.Repo.RevokeByID(strconv.FormatInt(id, 10))
}

func (s *SQLiteTokenStore) RevokeByFamilyID(familyID string) error {
	return s.Repo.RevokeByFamilyID(familyID)
}

func (s *SQLiteTokenStore) RevokeByUsername(username string) error {
	return s.Repo.RevokeByUsername(username)
}

func (s *SQLiteTokenStore) RevokeByUsernameAndFamilyID(username, familyID string) (int64, error) {
	return s.Repo.RevokeByUsernameAndFamilyID(username, familyID)
}

func (s *SQLiteTokenStore) DeleteExpired() (int64, error) {
	return s.Repo.DeleteExpired()
}

var nilUUID = uuid.UUID{}

func toModelRefreshToken(t *gonetauth.RefreshToken) *model.RefreshToken {
	id := strconv.FormatInt(t.ID, 10)
	if t.ID == 0 {
		id = strconv.FormatInt(time.Now().UnixNano(), 10)
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
	id, _ := strconv.ParseInt(rt.ID, 10, 64)
	return &gonetauth.RefreshToken{
		ID:         id,
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
