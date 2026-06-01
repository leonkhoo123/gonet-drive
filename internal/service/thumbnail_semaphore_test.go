package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go-file-server/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewThumbnailSemaphore_Defaults(t *testing.T) {
	s := newThumbnailSemaphore(2)
	assert.Equal(t, 2, s.limit)
	assert.Equal(t, 2, s.Available())
	assert.Equal(t, 0, s.Acquiring())
}

func TestNewThumbnailSemaphore_ClampsToMinimum(t *testing.T) {
	s := newThumbnailSemaphore(0)
	assert.Equal(t, 1, s.limit)
}

func TestThumbnailSemaphore_AcquireRelease(t *testing.T) {
	s := newThumbnailSemaphore(2)

	ctx := context.Background()
	require.NoError(t, s.Acquire(ctx))
	assert.Equal(t, 1, s.Available())
	assert.Equal(t, 1, s.Acquiring())

	require.NoError(t, s.Acquire(ctx))
	assert.Equal(t, 0, s.Available())
	assert.Equal(t, 2, s.Acquiring())

	s.Release()
	assert.Equal(t, 1, s.Available())

	require.NoError(t, s.Acquire(ctx))
	assert.Equal(t, 0, s.Available())

	s.Release()
	s.Release()
	s.Release()
	assert.Equal(t, 2, s.Available())
	assert.Equal(t, 0, s.Acquiring())
}

func TestThumbnailSemaphore_ConcurrencyLimit(t *testing.T) {
	s := newThumbnailSemaphore(2)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var maxSeen int

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, s.Acquire(ctx))
			mu.Lock()
			current := s.Acquiring()
			if current > maxSeen {
				maxSeen = current
			}
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			s.Release()
		}()
	}

	wg.Wait()
	assert.Equal(t, 2, maxSeen, "max concurrent should never exceed limit")
	assert.Equal(t, 2, s.Available())
	assert.Equal(t, 0, s.Acquiring())
}

func TestThumbnailSemaphore_ContextTimeout(t *testing.T) {
	s := newThumbnailSemaphore(1)
	require.NoError(t, s.Acquire(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := s.Acquire(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline")
}

func TestThumbnailSemaphore_ContextCancelled(t *testing.T) {
	s := newThumbnailSemaphore(1)
	require.NoError(t, s.Acquire(context.Background()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Acquire(ctx)
	assert.Error(t, err)
}

func TestThumbnailSemaphore_LimitNotExceeded(t *testing.T) {
	s := newThumbnailSemaphore(3)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, s.Acquire(ctx))
			assert.LessOrEqual(t, s.Acquiring(), 3, "limit exceeded during execution")
			time.Sleep(10 * time.Millisecond)
			s.Release()
		}()
	}

	wg.Wait()
	assert.Equal(t, 3, s.Available())
}

func TestGetThumbnailSemaphore_Singleton(t *testing.T) {
	// Save and restore global state to avoid interference with other tests
	prev := globalThumbnailSemaphore
	globalThumbnailSemaphore = nil
	semaphoreOnce = sync.Once{}
	defer func() {
		globalThumbnailSemaphore = prev
		semaphoreOnce = sync.Once{}
	}()

	// Call from multiple goroutines concurrently — all must return the same pointer
	var wg sync.WaitGroup
	results := make([]*thumbnailSemaphore, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = GetThumbnailSemaphore()
		}(i)
	}
	wg.Wait()

	first := results[0]
	require.NotNil(t, first, "first semaphore should not be nil")
	for i := 1; i < 20; i++ {
		assert.Same(t, first, results[i], "GetThumbnailSemaphore must return the same singleton instance across concurrent calls (goroutine %d)", i)
	}
}

func TestGetThumbnailSemaphore_PicksUpConfig(t *testing.T) {
	// Save and restore global state
	prev := globalThumbnailSemaphore
	prevCfg := config.AppConfig
	globalThumbnailSemaphore = nil
	semaphoreOnce = sync.Once{}
	defer func() {
		globalThumbnailSemaphore = prev
		semaphoreOnce = sync.Once{}
		config.AppConfig = prevCfg
	}()

	config.AppConfig = &config.CloudConfig{
		Server: config.ServerConfig{
			ThumbnailMaxConcurrent: 7,
		},
	}

	sem := GetThumbnailSemaphore()
	assert.Equal(t, 7, sem.limit, "should use ThumbnailMaxConcurrent from config")

	// Second call should return same instance with same limit
	sem2 := GetThumbnailSemaphore()
	assert.Same(t, sem, sem2, "second call must return the same instance")
	assert.Equal(t, 7, sem2.limit, "limit should not change on second call")
}

func TestServeVideoThumbnail_WithSemaphore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := testConfig(t)
	cfg.Server.ThumbnailMaxConcurrent = 1
	cfg.Server.ThumbnailGenerationTimeout = 30 * time.Second

	workDir := t.TempDir()
	cfg.Server.FileRoot = workDir
	require.NoError(t, os.MkdirAll(workDir, 0755))

	videoPath := filepath.Join(workDir, "test.mp4")
	makeTestVideo(t, videoPath)

	thumbDir := filepath.Join(workDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	sem := newThumbnailSemaphore(1)

	// Simulate 5 distinct thumbnails competing for 1 slot
	var wg sync.WaitGroup
	results := make([]bool, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			hasher := md5.New()
			hasher.Write([]byte(videoPath + "_" + string(rune('A'+idx))))
			hashStr := hex.EncodeToString(hasher.Sum(nil))
			thumbPath := filepath.Join(thumbDir, hashStr+".webp")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := sem.Acquire(ctx); err != nil {
				return
			}

			cmd := exec.CommandContext(ctx,
				"ffmpeg",
				"-loglevel", "error",
				"-threads", "1",
				"-ss", "00:00:00.000",
				"-i", videoPath,
				"-an",
				"-vframes", "1",
				"-vf", "scale='min(300,iw)':'min(300,ih)':force_original_aspect_ratio=decrease",
				"-c:v", "libwebp",
				"-y",
				thumbPath)
			results[idx] = cmd.Run() == nil
			os.Remove(thumbPath)
			sem.Release()
		}(i)
	}

	wg.Wait()

	successCount := 0
	for _, r := range results {
		if r {
			successCount++
		}
	}
	t.Logf("%d out of 5 thumbnail generations succeeded (limit=1)", successCount)
	assert.GreaterOrEqual(t, successCount, 1, "at least one should succeed")
}

