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

func setupShareRepo(t *testing.T) (*repository.SQLiteSharingRepo, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteSharingRepo(db)
	return repo, db
}

func createTestShare(t *testing.T, repo *repository.SQLiteSharingRepo, username, path, authority string) *model.SharingInfo {
	t.Helper()

	share := &model.SharingInfo{
		ID:          uuid.New().String(),
		Path:        path,
		PinHash:     "some-pin-hash",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Blocked:     false,
		Authority:   authority,
		Username:    username,
		Description: "test share",
		CreatedAt:   time.Now(),
	}
	err := repo.Create(share)
	require.NoError(t, err)
	return share
}

func TestShareRepo_Create(t *testing.T) {
	repo, _ := setupShareRepo(t)
	share := createTestShare(t, repo, "user1", "/test/path", "view")

	retrieved, err := repo.GetByID(share.ID)
	require.NoError(t, err)
	assert.Equal(t, share.ID, retrieved.ID)
	assert.Equal(t, "/test/path", retrieved.Path)
	assert.Equal(t, "user1", retrieved.Username)
	assert.Equal(t, "view", retrieved.Authority)
	assert.Equal(t, "test share", retrieved.Description)
	assert.False(t, retrieved.Blocked)
}

func TestShareRepo_GetByID_Found(t *testing.T) {
	repo, _ := setupShareRepo(t)
	share := createTestShare(t, repo, "alice", "/alice/docs", "modify")

	retrieved, err := repo.GetByID(share.ID)
	require.NoError(t, err)
	assert.Equal(t, "/alice/docs", retrieved.Path)
}

func TestShareRepo_GetByID_NotFound(t *testing.T) {
	repo, _ := setupShareRepo(t)

	_, err := repo.GetByID("nonexistent-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestShareRepo_ListByUsername_OnlyOwn(t *testing.T) {
	repo, _ := setupShareRepo(t)

	createTestShare(t, repo, "alice", "/alice/a", "view")
	createTestShare(t, repo, "alice", "/alice/b", "view")
	createTestShare(t, repo, "bob", "/bob/c", "view")

	shares, err := repo.ListByUsername("alice")
	require.NoError(t, err)
	assert.Len(t, shares, 2)
	for _, s := range shares {
		assert.Contains(t, s.Path, "/alice/")
	}
}

func TestShareRepo_ListByUsername_Empty(t *testing.T) {
	repo, _ := setupShareRepo(t)

	shares, err := repo.ListByUsername("nouser")
	require.NoError(t, err)
	assert.Empty(t, shares)
}

func TestShareRepo_UpdateBlockedStatus(t *testing.T) {
	repo, _ := setupShareRepo(t)
	share := createTestShare(t, repo, "charlie", "/charlie/docs", "view")

	err := repo.UpdateBlockedStatus(share.ID, "charlie", true)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(share.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.Blocked)
}

func TestShareRepo_Delete(t *testing.T) {
	repo, _ := setupShareRepo(t)
	share := createTestShare(t, repo, "dave", "/dave/docs", "view")

	rowsAffected, err := repo.Delete(share.ID, "dave")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	_, err = repo.GetByID(share.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestShareRepo_Delete_WrongUser(t *testing.T) {
	repo, _ := setupShareRepo(t)
	share := createTestShare(t, repo, "eve", "/eve/docs", "view")

	rowsAffected, err := repo.Delete(share.ID, "wronguser")
	require.NoError(t, err)
	assert.Equal(t, int64(0), rowsAffected)

	// Share should still exist
	_, err = repo.GetByID(share.ID)
	require.NoError(t, err)
}
