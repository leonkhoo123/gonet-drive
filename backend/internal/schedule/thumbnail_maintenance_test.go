package schedule

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/state"
	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsVideoFile(t *testing.T) {
	assert.True(t, isVideoFile("test.mp4"))
	assert.True(t, isVideoFile("test.MKV"))
	assert.True(t, isVideoFile("video.avi"))
	assert.True(t, isVideoFile("movie.mov"))
	assert.True(t, isVideoFile("clip.webm"))
	assert.False(t, isVideoFile("photo.jpg"))
	assert.False(t, isVideoFile("song.mp3"))
	assert.False(t, isVideoFile("readme.txt"))
}

func TestIsPhotoFile(t *testing.T) {
	assert.True(t, isPhotoFile("test.jpg"))
	assert.True(t, isPhotoFile("test.PNG"))
	assert.True(t, isPhotoFile("photo.heic"))
	assert.False(t, isPhotoFile("video.mp4"))
	assert.False(t, isPhotoFile("song.mp3"))
}

func TestGeneratePhotoThumbnail(t *testing.T) {
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "test.png")
	createTestPNG(t, srcPath, 800, 600)

	thumbPath := filepath.Join(dir, "thumb.webp")
	err := service.GeneratePhotoThumbnail(context.Background(), srcPath, thumbPath)
	require.NoError(t, err)

	data, err := os.ReadFile(thumbPath)
	require.NoError(t, err)
	assert.True(t, len(data) > 0)
	assert.True(t, len(data) >= 4 && string(data[0:4]) == "RIFF",
		"thumbnail should be a valid WebP file (RIFF header)")
}

func TestGeneratePhotoThumbnail_JPEG(t *testing.T) {
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "test.jpg")
	createTestJPEG(t, srcPath, 800, 600)

	thumbPath := filepath.Join(dir, "thumb.webp")
	err := service.GeneratePhotoThumbnail(context.Background(), srcPath, thumbPath)
	require.NoError(t, err)

	data, err := os.ReadFile(thumbPath)
	require.NoError(t, err)
	assert.True(t, len(data) > 0)
	assert.True(t, len(data) >= 4 && string(data[0:4]) == "RIFF",
		"JPEG thumbnail should be a valid WebP file")
}

func TestGeneratePhotoThumbnail_GIF(t *testing.T) {
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "test.gif")
	createTestGIF(t, srcPath, 800, 600)

	thumbPath := filepath.Join(dir, "thumb.webp")
	err := service.GeneratePhotoThumbnail(context.Background(), srcPath, thumbPath)
	require.NoError(t, err)

	data, err := os.ReadFile(thumbPath)
	require.NoError(t, err)
	assert.True(t, len(data) > 0)
	assert.True(t, len(data) >= 4 && string(data[0:4]) == "RIFF",
		"GIF thumbnail should be a valid WebP file")
}

func TestGeneratePhotoThumbnail_InvalidFile(t *testing.T) {
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "corrupt.png")
	require.NoError(t, os.WriteFile(srcPath, []byte("not an image"), 0644))

	thumbPath := filepath.Join(dir, "thumb.webp")
	err := service.GeneratePhotoThumbnail(context.Background(), srcPath, thumbPath)
	assert.Error(t, err, "should fail on corrupt/invalid image file")
}

func TestGenerateVideoThumbnail_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "fake.mp4")
	require.NoError(t, os.WriteFile(srcPath, []byte("not a video"), 0644))

	thumbPath := filepath.Join(dir, "thumb.webp")
	err := service.GenerateVideoThumbnail(context.Background(), srcPath, thumbPath)
	assert.Error(t, err)
}

func TestRepoUpsertAndMarkInactive(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	require.NoError(t, repo.Upsert("abc123", "/data/photo.jpg", false))
	require.NoError(t, repo.Upsert("def456", "/data/video.mp4", true))

	require.NoError(t, repo.MarkAllInactive())

	hashes, err := repo.DeleteInactive()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"abc123", "def456"}, hashes)

	// Re-inserted after mark should survive
	require.NoError(t, repo.Upsert("keep1", "/data/img.png", false))

	hashes, err = repo.DeleteInactive()
	require.NoError(t, err)
	assert.Empty(t, hashes)
}

func TestRepoDeleteInactive_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	hashes, err := repo.DeleteInactive()
	require.NoError(t, err)
	assert.Empty(t, hashes)
}

