package repository_test

import (
	"context"
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
	err := repo.Create(context.Background(), token)
	require.NoError(t, err)
	return token
}

func TestTokenRepo_Create(t *testing.T) {
	repo, _ := setupTokenRepo(t)
	token := createTestToken(t, repo, "testuser", "family-1")

	retrieved, err := repo.GetByTokenHash(context.Background(), token.TokenHash)
	require.NoError(t, err)
	assert.Equal(t, token.ID, retrieved.ID)
	assert.Equal(t, "testuser", retrieved.Username)
	assert.Equal(t, "family-1", retrieved.FamilyID)
	assert.Equal(t, "device-1", retrieved.DeviceID)
	assert.False(t, retrieved.IsRevoked)
}

func TestTokenRepo_GetByTokenHash_NotFound(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	_, err := repo.GetByTokenHash(context.Background(), "nonexistent-hash")
	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestTokenRepo_GetActiveSessions_Empty(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	sessions, err := repo.GetActiveSessions(context.Background(), "nouser")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestTokenRepo_GetActiveSessions_WithSessions(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "alice", "family-a")
	createTestToken(t, repo, "alice", "family-b")
	createTestToken(t, repo, "alice", "family-c")

	sessions, err := repo.GetActiveSessions(context.Background(), "alice")
	require.NoError(t, err)
	assert.Len(t, sessions, 3)
}

func TestTokenRepo_GetActiveSessions_SkipsRevoked(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	token := createTestToken(t, repo, "bob", "family-x")
	err := repo.RevokeByID(context.Background(), token.ID)
	require.NoError(t, err)

	sessions, err := repo.GetActiveSessions(context.Background(), "bob")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestTokenRepo_RevokeByID(t *testing.T) {
	repo, _ := setupTokenRepo(t)
	token := createTestToken(t, repo, "charlie", "family-1")

	err := repo.RevokeByID(context.Background(), token.ID)
	require.NoError(t, err)

	retrieved, err := repo.GetByTokenHash(context.Background(), token.TokenHash)
	require.NoError(t, err)
	assert.True(t, retrieved.IsRevoked, "token should be revoked")
}

func TestTokenRepo_RevokeByFamilyID(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "dave", "family-99")
	createTestToken(t, repo, "dave", "family-99") // same family

	err := repo.RevokeByFamilyID(context.Background(), "family-99")
	require.NoError(t, err)

	sessions, err := repo.GetActiveSessions(context.Background(), "dave")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestTokenRepo_RevokeByUsername(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "eve", "family-e1")
	createTestToken(t, repo, "frank", "family-f1")

	err := repo.RevokeByUsername(context.Background(), "eve")
	require.NoError(t, err)

	// eve's sessions should be gone
	sessionsEve, err := repo.GetActiveSessions(context.Background(), "eve")
	require.NoError(t, err)
	assert.Empty(t, sessionsEve)

	// frank's sessions should remain
	sessionsFrank, err := repo.GetActiveSessions(context.Background(), "frank")
	require.NoError(t, err)
	assert.Len(t, sessionsFrank, 1)
}

func TestTokenRepo_DeleteExpired_DeletesRevoked(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "grace", "family-g0")
	err := repo.RevokeByFamilyID(context.Background(), "family-g0")
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired(context.Background())
	require.NoError(t, err)
	assert.Greater(t, deleted, int64(0))

	sessions, err := repo.GetActiveSessions(context.Background(), "grace")
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
	err := repo.Create(context.Background(), token)
	require.NoError(t, err)

	deleted, err := repo.DeleteExpired(context.Background())
	require.NoError(t, err)
	assert.Greater(t, deleted, int64(0))

	_, err = repo.GetByTokenHash(context.Background(), token.TokenHash)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestTokenRepo_DeleteExpired_KeepsActive(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	active := createTestToken(t, repo, "activeuser", "family-active")

	deleted, err := repo.DeleteExpired(context.Background())
	require.NoError(t, err)

	retrieved, err := repo.GetByTokenHash(context.Background(), active.TokenHash)
	require.NoError(t, err)
	assert.False(t, retrieved.IsRevoked)
	_ = deleted
}

func TestTokenRepo_RevokeByUsernameAndFamilyID(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "grace", "family-g1")
	createTestToken(t, repo, "grace", "family-g2")

	rowsAffected, err := repo.RevokeByUsernameAndFamilyID(context.Background(), "grace", "family-g1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	sessions, err := repo.GetActiveSessions(context.Background(), "grace")
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "family-g2", sessions[0].FamilyID)
}

