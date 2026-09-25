package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadCompletion_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	completion := uploadCompletion{
		VirtualPath:  "/dest/file.txt",
		PhysicalPath: filepath.Join(tempDir, "file.txt"),
		Size:         123,
		CompletedAt:  time.Now().Round(time.Second),
	}

	require.NoError(t, writeUploadCompletion(tempDir, completion))

	got, ok := readUploadCompletion(tempDir)
	require.True(t, ok)
	assert.Equal(t, completion.VirtualPath, got.VirtualPath)
	assert.Equal(t, completion.PhysicalPath, got.PhysicalPath)
	assert.Equal(t, completion.Size, got.Size)
}

func TestReadUploadCompletion_MissingOrCorrupt(t *testing.T) {
	tempDir := t.TempDir()

	_, ok := readUploadCompletion(tempDir)
	assert.False(t, ok)

	require.NoError(t, os.WriteFile(uploadCompletionPath(tempDir), []byte("not json"), 0o644))
	_, ok = readUploadCompletion(tempDir)
	assert.False(t, ok)
}

func TestCompletedUpload_RequiresOutputFile(t *testing.T) {
	root := t.TempDir()
	identifier := "upload-abc"
	tempDir := uploadTempDir(root, identifier)
	require.NoError(t, os.MkdirAll(tempDir, 0o755))

	physical := filepath.Join(root, "result.txt")
	require.NoError(t, os.WriteFile(physical, []byte("data"), 0o644))
	require.NoError(t, writeUploadCompletion(tempDir, uploadCompletion{
		VirtualPath:  "/result.txt",
		PhysicalPath: physical,
		Size:         4,
	}))

	_, ok := completedUpload(root, identifier)
	assert.True(t, ok)

	require.NoError(t, os.Remove(physical))
	_, ok = completedUpload(root, identifier)
	assert.False(t, ok, "completion should be ignored once the output file is gone")
}

func TestClearUploadTemp_RemovesDir(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "upload")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "chunk_1"), []byte("x"), 0o644))

	clearUploadTemp(nested)

	_, err := os.Stat(nested)
	assert.True(t, os.IsNotExist(err))
}

func TestCleanupStaleUploadTemp(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, ".cloud_reserve", "upload_temp")

	stale := filepath.Join(base, "stale")
	fresh := filepath.Join(base, "fresh")
	require.NoError(t, os.MkdirAll(stale, 0o755))
	require.NoError(t, os.MkdirAll(fresh, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stale, "chunk_1"), []byte("old"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(fresh, "chunk_1"), []byte("new"), 0o644))

	// Backdate the stale upload beyond the TTL.
	old := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(stale, old, old))

	removed := CleanupStaleUploadTemp(root, 24*time.Hour)

	assert.Equal(t, 1, removed)
	assert.NoDirExists(t, stale)
	assert.DirExists(t, fresh)
}

func TestCleanupStaleUploadTemp_MissingBase(t *testing.T) {
	assert.Equal(t, 0, CleanupStaleUploadTemp(t.TempDir(), time.Hour))
}

func TestLockUpload_SerializesAndCleansUp(t *testing.T) {
	firstLocked := make(chan struct{})
	release := make(chan struct{})

	go func() {
		unlock := lockUpload("same-id")
		close(firstLocked)
		<-release
		unlock()
	}()

	<-firstLocked

	secondAcquired := make(chan struct{})
	go func() {
		unlock := lockUpload("same-id")
		unlock()
		close(secondAcquired)
	}()

	select {
	case <-secondAcquired:
		t.Fatal("second lock acquired while first was held")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	select {
	case <-secondAcquired:
	case <-time.After(2 * time.Second):
		t.Fatal("second lock never acquired after release")
	}

	uploadLocksMu.Lock()
	_, exists := uploadLocks["same-id"]
	uploadLocksMu.Unlock()
	assert.False(t, exists, "lock entry should be removed once refs reach zero")
}
