package service

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-file-server/database"
	"go-file-server/internal/repository"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_busy_timeout=5000")
	require.NoError(t, err)
	require.NoError(t, database.RunMigrations(db))
	t.Cleanup(func() { db.Close() })
	return db
}

func createTestMP4(t *testing.T, path string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=32x32:rate=1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-an",
		"-y", path,
	)
	require.NoError(t, cmd.Run(), "ffmpeg must be available to create test fixtures")
}

func TestProbeVideoStream_ValidFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "test.mp4")
	createTestMP4(t, videoPath)

	mime, codec, err := probeVideoStream(videoPath)
	require.NoError(t, err)
	assert.Equal(t, "h264", codec)
	assert.True(t, strings.HasPrefix(mime, "avc1."), "mime codec string should start with avc1.: got %q", mime)
	assert.False(t, strings.HasPrefix(mime, "avc1.00"), "valid file should NOT start with avc1.00")
}

func TestProbeVideoStream_NotFound(t *testing.T) {
	_, _, err := probeVideoStream("/nonexistent/file.mp4")
	assert.Error(t, err)
}

func TestProbeVideoStream_NotVideo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notavideo.mp4")
	require.NoError(t, os.WriteFile(path, []byte("not a video"), 0644))

	_, _, err := probeVideoStream(path)
	assert.Error(t, err)
}

func TestIsVideoIntegrityScanRunning_InitiallyFalse(t *testing.T) {
	scanRunning.Store(false)
	assert.False(t, IsVideoIntegrityScanRunning())
}

func TestIsVideoIntegrityScanRunning_Gate(t *testing.T) {
	scanRunning.Store(false)

	assert.True(t, scanRunning.CompareAndSwap(false, true))
	assert.True(t, IsVideoIntegrityScanRunning())

	assert.False(t, scanRunning.CompareAndSwap(false, true))

	scanRunning.Store(false)
	assert.False(t, IsVideoIntegrityScanRunning())
}

func TestGetScanStatus_RepoNotInitialized(t *testing.T) {
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = nil
	defer func() { videoIntegrityRepo = prevRepo }()

	_, _, _, err := GetScanStatus()
	assert.Error(t, err)
}

func TestGetScanStatus_EmptyDB(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	scanRunning.Store(false)
	defer func() { videoIntegrityRepo = prevRepo }()

	count, lastScan, running, err := GetScanStatus()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Nil(t, lastScan)
	assert.False(t, running)
}

func TestGetScanStatus_WithEntries(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	require.NoError(t, repo.Upsert("abc", "/v.mp4", "corrupt_avcC", "avc1.000032"))
	require.NoError(t, repo.Upsert("def", "/w.mov", "corrupt_avcC", "avc1.000033"))

	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	scanRunning.Store(false)
	defer func() { videoIntegrityRepo = prevRepo }()

	count, lastScan, running, err := GetScanStatus()
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.NotNil(t, lastScan)
	assert.False(t, running)
}

func TestScanVideoIntegrity_EmptyDir(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	scanRunning.Store(false)
	defer func() {
		videoIntegrityRepo = prevRepo
		scanRunning.Store(false)
	}()

	dir := t.TempDir()
	result, err := ScanVideoIntegrity(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalScanned)
	assert.Equal(t, 0, result.CorruptCount)
	assert.Greater(t, result.EndTime, int64(0))
}

func TestScanVideoIntegrity_NoVideoFiles(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	scanRunning.Store(false)
	defer func() {
		videoIntegrityRepo = prevRepo
		scanRunning.Store(false)
	}()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("photo"), 0644))

	result, err := ScanVideoIntegrity(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalScanned)
	assert.Equal(t, 0, result.CorruptCount)
}

