package repository_test

import (
	"database/sql"
	"testing"
	"time"

	"go-file-server/internal/repository"
	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupVideoIntegrityRepo(t *testing.T) (*repository.SQLiteVideoIntegrityRepo, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	return repo, db
}

func TestVideoIntegrityRepo_UpsertInsert(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	err := repo.Upsert("abc123", "/data/video.mp4", "corrupt_avcC", "avc1.000032")
	require.NoError(t, err)

	corruptSet, err := repo.GetCorruptHashes([]string{"abc123"})
	require.NoError(t, err)
	assert.True(t, corruptSet["abc123"])

	count, err := repo.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestVideoIntegrityRepo_UpsertUpdate(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	require.NoError(t, repo.Upsert("abc123", "/data/old.mp4", "corrupt_avcC", "avc1.000032"))
	require.NoError(t, repo.Upsert("abc123", "/data/new.mp4", "corrupt_avcC", "avc1.000032"))

	entries, err := repo.All()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "/data/new.mp4", entries[0].FilePath)

	count, err := repo.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestVideoIntegrityRepo_DeleteAll(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	require.NoError(t, repo.Upsert("abc", "/a.mp4", "corrupt_avcC", ""))
	require.NoError(t, repo.Upsert("def", "/b.mp4", "corrupt_avcC", ""))

	count, err := repo.Count()
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	require.NoError(t, repo.DeleteAll())

	count, err = repo.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestVideoIntegrityRepo_GetCorruptHashes(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	require.NoError(t, repo.Upsert("aaa", "/a.mp4", "corrupt_avcC", ""))
	require.NoError(t, repo.Upsert("bbb", "/b.mp4", "corrupt_avcC", ""))
	require.NoError(t, repo.Upsert("ccc", "/c.mp4", "corrupt_avcC", ""))

	corrupt, err := repo.GetCorruptHashes([]string{"aaa", "ccc", "ddd"})
	require.NoError(t, err)
	assert.True(t, corrupt["aaa"])
	assert.False(t, corrupt["bbb"]) // not requested
	assert.True(t, corrupt["ccc"])
	assert.False(t, corrupt["ddd"]) // not in db
	assert.Len(t, corrupt, 2)
}

func TestVideoIntegrityRepo_GetCorruptHashesEmpty(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	corrupt, err := repo.GetCorruptHashes([]string{})
	require.NoError(t, err)
	assert.Empty(t, corrupt)

	corrupt, err = repo.GetCorruptHashes(nil)
	require.NoError(t, err)
	assert.Empty(t, corrupt)
}

func TestVideoIntegrityRepo_All(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	require.NoError(t, repo.Upsert("aaa", "/a.mp4", "corrupt_avcC", "avc1.000032"))
	require.NoError(t, repo.Upsert("bbb", "/b.mp4", "corrupt_avcC", "avc1.000033"))

	entries, err := repo.All()
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	found := map[string]bool{}
	for _, e := range entries {
		found[e.Hash] = true
		assert.Equal(t, "corrupt_avcC", e.IssueType)
		assert.NotEmpty(t, e.MimeCodecString)
		assert.False(t, e.DetectedAt.IsZero())
		assert.False(t, e.LastCheckedAt.IsZero())
	}
	assert.True(t, found["aaa"])
	assert.True(t, found["bbb"])
}

func TestVideoIntegrityRepo_AllEmpty(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	entries, err := repo.All()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestVideoIntegrityRepo_CountEmpty(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	count, err := repo.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestVideoIntegrityRepo_LastScanTime(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	lt, err := repo.LastScanTime()
	require.NoError(t, err)
	assert.Nil(t, lt)

	require.NoError(t, repo.Upsert("aaa", "/a.mp4", "corrupt_avcC", ""))

	lt, err = repo.LastScanTime()
	require.NoError(t, err)
	require.NotNil(t, lt)
	assert.WithinDuration(t, time.Now(), *lt, 2*time.Second)
}

func TestVideoIntegrityRepo_UpsertUpdatesLastChecked(t *testing.T) {
	repo, _ := setupVideoIntegrityRepo(t)

	require.NoError(t, repo.Upsert("aaa", "/a.mp4", "corrupt_avcC", ""))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, repo.Upsert("aaa", "/a.mp4", "corrupt_avcC", ""))

	entries, err := repo.All()
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.True(t, entries[0].LastCheckedAt.After(entries[0].DetectedAt) ||
		entries[0].LastCheckedAt.Equal(entries[0].DetectedAt))
}
