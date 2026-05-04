package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"go-file-server/internal/config"

	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

var videoThumbnailGroup singleflight.Group

// ServeVideo serves video files with HTTP range support
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
		var start, end int64
		n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if n == 1 {
			end = stat.Size() - 1
		}
		if end >= stat.Size() {
			end = stat.Size() - 1
		}

		chunkSize := (end - start) + 1

		c.Status(http.StatusPartialContent)
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, stat.Size()))
		c.Header("Accept-Ranges", "bytes")
		c.Header("Content-Length", fmt.Sprintf("%d", chunkSize))
		c.Header("Content-Type", util.MimeType(fullPath))

		// Stream directly from file descriptor (no full buffering)
		file.Seek(start, io.SeekStart)
		io.CopyN(c.Writer, file, chunkSize)
		return
	}

	// no Range header → stream entire file
	c.Header("Accept-Ranges", "bytes")
	c.File(fullPath)
}

// ServeVideoThumbnail serves a generated thumbnail for a video file
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
		// Use ffmpeg to extract a frame and generate a thumbnail, limiting size to 300x300 while keeping aspect ratio.
		// ffmpeg natively respects orientation metadata when converting.
		cmd := exec.Command("ffmpeg",
			"-i", fullPath,
			"-ss", "00:00:00.000",
			"-vframes", "1",
			"-vf", "scale='min(300,iw)':min'(300,ih)':force_original_aspect_ratio=decrease",
			"-c:v", "libwebp",
			"-y",
			thumbPath)

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to generate video thumbnail: %w", err)
		}
		return nil, nil
	})

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.File(thumbPath)
}