func TestMaintainThumbnails_OrphanCleanup(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	rootDir := t.TempDir()
	thumbDir := filepath.Join(rootDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	filePath := filepath.Join(rootDir, "photo.jpg")
	createTestPNGFile(t, filePath)

	// orphan thumbnail on disk (source file does not exist)
	orphanThumb := filepath.Join(thumbDir, "deadbeef.webp")
	require.NoError(t, os.WriteFile(orphanThumb, []byte("orphan"), 0644))
	// also in DB so DeleteInactive finds it
	require.NoError(t, repo.Upsert("deadbeef", "/nonexistent.jpg", false))

	deleted, generated, err := maintainThumbnails(rootDir, repo)
	require.NoError(t, err)

	assert.Equal(t, 1, deleted)
	assert.Equal(t, 1, generated)

	_, err = os.Stat(orphanThumb)
	assert.True(t, os.IsNotExist(err))

	entries, err := os.ReadDir(thumbDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(entries))
}

func TestMaintainThumbnails_SkipsFreshThumbnails(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	rootDir := t.TempDir()
	thumbDir := filepath.Join(rootDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	filePath := filepath.Join(rootDir, "photo.png")
	createTestPNGFile(t, filePath)
	fileStat, err := os.Stat(filePath)
	require.NoError(t, err)

	hash := thumbnailHash(filePath)
	thumbPath := filepath.Join(thumbDir, hash+".webp")
	require.NoError(t, os.WriteFile(thumbPath, []byte("cached"), 0644))
	futureTime := fileStat.ModTime().Add(1 * time.Hour)
	require.NoError(t, os.Chtimes(thumbPath, futureTime, futureTime))

	deleted, generated, err := maintainThumbnails(rootDir, repo)
	require.NoError(t, err)

	assert.Equal(t, 0, deleted)
	assert.Equal(t, 0, generated)

	// fresh thumbnail should be upserted active, so nothing inactive
	hashes, err := repo.DeleteInactive()
	require.NoError(t, err)
	assert.Empty(t, hashes)
}

func TestMaintainThumbnails_SkipsNonMediaFiles(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	rootDir := t.TempDir()
	thumbDir := filepath.Join(rootDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "readme.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "song.mp3"), []byte("music"), 0644))

	deleted, generated, err := maintainThumbnails(rootDir, repo)
	require.NoError(t, err)

	assert.Equal(t, 0, deleted)
	assert.Equal(t, 0, generated)
}

func TestMaintainThumbnails_SkipsCloudReserve(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	rootDir := t.TempDir()
	thumbDir := filepath.Join(rootDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	internalFile := filepath.Join(rootDir, ".cloud_reserve", "logo.png")
	createTestPNGFile(t, internalFile)

	deleted, generated, err := maintainThumbnails(rootDir, repo)
	require.NoError(t, err)

	assert.Equal(t, 0, deleted)
	assert.Equal(t, 0, generated)
}

func TestMaintainThumbnails_NoThumbDir(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	rootDir := t.TempDir()

	deleted, generated, err := maintainThumbnails(rootDir, repo)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted)
	assert.Equal(t, 0, generated)
}

func TestMaintainThumbnails_StaleThumbnailReplaced(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	rootDir := t.TempDir()
	thumbDir := filepath.Join(rootDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	filePath := filepath.Join(rootDir, "photo.png")
	createTestPNGFile(t, filePath)
	fileStat, err := os.Stat(filePath)
	require.NoError(t, err)

	hash := thumbnailHash(filePath)
	thumbPath := filepath.Join(thumbDir, hash+".webp")
	require.NoError(t, os.WriteFile(thumbPath, []byte("old stale"), 0644))
	oldTime := fileStat.ModTime().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(thumbPath, oldTime, oldTime))

	deleted, generated, err := maintainThumbnails(rootDir, repo)
	require.NoError(t, err)

	assert.Equal(t, 0, deleted)
	assert.Equal(t, 1, generated)

	data, err := os.ReadFile(thumbPath)
	require.NoError(t, err)
	assert.NotEqual(t, "old stale", string(data))
}

// TestMaintainThumbnails_CoversRealtimeThumbnail — thumbnail created at
// runtime after MarkAllInactive. The walk re-encounters it in cache and
// upserts it with active=true, so it survives cleanup.
func TestMaintainThumbnails_CoversRealtimeThumbnail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	rootDir := t.TempDir()
	thumbDir := filepath.Join(rootDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	f1 := filepath.Join(rootDir, "pic1.jpg")
	createTestPNGFile(t, f1)
	f2 := filepath.Join(rootDir, "pic2.jpg")
	createTestPNGFile(t, f2)

	hash1 := thumbnailHash(f1)
	thumb1 := filepath.Join(thumbDir, hash1+".webp")
	f1Stat, _ := os.Stat(f1)
	require.NoError(t, os.WriteFile(thumb1, []byte("cached"), 0644))
	require.NoError(t, os.Chtimes(thumb1, f1Stat.ModTime().Add(1*time.Hour), f1Stat.ModTime().Add(1*time.Hour)))

	require.NoError(t, repo.Upsert(hash1, f1, false))
	hash2 := thumbnailHash(f2)
	require.NoError(t, repo.Upsert(hash2, f2, false))

	// Simulate runtime creation for f2's thumbnail
	thumb2 := filepath.Join(thumbDir, hash2+".webp")
	f2Stat, _ := os.Stat(f2)
	require.NoError(t, os.WriteFile(thumb2, []byte("runtime created"), 0644))
	require.NoError(t, os.Chtimes(thumb2, f2Stat.ModTime().Add(1*time.Hour), f2Stat.ModTime().Add(1*time.Hour)))

	deleted, generated, err := maintainThumbnails(rootDir, repo)
	require.NoError(t, err)

	assert.Equal(t, 0, deleted)
	assert.Equal(t, 0, generated)

	hashes, err := repo.DeleteInactive()
	require.NoError(t, err)
	assert.Empty(t, hashes)
}

func TestMaintainThumbnails_RenamedFileCleansOldThumbnail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	rootDir := t.TempDir()
	thumbDir := filepath.Join(rootDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	// Old file with cached thumbnail
	oldFile := filepath.Join(rootDir, "photo.png")
	createTestPNGFile(t, oldFile)
	fStat, _ := os.Stat(oldFile)
	oldHash := thumbnailHash(oldFile)
	oldThumb := filepath.Join(thumbDir, oldHash+".webp")
	require.NoError(t, os.WriteFile(oldThumb, []byte("old"), 0644))
	require.NoError(t, os.Chtimes(oldThumb, fStat.ModTime().Add(1*time.Hour), fStat.ModTime().Add(1*time.Hour)))
	require.NoError(t, repo.Upsert(oldHash, oldFile, false))

	// "Rename" — remove old, create new with different path/hash
	require.NoError(t, os.Remove(oldFile))
	newFile := filepath.Join(rootDir, "renamed.png")
	createTestPNGFile(t, newFile)

	deleted, generated, err := maintainThumbnails(rootDir, repo)
	require.NoError(t, err)

	assert.Equal(t, 1, deleted)
	assert.Equal(t, 1, generated)

	_, err = os.Stat(oldThumb)
	assert.True(t, os.IsNotExist(err))
}

func TestPreGenerateThumbnails_PausesForAPI(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteThumbnailRepo(db)

	dir := t.TempDir()
	thumbDir := filepath.Join(dir, ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	f1 := filepath.Join(dir, "pic1.png")
	createTestPNGFile(t, f1)
	f2 := filepath.Join(dir, "pic2.png")
	createTestPNGFile(t, f2)

	h1 := thumbnailHash(f1)
	tp1 := filepath.Join(thumbDir, h1+".webp")
	h2 := thumbnailHash(f2)
	tp2 := filepath.Join(thumbDir, h2+".webp")

	pending := []fileNeedingThumbnail{
		{fullPath: f1, thumbPath: tp1, hash: h1, isVideo: false},
		{fullPath: f2, thumbPath: tp2, hash: h2, isVideo: false},
	}

	// Simulate recent API activity so first idle check fails
	state.ResetLastAPIRequest()
	state.RecordAPIRequest()

	done := make(chan int, 1)
	go func() {
		done <- preGenerateThumbnails(pending, repo, 500*time.Millisecond, 50*time.Millisecond)
	}()

	// Should be paused — wait a bit and verify not done
	select {
	case <-done:
		t.Fatal("preGenerateThumbnails completed too fast — did not pause for API activity")
	case <-time.After(300 * time.Millisecond):
	}

	// Reset activity so it becomes idle and resumes
	state.ResetLastAPIRequest()

	select {
	case generated := <-done:
		assert.Equal(t, 2, generated)
	case <-time.After(5 * time.Second):
		t.Fatal("preGenerateThumbnails did not resume after idle")
	}

	stat1, err := os.Stat(tp1)
	require.NoError(t, err)
	assert.True(t, stat1.Size() > 0)

	stat2, err := os.Stat(tp2)
	require.NoError(t, err)
	assert.True(t, stat2.Size() > 0)
}

// --- helpers ---

func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, png.Encode(f, img))
}

func createTestPNGFile(t *testing.T, path string) {
	t.Helper()
	createTestPNG(t, path, 100, 100)
}

func createTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, jpeg.Encode(f, img, &jpeg.Options{Quality: 90}))
}

func createTestGIF(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{
		color.RGBA{0, 0, 0, 255},
		color.RGBA{255, 255, 255, 255},
		color.RGBA{255, 0, 0, 255},
	})
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, gif.Encode(f, img, nil))
}

func thumbnailHash(path string) string {
	h := md5.Sum([]byte(path))
	return hex.EncodeToString(h[:])
}
