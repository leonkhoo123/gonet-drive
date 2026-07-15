package repository_test

import (
	"testing"

	"go-file-server/internal/repository"
	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPinnedFolderRepo(t *testing.T) repository.PinnedFolderRepository {
	t.Helper()
	db := testutil.SetupTestDB(t)
	return repository.NewSQLitePinnedFolderRepo(db)
}

func TestPinnedFolderRepo_Add(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	err := repo.Add("user1", "/docs/work")
	require.NoError(t, err)

	exists, err := repo.Exists("user1", "/docs/work")
	require.NoError(t, err)
	assert.True(t, exists)

	count, err := repo.Count("user1")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPinnedFolderRepo_AddDuplicate(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	require.NoError(t, repo.Add("user1", "/docs/work"))
	err := repo.Add("user1", "/docs/work")
	require.NoError(t, err)

	count, err := repo.Count("user1")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestPinnedFolderRepo_AddMultipleUsers(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	require.NoError(t, repo.Add("user1", "/docs/a"))
	require.NoError(t, repo.Add("user2", "/docs/b"))

	count1, err := repo.Count("user1")
	require.NoError(t, err)
	assert.Equal(t, 1, count1)

	count2, err := repo.Count("user2")
	require.NoError(t, err)
	assert.Equal(t, 1, count2)
}

func TestPinnedFolderRepo_Remove(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	require.NoError(t, repo.Add("user1", "/docs/work"))

	rows, err := repo.Remove("user1", "/docs/work")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	exists, err := repo.Exists("user1", "/docs/work")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPinnedFolderRepo_RemoveNotFound(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	rows, err := repo.Remove("user1", "/nonexistent")
	require.NoError(t, err)
	assert.Equal(t, int64(0), rows)
}

func TestPinnedFolderRepo_List(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	require.NoError(t, repo.Add("user1", "/docs/c"))
	require.NoError(t, repo.Add("user1", "/docs/a"))
	require.NoError(t, repo.Add("user1", "/docs/b"))

	folders, err := repo.List("user1")
	require.NoError(t, err)
	require.Len(t, folders, 3)

	assert.Equal(t, "/docs/c", folders[0].Path)
	assert.Equal(t, 0, folders[0].Position)
	assert.Equal(t, "/docs/a", folders[1].Path)
	assert.Equal(t, 1, folders[1].Position)
	assert.Equal(t, "/docs/b", folders[2].Path)
	assert.Equal(t, 2, folders[2].Position)
}

func TestPinnedFolderRepo_ListEmpty(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	folders, err := repo.List("user1")
	require.NoError(t, err)
	assert.Empty(t, folders)
}

func TestPinnedFolderRepo_Count(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	require.NoError(t, repo.Add("user1", "/a"))
	require.NoError(t, repo.Add("user1", "/b"))
	require.NoError(t, repo.Add("user2", "/c"))

	count, err := repo.Count("user1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestPinnedFolderRepo_Reorder(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	require.NoError(t, repo.Add("user1", "/a"))
	require.NoError(t, repo.Add("user1", "/b"))
	require.NoError(t, repo.Add("user1", "/c"))

	err := repo.Reorder("user1", []string{"/c", "/a", "/b"})
	require.NoError(t, err)

	folders, err := repo.List("user1")
	require.NoError(t, err)
	require.Len(t, folders, 3)

	assert.Equal(t, "/c", folders[0].Path)
	assert.Equal(t, 0, folders[0].Position)
	assert.Equal(t, "/a", folders[1].Path)
	assert.Equal(t, 1, folders[1].Position)
	assert.Equal(t, "/b", folders[2].Path)
	assert.Equal(t, 2, folders[2].Position)
}

func TestPinnedFolderRepo_Exists(t *testing.T) {
	repo := setupPinnedFolderRepo(t)

	require.NoError(t, repo.Add("user1", "/docs"))

	exists, err := repo.Exists("user1", "/docs")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.Exists("user1", "/notexist")
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = repo.Exists("user2", "/docs")
	require.NoError(t, err)
	assert.False(t, exists)
}
