package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

var videoThumbnailGroup singleflight.Group

// ServeVideo serves video files with HTTP range support.
// @Summary      Serve Video
// @Description  Stream a video file with HTTP range support.
// @Tags         Media
// @Produce      octet-stream
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true  "Relative file path"
// @Success      200  {file}  binary
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/video/play/file/{filepath} [get]
func ServeVideo(c *gin.Context, cfg *config.CloudConfig) {
	relPath := c.Param("filepath")
	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, relPath)
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	file, err := os.Open(fullPath)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		start, end, ok := parseRangeHeader(rangeHeader, stat.Size())
		if ok {
			chunkSize := (end - start) + 1

			c.Status(http.StatusPartialContent)
			c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, stat.Size()))
			c.Header("Accept-Ranges", "bytes")
			c.Header("Content-Length", strconv.FormatInt(chunkSize, 10))
			c.Header("Content-Type", util.MimeType(fullPath))

			// Stream directly from file descriptor (no full buffering)
			if _, err := file.Seek(start, io.SeekStart); err == nil {
				io.CopyN(c.Writer, file, chunkSize)
			}
			return
		}
		// Unparseable or unsatisfiable range: delegate to c.File, which
		// re-parses the header and answers 416 with `Content-Range: bytes */size`.
	}

	// no Range header → stream entire file
	c.Header("Accept-Ranges", "bytes")
	c.File(fullPath)
}

// parseRangeHeader parses a single-range `bytes=` header (RFC 7233) against a
// known file size. It supports the three forms browsers actually send:
//
//	bytes=<start>-<end>   explicit range
//	bytes=<start>-        open-ended range to EOF
//	bytes=-<suffix>       last N bytes (e.g. Safari reading a trailing moov)
//
// ok is false when the header is absent, malformed, multi-range, or
// unsatisfiable; the caller then serves the whole file (a 200 response to a
// Range request is valid when the server chooses not to honour it).
func parseRangeHeader(header string, size int64) (start, end int64, ok bool) {
	const prefix = "bytes="
	if size <= 0 || !strings.HasPrefix(header, prefix) {
		return 0, 0, false
	}

	spec := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	// Only single ranges are supported; multi-range requests are ignored.
	if spec == "" || strings.Contains(spec, ",") {
		return 0, 0, false
	}

	dash := strings.Index(spec, "-")
	if dash < 0 {
		return 0, 0, false
	}
	startStr := strings.TrimSpace(spec[:dash])
	endStr := strings.TrimSpace(spec[dash+1:])

	switch {
	case startStr == "" && endStr == "":
		return 0, 0, false

	case startStr == "":
		// Suffix range: the final N bytes.
		n, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || n <= 0 {
			return 0, 0, false
		}
		if n > size {
			n = size
		}
		start, end = size-n, size-1

	default:
		s, err := strconv.ParseInt(startStr, 10, 64)
		if err != nil || s < 0 || s > size-1 {
			return 0, 0, false
		}
		start = s
		if endStr == "" {
			end = size - 1
		} else {
			e, err := strconv.ParseInt(endStr, 10, 64)
			if err != nil || e < s {
				return 0, 0, false
			}
			if e > size-1 {
				e = size - 1
			}
			end = e
		}
	}

	if start < 0 {
		start = 0
	}
	if end > size-1 {
		end = size - 1
	}
	if start > end {
		return 0, 0, false
	}
	return start, end, true
}

// ServeVideoThumbnail serves a generated thumbnail for a video file
// ServeVideoThumbnail generates and serves a video thumbnail.
// @Summary      Video Thumbnail
// @Description  Generate and serve a thumbnail image for a video file.
// @Tags         Media
// @Produce      image/jpeg
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true  "Relative file path"
// @Success      200  {file}  binary
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/video/thumbnail/file/{filepath} [get]
func ServeVideoThumbnail(c *gin.Context, cfg *config.CloudConfig) {
	relPath := c.Param("filepath")
	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, relPath)
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	stat, err := os.Stat(fullPath)
	if err != nil || stat.IsDir() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Ensure .cloud_reserve/.thumbnails directory exists
	thumbnailsDir := filepath.Join(cfg.Server.FileRoot, ".cloud_reserve", ".thumbnails")
	if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Create hash for filename based on full path
	// This uniquely links the thumbnail to the original file path
	hasher := md5.New()
	hasher.Write([]byte(fullPath))
	hashStr := hex.EncodeToString(hasher.Sum(nil))
	thumbPath := filepath.Join(thumbnailsDir, hashStr+".webp")

	// Check if thumbnail exists and is newer than the original file
	thumbStat, err := os.Stat(thumbPath)
	if err == nil && thumbStat.ModTime().After(stat.ModTime()) {
		c.File(thumbPath)
		return
	}

	// Use singleflight to prevent multiple requests from generating the same thumbnail concurrently
	_, err, _ = videoThumbnailGroup.Do(thumbPath, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Server.ThumbnailGenerationTimeout)
		defer cancel()

		sem := GetThumbnailSemaphore()
		if err := sem.Acquire(ctx); err != nil {
			return nil, err
		}
		defer sem.Release()

		if err := GenerateVideoThumbnail(ctx, fullPath, thumbPath); err != nil {
			return nil, err
		}

		UpsertThumbnailRecord(hashStr, fullPath, true)
		return nil, nil
	})

	if err != nil {
		// Check if error is due to concurrency limit (context timeout/cancelled)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			c.Header("Retry-After", "5")
			httpx.Err(c, http.StatusTooManyRequests, "thumbnail generation is busy, retry after a few seconds")
			return
		}
		logger.L.Error("video thumbnail generation failed", "file", fullPath, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.File(thumbPath)
}
