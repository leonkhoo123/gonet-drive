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

func setupUserRepo(t *testing.T) (*repository.SQLiteUserRepo, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteUserRepo(db)
	return repo, db
}

func TestUserRepo_Create(t *testing.T) {
	repo, _ := setupUserRepo(t)

	user := testutil.CreateTestUser(t, repo.DB, "testuser", "password123", "user")

	retrieved, err := repo.GetByUsername(context.Background(), "testuser")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrieved.ID)
	assert.Equal(t, "testuser", retrieved.Username)
	assert.Equal(t, user.PasswordHash, retrieved.PasswordHash)
	assert.Equal(t, "user", retrieved.Role)
	assert.Equal(t, 1, retrieved.TokenVersion)
}

func TestUserRepo_GetByUsername_Found(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "alice", "pass1", "user")

	user, err := repo.GetByUsername(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "user", user.Role)
	assert.NotEmpty(t, user.ID)
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)

	_, err := repo.GetByUsername(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUserRepo_GetByID(t *testing.T) {
	repo, _ := setupUserRepo(t)
	created := testutil.CreateTestUser(t, repo.DB, "bob", "pass2", "admin")

	user, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "bob", user.Username)
}

func TestUserRepo_ListAll_Empty(t *testing.T) {
	repo, _ := setupUserRepo(t)

	users, err := repo.ListAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestUserRepo_ListAll_WithUsers(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "user1", "p1", "user")
	testutil.CreateTestUser(t, repo.DB, "user2", "p2", "user")
	testutil.CreateTestUser(t, repo.DB, "user3", "p3", "admin")

	users, err := repo.ListAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestUserRepo_Update(t *testing.T) {
	repo, _ := setupUserRepo(t)
	user := testutil.CreateTestUser(t, repo.DB, "charlie", "pass3", "user")

	user.Role = "admin"
	err := repo.Update(context.Background(), user)
	require.NoError(t, err)

	retrieved, err := repo.GetByUsername(context.Background(), "charlie")
	require.NoError(t, err)
	assert.Equal(t, "admin", retrieved.Role)
}

func TestUserRepo_Delete(t *testing.T) {
	repo, _ := setupUserRepo(t)
	user := testutil.CreateTestUser(t, repo.DB, "dave", "pass4", "user")

	err := repo.Delete(context.Background(), user.ID)
	require.NoError(t, err)

	_, err = repo.GetByUsername(context.Background(), "dave")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUserRepo_Exists_True(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "frank", "pass6", "user")

	exists, err := repo.Exists(context.Background(), "frank")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUserRepo_Exists_False(t *testing.T) {
	repo, _ := setupUserRepo(t)

	exists, err := repo.Exists(context.Background(), "nobody")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUserRepo_IncrementTokenVersion(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "grace", "pass7", "user")

	err := repo.IncrementTokenVersion(context.Background(), "grace")
	require.NoError(t, err)

	user, err := repo.GetByUsername(context.Background(), "grace")
	require.NoError(t, err)
	assert.Equal(t, 2, user.TokenVersion)
}

func TestUserRepo_IncrementTokenVersionByID(t *testing.T) {
	repo, _ := setupUserRepo(t)
	user := testutil.CreateTestUser(t, repo.DB, "henry", "pass8", "user")

	err := repo.IncrementTokenVersionByID(context.Background(), user.ID)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, retrieved.TokenVersion)
}

func TestUserRepo_UpdateMFASecret(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "iris", "pass9", "user")

	secret := "JBSWY3DPEHPK3PXP"
	err := repo.UpdateMFASecret(context.Background(), "iris", secret)
	require.NoError(t, err)

	user, err := repo.GetByUsername(context.Background(), "iris")
	require.NoError(t, err)
	require.NotNil(t, user.MFASecret)
	assert.Equal(t, secret, *user.MFASecret)
}

func TestUserRepo_EnableMFA(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "jack", "pass10", "user")

	err := repo.EnableMFA(context.Background(), "jack")
	require.NoError(t, err)

	user, err := repo.GetByUsername(context.Background(), "jack")
	require.NoError(t, err)
	assert.True(t, user.MFAEnabled)
}

func TestUserRepo_DuplicateUsername(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "kate", "pass11", "user")

	// Attempt to create a user with the same username should fail
	user := &model.User{
		ID:           uuid.New().String(),
		Username:     "kate",
		PasswordHash: "somehash",
		Role:         "user",
		TokenVersion: 1,
	}
	err := repo.Create(context.Background(), user)
	assert.Error(t, err)
}

// ---------- Finding 4.1: Password lockout ----------

func TestUserRepo_IncrementFailedAttempts(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "lockuser", "pass123", "user")

	// First attempt
	rows, err := repo.IncrementFailedAttempts(context.Background(), "lockuser", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, rows)

	user, err := repo.GetByUsername(context.Background(), "lockuser")
	require.NoError(t, err)
	assert.Equal(t, 1, user.FailedAttempts)
	assert.Nil(t, user.LockedUntil)
}