func TestScanVideoIntegrity_ValidVideoNotFlagged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	scanRunning.Store(false)
	defer func() {
		videoIntegrityRepo = prevRepo
		scanRunning.Store(false)
	}()

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "video.mp4")
	createTestMP4(t, videoPath)

	result, err := ScanVideoIntegrity(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalScanned)
	assert.Equal(t, 0, result.CorruptCount)

	entries, err := repo.All()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestScanVideoIntegrity_AlreadyRunning(t *testing.T) {
	scanRunning.Store(true)
	defer scanRunning.Store(false)

	_, err := ScanVideoIntegrity("/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestScanVideoIntegrity_SkipsCloudReserve(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	scanRunning.Store(false)
	defer func() {
		videoIntegrityRepo = prevRepo
		scanRunning.Store(false)
	}()

	dir := t.TempDir()
	cloudDir := filepath.Join(dir, ".cloud_reserve")
	require.NoError(t, os.MkdirAll(cloudDir, 0755))
	createTestMP4(t, filepath.Join(cloudDir, "hidden.mp4"))

	videoPath := filepath.Join(dir, "visible.mp4")
	createTestMP4(t, videoPath)

	result, err := ScanVideoIntegrity(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalScanned)
	assert.Equal(t, 0, result.CorruptCount)
}

func TestScanVideoIntegrity_OnlyMP4AndMOV(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mkv"), []byte("dummy"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "video.avi"), []byte("dummy"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "video.webm"), []byte("dummy"), 0644))

	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	scanRunning.Store(false)
	defer func() {
		videoIntegrityRepo = prevRepo
		scanRunning.Store(false)
	}()

	result, err := ScanVideoIntegrity(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalScanned)
}

func TestScanVideoIntegrity_RepoNotInitialized(t *testing.T) {
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = nil
	scanRunning.Store(false)
	defer func() {
		videoIntegrityRepo = prevRepo
		scanRunning.Store(false)
	}()

	_, err := ScanVideoIntegrity("/tmp")
	assert.Error(t, err)
}

func TestRequestScanStop_NotRunning(t *testing.T) {
	scanRunning.Store(false)
	scanStop.Store(false)

	assert.False(t, RequestScanStop())
	assert.False(t, scanStop.Load())
}

func TestRequestScanStop_Running(t *testing.T) {
	scanRunning.Store(true)
	scanStop.Store(false)
	defer func() {
		scanRunning.Store(false)
		scanStop.Store(false)
	}()

	assert.True(t, RequestScanStop())
	assert.True(t, scanStop.Load())
}

func TestScanVideoIntegrity_StopsMidScan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	scanRunning.Store(false)
	scanStop.Store(false)
	defer func() {
		videoIntegrityRepo = prevRepo
		scanRunning.Store(false)
		scanStop.Store(false)
	}()

	dir := t.TempDir()
	// Create several MP4 files so the scan runs long enough to be stopped mid-way
	for i := range 5 {
		createTestMP4(t, filepath.Join(dir, fmt.Sprintf("video%d.mp4", i)))
		_ = i
	}

	// Start scan in goroutine, then request stop after a short delay
	var result *ScanResult
	var scanErr error
	done := make(chan struct{})
	go func() {
		result, scanErr = ScanVideoIntegrity(dir)
		close(done)
	}()

	// Wait for the scan to process at least one file, then stop
	time.Sleep(1 * time.Second)
	assert.True(t, scanRunning.Load())
	RequestScanStop()

	<-done
	assert.NoError(t, scanErr)
	require.NotNil(t, result)
	// With 5 files and stop requested after ~1s, we should not have scanned all 5
	assert.Less(t, result.TotalScanned, 5)
}

func TestScanVideoIntegrity_StopFlagResetOnStart(t *testing.T) {
	scanRunning.Store(false)
	scanStop.Store(true)
	defer func() {
		scanRunning.Store(false)
		scanStop.Store(false)
	}()

	// Even with scanStop set, a new scan resets it at start
	db := openTestDB(t)
	repo := repository.NewSQLiteVideoIntegrityRepo(db)
	prevRepo := videoIntegrityRepo
	videoIntegrityRepo = repo
	defer func() { videoIntegrityRepo = prevRepo }()

	dir := t.TempDir()
	// No video files, so scan completes instantly
	result, err := ScanVideoIntegrity(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalScanned)
	// scanStop should still be false after completion
	assert.False(t, scanStop.Load())
}