func TestVideoThumbnail_SingleflightStillWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := testConfig(t)
	cfg.Server.ThumbnailMaxConcurrent = 5
	cfg.Server.ThumbnailGenerationTimeout = 30 * time.Second

	workDir := t.TempDir()
	cfg.Server.FileRoot = workDir
	require.NoError(t, os.MkdirAll(workDir, 0755))

	videoPath := filepath.Join(workDir, "test_singleflight.mp4")
	makeTestVideo(t, videoPath)

	thumbDir := filepath.Join(workDir, ".cloud_reserve", ".thumbnails")
	require.NoError(t, os.MkdirAll(thumbDir, 0755))

	hasher := md5.New()
	hasher.Write([]byte(videoPath))
	hashStr := hex.EncodeToString(hasher.Sum(nil))
	thumbPath := filepath.Join(thumbDir, hashStr+".webp")
	defer os.Remove(thumbPath)

	var wg sync.WaitGroup
	errs := make([]error, 5)
	start := time.Now()

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx], _ = videoThumbnailGroup.Do(thumbPath, func() (interface{}, error) {
				ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ThumbnailGenerationTimeout)
				defer cancel()

				sem := GetThumbnailSemaphore()
				if err := sem.Acquire(ctx); err != nil {
					return nil, err
				}
				defer sem.Release()

				cmd := exec.CommandContext(ctx,
					"ffmpeg",
					"-loglevel", "error",
					"-threads", "1",
					"-ss", "00:00:00.000",
					"-i", videoPath,
					"-an",
					"-vframes", "1",
					"-vf", "scale='min(300,iw)':'min(300,ih)':force_original_aspect_ratio=decrease",
					"-c:v", "libwebp",
					"-y",
					thumbPath)
				return nil, cmd.Run()
			})
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	t.Logf("5 concurrent singleflight calls completed in %v", duration)

	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d: %v", i, err)
	}

	_, err := os.Stat(thumbPath)
	assert.NoError(t, err, "thumbnail should exist after generation")
}

// testConfig creates a minimal config for testing without importing testutil
func testConfig(t *testing.T) *config.CloudConfig {
	t.Helper()
	return &config.CloudConfig{
		Server: config.ServerConfig{
			AppEnv:                     "local",
			FileRoot:                   t.TempDir(),
			ListenAddr:                 ":0",
			AllowedOrigins:             []string{"*"},
			ThumbnailMaxConcurrent:     2,
			ThumbnailGenerationTimeout: 30 * time.Second,
		},
		Auth: config.AuthConfig{
			AppJwt:             "ON",
			JwtSecret:          "test-secret-key-for-testing-only",
			TokenName:          "file_server_token",
			CookieAccessToken:  "access_token",
			CookieRefreshToken: "refresh_token",
			CookieMfaPending:   "mfa_pending",
			CookieShareJwt:     "shareJwt",
			AccessTokenMaxAge:  15 * time.Minute,
			RefreshTokenMaxAge: 7 * 24 * time.Hour,
			MfaPendingMaxAge:   5 * time.Minute,
			ShareJwtMaxAge:     7 * 24 * time.Hour,
		},
		Defaults: config.AppDefaults{
			ServiceName:     "Test",
			UploadChunkSize: "5",
			StorageLimit:    "20480",
		},
	}
}

func makeTestVideo(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "color=c=blue:s=320x240:d=1",
		"-frames:v", "1",
		"-y",
		path)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to create test video: %s", string(out))
}