func TestUserRepo_IncrementFailedAttempts_WithLockout(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "lockuser2", "pass123", "user")

	lockTime := time.Now().Add(15 * time.Minute)
	rows, err := repo.IncrementFailedAttempts(context.Background(), "lockuser2", &lockTime)
	require.NoError(t, err)
	assert.Equal(t, 1, rows)

	user, err := repo.GetByUsername(context.Background(), "lockuser2")
	require.NoError(t, err)
	assert.Equal(t, 1, user.FailedAttempts)
	assert.NotNil(t, user.LockedUntil)
}

func TestUserRepo_ResetFailedAttempts(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "lockuser3", "pass123", "user")

	// Increment a few times
	lockTime := time.Now().Add(15 * time.Minute)
	_, err := repo.IncrementFailedAttempts(context.Background(), "lockuser3", &lockTime)
	require.NoError(t, err)
	_, err = repo.IncrementFailedAttempts(context.Background(), "lockuser3", &lockTime)
	require.NoError(t, err)

	// Reset
	err = repo.ResetFailedAttempts(context.Background(), "lockuser3")
	require.NoError(t, err)

	user, err := repo.GetByUsername(context.Background(), "lockuser3")
	require.NoError(t, err)
	assert.Equal(t, 0, user.FailedAttempts)
	assert.Nil(t, user.LockedUntil)
}

// ---------- Finding 12.1: Recovery codes ----------

func TestUserRepo_SaveAndRecoveryCodes(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "recoveryuser", "pass123", "user")

	codes := []string{"hash1", "hash2", "hash3"}
	err := repo.SaveRecoveryCodes(context.Background(), "recoveryuser", codes)
	require.NoError(t, err)

	retrieved, err := repo.GetRecoveryCodes(context.Background(), "recoveryuser")
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)
	assert.ElementsMatch(t, codes, retrieved)
}

func TestUserRepo_SaveRecoveryCodes_ReplacesExisting(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "recoveryuser2", "pass123", "user")

	err := repo.SaveRecoveryCodes(context.Background(), "recoveryuser2", []string{"old1", "old2"})
	require.NoError(t, err)

	err = repo.SaveRecoveryCodes(context.Background(), "recoveryuser2", []string{"new1", "new2", "new3"})
	require.NoError(t, err)

	retrieved, err := repo.GetRecoveryCodes(context.Background(), "recoveryuser2")
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)
	assert.Contains(t, retrieved, "new1")
	assert.NotContains(t, retrieved, "old1")
}

func TestUserRepo_ConsumeRecoveryCode(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "recoveryuser3", "pass123", "user")

	codes := []string{"code_a", "code_b", "code_c"}
	err := repo.SaveRecoveryCodes(context.Background(), "recoveryuser3", codes)
	require.NoError(t, err)

	// Consume one code
	err = repo.ConsumeRecoveryCode(context.Background(), "recoveryuser3", "code_b")
	require.NoError(t, err)

	retrieved, err := repo.GetRecoveryCodes(context.Background(), "recoveryuser3")
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)
	assert.NotContains(t, retrieved, "code_b")
	assert.Contains(t, retrieved, "code_a")
	assert.Contains(t, retrieved, "code_c")
}

func TestUserRepo_ConsumeRecoveryCode_NotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "recoveryuser4", "pass123", "user")

	err := repo.SaveRecoveryCodes(context.Background(), "recoveryuser4", []string{"code_x"})
	require.NoError(t, err)

	err = repo.ConsumeRecoveryCode(context.Background(), "recoveryuser4", "nonexistent")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ---------- Finding 12.1: DisableMFA ----------

func TestUserRepo_DisableMFA(t *testing.T) {
	repo, _ := setupUserRepo(t)
	user := testutil.CreateTestUser(t, repo.DB, "disablemfa", "pass123", "user")

	// Enable MFA first
	err := repo.UpdateMFASecret(context.Background(), "disablemfa", "JBSWY3DPEHPK3PXP")
	require.NoError(t, err)
	err = repo.EnableMFA(context.Background(), "disablemfa")
	require.NoError(t, err)

	// Verify MFA is enabled
	user, err = repo.GetByUsername(context.Background(), "disablemfa")
	require.NoError(t, err)
	assert.True(t, user.MFAEnabled)

	// Disable MFA
	err = repo.DisableMFA(context.Background(), "disablemfa")
	require.NoError(t, err)

	user, err = repo.GetByUsername(context.Background(), "disablemfa")
	require.NoError(t, err)
	assert.False(t, user.MFAEnabled)
	assert.Nil(t, user.MFASecret)
}
