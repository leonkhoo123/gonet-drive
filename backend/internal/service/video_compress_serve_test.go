package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"go-file-server/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressScaleFilter(t *testing.T) {
	got := compressScaleFilter(480)
	want := "scale=w='if(gt(min(iw\\,ih)\\,480)\\,floor(iw*480/min(iw\\,ih)/2)*2\\,iw)':" +
		"h='if(gt(min(iw\\,ih)\\,480)\\,floor(ih*480/min(iw\\,ih)/2)*2\\,ih)'"
	assert.Equal(t, want, got)

	// A different cap is threaded through both dimensions.
	assert.Contains(t, compressScaleFilter(720), "720")
}

func TestParseStartParam(t *testing.T) {
	cases := []struct {
		raw  string
		want int
		ok   bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"-3", 0, false},
		{"abc", 0, false},
		{"NaN", 0, false},
		{"12", 12, true},
		{"12.9", 12, true},
		{"1e3", 1000, true},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, ok := parseStartParam(tc.raw)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseQualityParam(t *testing.T) {
	cases := []struct {
		raw      string
		fallback int
		want     int
	}{
		{"", 480, 480},
		{"1080", 480, 1080},
		{"720", 480, 720},
		{" 720 ", 480, 720},
		{"0", 480, 480},     // below minimum
		{"10000", 480, 480}, // above maximum
		{"abc", 480, 480},   // not a number
		{"-5", 480, 480},    // negative
		{"144", 480, 144},   // lower bound inclusive
		{"2160", 480, 2160}, // upper bound inclusive
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, parseQualityParam(tc.raw, tc.fallback), "raw=%q", tc.raw)
	}
}

// testCompressParams returns a representative 480p request; individual tests
// override the field they exercise.
func testCompressParams() compressParams {
	return compressParams{
		Path:      "/videos/clip.mp4",
		ShortSide: 480,
		Tier:      tierForShortSide(480),
		SourceFPS: 60,
		CRF:       20,
		Threads:   2,
		Preset:    "veryfast",
	}
}

// argValue returns the value following flag, or "" when absent.
func argValue(args []string, flag string) string {
	i := slices.Index(args, flag)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func TestTierForShortSide(t *testing.T) {
	cases := []struct {
		request int
		want    int
	}{
		{480, 480},
		{720, 720},
		{1080, 1080},
		{144, 480},   // below the smallest tier -> smallest
		{300, 480},   // below the smallest tier -> smallest
		{900, 1080},  // in-between -> next tier up
		{2160, 1080}, // above the largest -> largest
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tierForShortSide(tc.request).ShortSide, "request=%d", tc.request)
	}
}

func TestBuildCompressArgs_StartUsesInputSeek(t *testing.T) {
	p := testCompressParams()
	p.Start = 30
	args := buildCompressArgs(p)

	ssIndex := slices.Index(args, "-ss")
	iIndex := slices.Index(args, "-i")
	require.NotEqual(t, -1, ssIndex, "start>0 must add -ss")
	require.NotEqual(t, -1, iIndex)
	assert.Less(t, ssIndex, iIndex, "-ss must come before -i (input seeking)")
	assert.Equal(t, "30", args[ssIndex+1])

	// The streaming contract must be present.
	assert.Contains(t, args, compressMovFlags)
	assert.Contains(t, args, "pipe:1")
	assert.Contains(t, args, "libx264")
	assert.Contains(t, args, "aac")

	// B-frames are disabled to avoid a timestamp gap at each segment start.
	bfIndex := slices.Index(args, "-bf")
	require.NotEqual(t, -1, bfIndex, "expected -bf flag")
	assert.Equal(t, "0", args[bfIndex+1])

	// No upscaling/segmentation surprises: no -t and no output-level -ss.
	assert.NotContains(t, args, "-t")
}

func TestBuildCompressArgs_ZeroStartOmitsSeek(t *testing.T) {
	args := buildCompressArgs(testCompressParams())
	assert.NotContains(t, args, "-ss")
}

// TestBuildCompressArgs_AppliesBandwidthCeiling guards the predictable-ceiling
// contract: every compressed stream carries a maxrate/bufsize pair and a
// per-tier audio bitrate, which is what makes the declared ceiling meaningful.
func TestBuildCompressArgs_AppliesBandwidthCeiling(t *testing.T) {
	args := buildCompressArgs(testCompressParams())

	tier := tierForShortSide(480)
	assert.Equal(t, strconv.Itoa(tier.MaxRate), argValue(args, "-maxrate"))
	assert.Equal(t, strconv.Itoa(tier.BufSize), argValue(args, "-bufsize"))
	assert.Equal(t, strconv.Itoa(tier.AudioRate), argValue(args, "-b:a"))
	// bufsize is 3x maxrate: the 3s VBV window absorbs scene-cut bursts.
	assert.Equal(t, 3, tier.BufSize/tier.MaxRate)
}

