package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go-file-server/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRangeHeader(t *testing.T) {
	const size = int64(1000)

	cases := []struct {
		name       string
		header     string
		start, end int64
		ok         bool
	}{
		{"explicit", "bytes=0-1", 0, 1, true},
		{"open-ended", "bytes=0-", 0, 999, true},
		{"middle", "bytes=500-999", 500, 999, true},
		{"end clamped", "bytes=900-5000", 900, 999, true},
		{"suffix", "bytes=-500", 500, 999, true},
		{"suffix larger than file clamped", "bytes=-5000", 0, 999, true},
		{"suffix exactly file", "bytes=-1000", 0, 999, true},
		{"suffix zero invalid", "bytes=-0", 0, 0, false},
		{"missing prefix", "0-1", 0, 0, false},
		{"empty", "", 0, 0, false},
		{"multi-range", "bytes=0-1,5-6", 0, 0, false},
		{"start beyond size", "bytes=5000-", 0, 0, false},
		{"backwards", "bytes=500-100", 0, 0, false},
		{"garbage", "bytes=abc", 0, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end, ok := parseRangeHeader(tc.header, size)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.start, start)
				assert.Equal(t, tc.end, end)
			}
		})
	}
}

func TestServeVideo_StaleIfRangeServesFull(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "clip.mp4"), []byte("0123456789"), 0o644))

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newServeVideoRouter(cfg)

	// The client cached the old copy and sends its validator; the file changed,
	// so the range must be ignored and the whole file returned (no 416).
	req := httptest.NewRequest(http.MethodGet, "/api/user/video/play/file/clip.mp4", nil)
	req.Header.Set("Range", "bytes=0-1")
	req.Header.Set("If-Range", `"stale-etag"`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "0123456789", w.Body.String())
	assert.NotEmpty(t, w.Header().Get("ETag"))
}

func TestServeVideo_MatchingIfRangeServesPartial(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "clip.mp4"), []byte("0123456789"), 0o644))

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newServeVideoRouter(cfg)

	// Fetch the validator, then reuse it: a matching If-Range keeps the 206.
	head := httptest.NewRecorder()
	router.ServeHTTP(head, httptest.NewRequest(http.MethodGet, "/api/user/video/play/file/clip.mp4", nil))
	etag := head.Header().Get("ETag")
	require.NotEmpty(t, etag)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/play/file/clip.mp4", nil)
	req.Header.Set("Range", "bytes=0-1")
	req.Header.Set("If-Range", etag)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusPartialContent, w.Code)
	assert.Equal(t, "bytes 0-1/10", w.Header().Get("Content-Range"))
	assert.Equal(t, "01", w.Body.String())
}

func newServeVideoRouter(cfg *config.CloudConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/user/video/play/file/*filepath", func(c *gin.Context) {
		ServeVideo(c, cfg)
	})
	return router
}

func TestServeVideo_SuffixRangeServesTail(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "clip.mp4"), []byte("0123456789"), 0o644))

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newServeVideoRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/user/video/play/file/clip.mp4", nil)
	req.Header.Set("Range", "bytes=-3")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusPartialContent, w.Code)
	assert.Equal(t, "bytes 7-9/10", w.Header().Get("Content-Range"))
	assert.Equal(t, "789", w.Body.String())
}

func TestServeVideo_RangeForms(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "clip.mp4"), []byte("0123456789"), 0o644))

	cfg := &config.CloudConfig{}
	cfg.Server.FileRoot = dir
	router := newServeVideoRouter(cfg)

	cases := []struct {
		name         string
		header       string
		status       int
		contentRange string
		body         string
	}{
		{"safari probe", "bytes=0-1", http.StatusPartialContent, "bytes 0-1/10", "01"},
		{"open ended", "bytes=3-", http.StatusPartialContent, "bytes 3-9/10", "3456789"},
		{"explicit", "bytes=2-4", http.StatusPartialContent, "bytes 2-4/10", "234"},
		// Every unsatisfiable range is ignored and the whole file served (200),
		// never a 416 that wedges the player.
		{"stale beyond eof", "bytes=5000-", http.StatusOK, "", "0123456789"},
		{"multi-range", "bytes=0-1,5-6", http.StatusOK, "", "0123456789"},
		{"malformed", "bytes=abc", http.StatusOK, "", "0123456789"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/user/video/play/file/clip.mp4", nil)
			req.Header.Set("Range", tc.header)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Code)
			if tc.contentRange != "" {
				assert.Equal(t, tc.contentRange, w.Header().Get("Content-Range"))
			}
			if tc.body != "" {
				assert.Equal(t, tc.body, w.Body.String())
			}
		})
	}
}
