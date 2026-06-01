package repository_test

import (
	"database/sql"
	"testing"
	"time"

	"go-file-server/internal/model"
	"go-file-server/internal/repository"
	"go-file-server/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTokenRepo(t *testing.T) (*repository.SQLiteRefreshTokenRepo, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteRefreshTokenRepo(db)
	return repo, db
}

func createTestToken(t *testing.T, repo *repository.SQLiteRefreshTokenRepo, username, familyID string) *model.RefreshToken {
	t.Helper()

	token := &model.RefreshToken{
		ID:         uuid.New().String(),
		Username:   username,
		TokenHash:  uuid.New().String(),
		FamilyID:   familyID,
		DeviceID:   "device-1",
		DeviceInfo: "test-agent",
		IPAddress:  "127.0.0.1",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	err := repo.Create(token)
	require.NoError(t, err)
	return token
}

func TestTokenRepo_Create(t *testing.T) {
	repo, _ := setupTokenRepo(t)
	token := createTestToken(t, repo, "testuser", "family-1")

	retrieved, err := repo.GetByTokenHash(token.TokenHash)
	require.NoError(t, err)
	assert.Equal(t, token.ID, retrieved.ID)
	assert.Equal(t, "testuser", retrieved.Username)
	assert.Equal(t, "family-1", retrieved.FamilyID)
	assert.Equal(t, "device-1", retrieved.DeviceID)
	assert.False(t, retrieved.IsRevoked)
}

func TestTokenRepo_GetByTokenHash_NotFound(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	_, err := repo.GetByTokenHash("nonexistent-hash")
	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestTokenRepo_GetActiveSessions_Empty(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	sessions, err := repo.GetActiveSessions("nouser")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestTokenRepo_GetActiveSessions_WithSessions(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "alice", "family-a")
	createTestToken(t, repo, "alice", "family-b")
	createTestToken(t, repo, "alice", "family-c")

	sessions, err := repo.GetActiveSessions("alice")
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestTokenRepo_GetActiveSessions_SkipsRevoked(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	token := createTestToken(t, repo, "bob", "family-x")
	err := repo.RevokeByID(token.ID)
	require.NoError(t, err)

	sessions, err := repo.GetActiveSessions("bob")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestTokenRepo_RevokeByID(t *testing.T) {
	repo, _ := setupTokenRepo(t)
	token := createTestToken(t, repo, "charlie", "family-1")

	err := repo.RevokeByID(token.ID)
	require.NoError(t, err)

	retrieved, err := repo.GetByTokenHash(token.TokenHash)
	require.NoError(t, err)
	assert.True(t, retrieved.IsRevoked, "token should be revoked")
}

func TestTokenRepo_RevokeByFamilyID(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "dave", "family-99")
	createTestToken(t, repo, "dave", "family-99") // same family

	err := repo.RevokeByFamilyID("family-99")
	require.NoError(t, err)

	sessions, err := repo.GetActiveSessions("dave")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestTokenRepo_RevokeByUsername(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "eve", "family-e1")
	createTestToken(t, repo, "frank", "family-f1")

	err := repo.RevokeByUsername("eve")
	require.NoError(t, err)

	// eve's sessions should be gone
	sessionsEve, err := repo.GetActiveSessions("eve")
	require.NoError(t, err)
	assert.Empty(t, sessionsEve)

	// frank's sessions should remain
	sessionsFrank, err := repo.GetActiveSessions("frank")
	require.NoError(t, err)
	assert.Len(t, sessionsFrank, 1)
}

func TestTokenRepo_DeleteExpired_DeletesRevoked(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "grace", "family-g0")
	err := repo.RevokeByFamilyID("family-g0")
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired()
	require.NoError(t, err)
	assert.Greater(t, deleted, int64(0))

	sessions, err := repo.GetActiveSessions("grace")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestTokenRepo_DeleteExpired_DeletesExpired(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	token := &model.RefreshToken{
		ID:         uuid.New().String(),
		Username:   "expireduser",
		TokenHash:  uuid.New().String(),
		FamilyID:   "family-expired",
		DeviceID:   "device-1",
		DeviceInfo: "test-agent",
		IPAddress:  "127.0.0.1",
		ExpiresAt:  time.Now().Add(-24 * time.Hour),
	}
	err := repo.Create(token)
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired()
	require.NoError(t, err)
	assert.Greater(t, deleted, int64(0))

	_, err = repo.GetByTokenHash(token.TokenHash)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestTokenRepo_DeleteExpired_KeepsActive(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	active := createTestToken(t, repo, "activeuser", "family-active")

	deleted, err := repo.DeleteExpired()
	require.NoError(t, err)

	retrieved, err := repo.GetByTokenHash(active.TokenHash)
	require.NoError(t, err)
	assert.False(t, retrieved.IsRevoked)
	_ = deleted
}

func TestTokenRepo_RevokeByUsernameAndFamilyID(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "grace", "family-g1")
	createTestToken(t, repo, "grace", "family-g2")

	rowsAffected, err := repo.RevokeByUsernameAndFamilyID("grace", "family-g1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	sessions, err := repo.GetActiveSessions("grace")
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "family-g2", sessions[0].FamilyID)
}
