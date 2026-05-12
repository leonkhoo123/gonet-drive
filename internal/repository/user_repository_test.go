package repository_test

import (
	"database/sql"
	"testing"

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

	retrieved, err := repo.GetByUsername("testuser")
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

	user, err := repo.GetByUsername("alice")
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "user", user.Role)
	assert.NotEmpty(t, user.ID)
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)

	_, err := repo.GetByUsername("nonexistent")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUserRepo_GetByID(t *testing.T) {
	repo, _ := setupUserRepo(t)
	created := testutil.CreateTestUser(t, repo.DB, "bob", "pass2", "admin")

	user, err := repo.GetByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "bob", user.Username)
}

func TestUserRepo_ListAll_Empty(t *testing.T) {
	repo, _ := setupUserRepo(t)

	users, err := repo.ListAll()
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestUserRepo_ListAll_WithUsers(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "user1", "p1", "user")
	testutil.CreateTestUser(t, repo.DB, "user2", "p2", "user")
	testutil.CreateTestUser(t, repo.DB, "user3", "p3", "admin")

	users, err := repo.ListAll()
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestUserRepo_Update(t *testing.T) {
	repo, _ := setupUserRepo(t)
	user := testutil.CreateTestUser(t, repo.DB, "charlie", "pass3", "user")

	user.Role = "admin"
	err := repo.Update(user)
	require.NoError(t, err)

	retrieved, err := repo.GetByUsername("charlie")
	require.NoError(t, err)
	assert.Equal(t, "admin", retrieved.Role)
}

func TestUserRepo_Delete(t *testing.T) {
	repo, _ := setupUserRepo(t)
	user := testutil.CreateTestUser(t, repo.DB, "dave", "pass4", "user")

	err := repo.Delete(user.ID)
	require.NoError(t, err)

	_, err = repo.GetByUsername("dave")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUserRepo_Exists_True(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "frank", "pass6", "user")

	exists, err := repo.Exists("frank")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUserRepo_Exists_False(t *testing.T) {
	repo, _ := setupUserRepo(t)

	exists, err := repo.Exists("nobody")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUserRepo_IncrementTokenVersion(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "grace", "pass7", "user")

	err := repo.IncrementTokenVersion("grace")
	require.NoError(t, err)

	user, err := repo.GetByUsername("grace")
	require.NoError(t, err)
	assert.Equal(t, 2, user.TokenVersion)
}

func TestUserRepo_IncrementTokenVersionByID(t *testing.T) {
	repo, _ := setupUserRepo(t)
	user := testutil.CreateTestUser(t, repo.DB, "henry", "pass8", "user")

	err := repo.IncrementTokenVersionByID(user.ID)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, retrieved.TokenVersion)
}

func TestUserRepo_UpdateMFASecret(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "iris", "pass9", "user")

	secret := "JBSWY3DPEHPK3PXP"
	err := repo.UpdateMFASecret("iris", secret)
	require.NoError(t, err)

	user, err := repo.GetByUsername("iris")
	require.NoError(t, err)
	require.NotNil(t, user.MFASecret)
	assert.Equal(t, secret, *user.MFASecret)
}

func TestUserRepo_EnableMFA(t *testing.T) {
	repo, _ := setupUserRepo(t)
	testutil.CreateTestUser(t, repo.DB, "jack", "pass10", "user")

	err := repo.EnableMFA("jack")
	require.NoError(t, err)

	user, err := repo.GetByUsername("jack")
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
	err := repo.Create(user)
	assert.Error(t, err)
}