// TestBuildCompressArgs_TimeBasedKeyframes guards the fps-independent keyframe
// cadence: the interval is expressed in output seconds, not frames.
func TestBuildCompressArgs_TimeBasedKeyframes(t *testing.T) {
	args := buildCompressArgs(testCompressParams())

	assert.Equal(t, "expr:gte(t,n_forced*1)", argValue(args, "-force_key_frames"))
	assert.Equal(t, strconv.Itoa(compressSceneCutThreshold), argValue(args, "-sc_threshold"))
	// 60fps output -> 0.5s floor = 30 frames.
	assert.Equal(t, "30", argValue(args, "-keyint_min"))
	// -g must be gone: a frame count would mean different intervals per fps.
	assert.NotContains(t, args, "-g")
}

// TestBuildCompressArgs_LeavesSourceFPSBelowCapUntouched checks that we do not
// force a constant rate when there is nothing to cap: re-quantizing a source at
// or below the cap can introduce judder on variable-frame-rate input.
func TestBuildCompressArgs_LeavesSourceFPSBelowCapUntouched(t *testing.T) {
	p := testCompressParams() // 480 tier caps at 60fps
	p.SourceFPS = 24
	args := buildCompressArgs(p)

	vf := argValue(args, "-vf")
	assert.NotContains(t, vf, "fps=", "no frame-rate filter below the cap")
	assert.Contains(t, vf, "scale=")
	// The keyframe floor still follows the source rate: 24fps * 0.5s = 12.
	assert.Equal(t, "12", argValue(args, "-keyint_min"))
}

// TestBuildCompressArgs_DownsamplesAboveCap checks the cap still applies when
// the source really is faster than the tier allows.
func TestBuildCompressArgs_DownsamplesAboveCap(t *testing.T) {
	p := testCompressParams()
	p.SourceFPS = 120
	args := buildCompressArgs(p)

	assert.Contains(t, argValue(args, "-vf"), "fps=60")
	assert.Equal(t, "30", argValue(args, "-keyint_min"), "60fps output * 0.5s")
}

func TestBuildCompressArgs_UnknownSourceFPSClampsInFilter(t *testing.T) {
	p := testCompressParams()
	p.SourceFPS = 0
	args := buildCompressArgs(p)

	// With no probe result the filter must clamp against source_fps instead of
	// pinning the cap, so a low-rate source is never upscaled.
	assert.Contains(t, argValue(args, "-vf"), "min(source_fps,60)")
	assert.Equal(t, "30", argValue(args, "-keyint_min"))
}

// TestBuildCompressArgs_PinsCodecProfileAndLevel guards the MSE contract: the
// frontend feeds this stream into a SourceBuffer with the codec string
// "avc1.64002a", which only matches if the encoder is pinned to High profile
// level 4.2. If these flags drift, iOS playback silently breaks.
func TestBuildCompressArgs_PinsCodecProfileAndLevel(t *testing.T) {
	args := buildCompressArgs(testCompressParams())

	profileIndex := slices.Index(args, "-profile:v")
	require.NotEqual(t, -1, profileIndex, "expected -profile:v flag")
	assert.Equal(t, "high", args[profileIndex+1])

	levelIndex := slices.Index(args, "-level:v")
	require.NotEqual(t, -1, levelIndex, "expected -level:v flag")
	assert.Equal(t, "4.2", args[levelIndex+1])
}

func TestParseFrameRate(t *testing.T) {
	cases := []struct {
		raw  string
		want float64
	}{
		{"60/1", 60},
		{"30000/1001", 29.97002997},
		{"25", 25},
		{"0/0", 0},
		{"30/0", 0},
		{"-30/1", 0},
		{"", 0},
		{"abc", 0},
	}
	for _, tc := range cases {
		assert.InDelta(t, tc.want, parseFrameRate(tc.raw), 0.001, "raw=%q", tc.raw)
	}
}

