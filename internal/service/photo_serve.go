package service

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"

	"go-file-server/internal/config"
	"go-file-server/internal/util"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

var thumbnailGroup singleflight.Group

// ServePhoto serves photo files
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
		// Open original image
		src, err := imaging.Open(fullPath, imaging.AutoOrientation(true))
		if err != nil {
			return nil, err
		}

		// Resize to 300px width, preserving aspect ratio
		// Using Lanczos for better downscaling quality compared to Box
		dst := imaging.Resize(src, 300, 0, imaging.Lanczos)

		// Save as WebP
		out, err := os.Create(thumbPath)
		if err != nil {
			return nil, err
		}
		defer out.Close()

		if err := webp.Encode(out, dst, &webp.Options{Lossless: false, Quality: 85}); err != nil {
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.File(thumbPath)
}