// ---------- Finding 2.2: CountActiveSessions ----------

func TestTokenRepo_CountActiveSessions_Empty(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	count, err := repo.CountActiveSessions(context.Background(), "nouser")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTokenRepo_CountActiveSessions_Multiple(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	createTestToken(t, repo, "countuser", "family-c1")
	createTestToken(t, repo, "countuser", "family-c2")
	createTestToken(t, repo, "countuser", "family-c3")

	count, err := repo.CountActiveSessions(context.Background(), "countuser")
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestTokenRepo_CountActiveSessions_ExcludesRevoked(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	token := createTestToken(t, repo, "countrevoked", "family-cr1")
	createTestToken(t, repo, "countrevoked", "family-cr2")

	err := repo.RevokeByID(context.Background(), token.ID)
	require.NoError(t, err)

	count, err := repo.CountActiveSessions(context.Background(), "countrevoked")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// ---------- Finding 12.3: RotateTx ----------

func TestTokenRepo_RotateTx(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	oldToken := createTestToken(t, repo, "rotateuser", "family-r1")

	newToken := &model.RefreshToken{
		ID:         uuid.New().String(),
		Username:   "rotateuser",
		TokenHash:  uuid.New().String(),
		FamilyID:   "family-r1",
		DeviceID:   "device-1",
		DeviceInfo: "test-agent",
		IPAddress:  "127.0.0.1",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	err := repo.RotateTx(context.Background(), oldToken.ID, newToken)
	require.NoError(t, err)

	// Old token should be revoked
	oldRetrieved, err := repo.GetByTokenHash(context.Background(), oldToken.TokenHash)
	require.NoError(t, err)
	assert.True(t, oldRetrieved.IsRevoked, "old token should be revoked after rotation")

	// New token should exist and be active
	newRetrieved, err := repo.GetByTokenHash(context.Background(), newToken.TokenHash)
	require.NoError(t, err)
	assert.False(t, newRetrieved.IsRevoked, "new token should not be revoked")
	assert.Equal(t, "rotateuser", newRetrieved.Username)
}

func TestTokenRepo_RotateTx_SameFamily(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	oldToken := createTestToken(t, repo, "rotateuser2", "family-r2")

	newToken := &model.RefreshToken{
		ID:         uuid.New().String(),
		Username:   "rotateuser2",
		TokenHash:  uuid.New().String(),
		FamilyID:   "family-r2",
		DeviceID:   "device-2",
		DeviceInfo: "test-agent-2",
		IPAddress:  "10.0.0.1",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	err := repo.RotateTx(context.Background(), oldToken.ID, newToken)
	require.NoError(t, err)

	// Both tokens should be in the same family
	sessions, err := repo.GetActiveSessions(context.Background(), "rotateuser2")
	require.NoError(t, err)
	assert.Len(t, sessions, 1, "only new token should be active")
	assert.Equal(t, "family-r2", sessions[0].FamilyID)
}

func TestTokenRepo_RotateTx_InvalidOldID(t *testing.T) {
	repo, _ := setupTokenRepo(t)

	newToken := &model.RefreshToken{
		ID:         uuid.New().String(),
		Username:   "rotateuser3",
		TokenHash:  uuid.New().String(),
		FamilyID:   "family-r3",
		DeviceID:   "device-1",
		DeviceInfo: "test-agent",
		IPAddress:  "127.0.0.1",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	// Rotating with non-existent old ID should fail
	err := repo.RotateTx(context.Background(), "nonexistent-id", newToken)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
