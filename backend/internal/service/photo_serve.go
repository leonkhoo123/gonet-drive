package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

var thumbnailGroup singleflight.Group

// ServePhoto serves photo/image files.
// @Summary      Serve Photo
// @Description  Serve a photo/image file.
// @Tags         Media
// @Produce      image/*
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true  "Relative file path"
// @Success      200  {file}  binary
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/photo/play/file/{filepath} [get]
func ServePhoto(c *gin.Context, cfg *config.CloudConfig) {
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

	c.File(fullPath)
}

// ServePhotoThumbnail serves a generated thumbnail for a photo file
// ServePhotoThumbnail generates and serves a photo thumbnail.
// @Summary      Photo Thumbnail
// @Description  Generate and serve a thumbnail for a photo/image file.
// @Tags         Media
// @Produce      image/webp
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true  "Relative file path"
// @Success      200  {file}  binary
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/photo/thumbnail/file/{filepath} [get]
func ServePhotoThumbnail(c *gin.Context, cfg *config.CloudConfig) {
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
	_, err, _ = thumbnailGroup.Do(thumbPath, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Server.ThumbnailGenerationTimeout)
		defer cancel()

		sem := GetThumbnailSemaphore()
		if err := sem.Acquire(ctx); err != nil {
			return nil, err
		}
		defer sem.Release()

		if err := GeneratePhotoThumbnail(ctx, fullPath, thumbPath); err != nil {
			return nil, err
		}

		UpsertThumbnailRecord(hashStr, fullPath, false)
		return nil, nil
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			c.Header("Retry-After", "5")
			httpx.Err(c, http.StatusTooManyRequests, "thumbnail generation is busy, retry after a few seconds")
			return
		}
		logger.L.Error("photo thumbnail generation failed", "file", fullPath, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.File(thumbPath)
}