// newCompressRouter wires the two endpoints exactly like VideoRoutes so path
// params and gin behaviour match production.
func newCompressRouter(cfg *config.CloudConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/user/video/stream/file/*filepath", func(c *gin.Context) {
		ServeCompressedStream(c, cfg)
	})
	router.GET("/api/user/video/duration/file/*filepath", func(c *gin.Context) {
		GetVideoDuration(c, cfg)
	})
	return router
}

func TestServeCompressedStream_PathTraversalForbidden(t *testing.T) {
	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = t.TempDir()
	router := newCompressRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/stream/file/../secret.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestServeCompressedStream_NotFound(t *testing.T) {
	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = t.TempDir()
	router := newCompressRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/stream/file/missing.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetVideoDuration_PathTraversalForbidden(t *testing.T) {
	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = t.TempDir()
	router := newCompressRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/duration/file/../secret.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetVideoDuration_ReturnsDurationAndDimensions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	createTestMP4(t, video)

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newCompressRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/duration/file/clip.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"duration"`)
	// The fixture is a 1s 32x32 clip (createTestMP4).
	assert.Contains(t, w.Body.String(), `"width":32`)
	assert.Contains(t, w.Body.String(), `"height":32`)
}

func TestProbeVideoMeta(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	createTestMP4(t, video)

	meta, err := probeVideoMeta(context.Background(), video)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, meta.Duration, 0.5)
	assert.Equal(t, 32, meta.Width)
	assert.Equal(t, 32, meta.Height)
	// createTestMP4 encodes at rate=1.
	assert.InDelta(t, 1.0, meta.FPS, 0.1)
}

func TestGetVideoQualityTiers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/user/video/capabilities", GetVideoQualityTiers)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/capabilities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"shortSide":480`)
	assert.Contains(t, body, `"shortSide":720`)
	assert.Contains(t, body, `"shortSide":1080`)
	assert.Contains(t, body, `"maxBitrate":8000000`)
	assert.Contains(t, body, `"audioBitrate":128000`)
	// 1080p ceiling = (8_000_000 + 128_000) * 1.01.
	assert.Contains(t, body, `"ceiling":8209280`)
}

func TestServeCompressedStream_StreamsFragmentedMP4(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	createTestMP4(t, video)

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newCompressRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/stream/file/clip.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "video/mp4", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	body := w.Body.Bytes()
	require.Greater(t, len(body), 8, "stream should contain an MP4 box header")
	// Fragmented MP4 begins with an ftyp box; its type sits at bytes 4..8.
	assert.Equal(t, "ftyp", string(body[4:8]))
	assert.False(t, strings.HasPrefix(string(body), "<"), "must be binary MP4, not an error page")
}

// videoFrameTimes returns the video frame presentation timestamps (seconds) in
// ascending order.
func videoFrameTimes(t *testing.T, path string) []float64 {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "frame=pts_time",
		"-of", "csv=p=0", path,
	).Output()
	require.NoError(t, err, "ffprobe pts extraction failed")

	var times []float64
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		// CSV rows can carry trailing empty fields; take the first token.
		field := strings.TrimSpace(strings.SplitN(strings.TrimSpace(line), ",", 2)[0])
		if field == "" || field == "N/A" {
			continue
		}
		v, err := strconv.ParseFloat(field, 64)
		require.NoError(t, err, "unexpected pts_time value %q", line)
		times = append(times, v)
	}
	sort.Float64s(times)
	return times
}

// TestServeCompressedStream_NoStartupTimestampGap guards the -bf 0 setting:
// without it, fragmented MP4 + B-frame reordering introduced a ~0.1s gap at the
// start of every segment, felt as a hitch on each seek.
func TestServeCompressedStream_NoStartupTimestampGap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	out, err := exec.Command("ffmpeg",
		"-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc2=size=160x120:rate=30:duration=2",
		"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p",
		video,
	).CombinedOutput()
	require.NoError(t, err, "failed to create test video: %s", string(out))

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newCompressRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/stream/file/clip.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	streamed := filepath.Join(dir, "streamed.mp4")
	require.NoError(t, os.WriteFile(streamed, w.Body.Bytes(), 0o644))

	times := videoFrameTimes(t, streamed)
	require.Greater(t, len(times), 30, "expected a multi-frame output")

	const frameDuration = 1.0 / 30.0
	for i := 1; i < len(times); i++ {
		delta := times[i] - times[i-1]
		assert.LessOrEqual(t, delta, frameDuration*1.5,
			"timestamp gap of %.4fs between frames %d and %d", delta, i-1, i)
	}
}

func TestServeCompressedStream_RejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "folder")
	require.NoError(t, os.Mkdir(sub, 0o755))

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newCompressRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/stream/file/folder", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBuildCompressArgs_BoundsThreads(t *testing.T) {
	p := testCompressParams()
	p.Threads = 3
	args := buildCompressArgs(p)

	threadsIndex := slices.Index(args, "-threads")
	require.NotEqual(t, -1, threadsIndex, "expected -threads flag")
	assert.Equal(t, "3", args[threadsIndex+1])
	// -threads is an output option, so it must follow the input.
	assert.Greater(t, threadsIndex, slices.Index(args, "-i"))
}

func TestCompressThreads_DefaultAndOverride(t *testing.T) {
	assert.Equal(t, compressDefaultThreads, compressThreads(&config.CloudConfig{}))

	cfg := &config.CloudConfig{}
	cfg.Server.VideoCompressThreads = 1
	assert.Equal(t, 1, compressThreads(cfg))
}

func TestNewCompressedCommand_WrapsPrlimitWhenConfigured(t *testing.T) {
	args := buildCompressArgs(testCompressParams())
	ctx := context.Background()

	cfg := &config.CloudConfig{}
	cfg.Server.VideoCompressMemoryLimitMB = 512
	wrapped := newCompressedCommand(ctx, cfg, args)
	require.NotEmpty(t, wrapped.Args)
	assert.Equal(t, "prlimit", wrapped.Args[0])
	assert.Contains(t, wrapped.Args, "--as=536870912")
	assert.Contains(t, wrapped.Args, "ffmpeg")
	// The transcode args still ride along.
	assert.Contains(t, wrapped.Args, "pipe:1")

	// Disabled (0) runs ffmpeg directly with no prlimit wrapper.
	cfg.Server.VideoCompressMemoryLimitMB = 0
	plain := newCompressedCommand(ctx, cfg, args)
	require.NotEmpty(t, plain.Args)
	assert.Equal(t, "ffmpeg", plain.Args[0])
	assert.NotContains(t, plain.Args, "--as=")
}

func TestLimiter_TryAcquire(t *testing.T) {
	l := newLimiter(1, "test")

	assert.True(t, l.TryAcquire())
	assert.False(t, l.TryAcquire(), "a full limiter must not block")
	assert.Equal(t, 1, l.Acquiring())

	l.Release()
	assert.True(t, l.TryAcquire())
	l.Release()
	assert.Equal(t, 0, l.Acquiring())
}

func TestGetVideoCompressSemaphore_PicksUpConfig(t *testing.T) {
	prev := globalVideoCompressSemaphore
	prevCfg := config.AppConfig
	globalVideoCompressSemaphore = nil
	videoCompressOnce = sync.Once{}
	defer func() {
		globalVideoCompressSemaphore = prev
		videoCompressOnce = sync.Once{}
		config.AppConfig = prevCfg
	}()

	config.AppConfig = &config.CloudConfig{
		Server: config.ServerConfig{VideoCompressMaxConcurrent: 3},
	}

	sem := GetVideoCompressSemaphore()
	assert.Equal(t, 3, sem.limit, "should use VideoCompressMaxConcurrent from config")
	assert.Same(t, sem, GetVideoCompressSemaphore(), "second call must reuse the singleton")
}

// TestServeCompressedStream_BusyReturns429 verifies the client-facing contract
// when the host is already transcoding at capacity: an immediate 429 with
// Retry-After (not a queued, hung connection).
func TestServeCompressedStream_BusyReturns429(t *testing.T) {
	dir := t.TempDir()
	// Existence and non-directory is all that is needed: the semaphore gate
	// runs before ffmpeg is invoked, so the file never has to be decodable.
	video := filepath.Join(dir, "clip.mp4")
	require.NoError(t, os.WriteFile(video, []byte("stub"), 0o644))

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newCompressRouter(cfg)

	prev := globalVideoCompressSemaphore
	globalVideoCompressSemaphore = newLimiter(1, "video-compress")
	videoCompressOnce.Do(func() {}) // mark initialised so Get() returns ours
	defer func() {
		globalVideoCompressSemaphore = prev
		videoCompressOnce = sync.Once{}
	}()

	require.True(t, globalVideoCompressSemaphore.TryAcquire(), "occupy the only slot")
	defer globalVideoCompressSemaphore.Release()

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/stream/file/clip.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "5", w.Header().Get("Retry-After"))
	assert.Contains(t, w.Body.String(), "busy")
}
