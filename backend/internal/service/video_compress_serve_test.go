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

func TestBuildCompressArgs_StartUsesInputSeek(t *testing.T) {
	args := buildCompressArgs("/videos/clip.mp4", 30, 480, 20, "veryfast")

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
	args := buildCompressArgs("/videos/clip.mp4", 0, 480, 20, "veryfast")
	assert.NotContains(t, args, "-ss")
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
