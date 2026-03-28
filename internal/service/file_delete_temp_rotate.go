package service

import (
	"go-file-server/internal/config"
	"go-file-server/internal/util"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func DeleteTempRotate(c *gin.Context, cfg *config.CloudConfig) {
	var req struct {
		Path string `json:"path"`
		OpID string `json:"opId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	log.Printf("[OpID: %s] DeleteTempRotate: path=%s", req.OpID, req.Path)

	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Access forbidden: " + err.Error(),
		})
		return
	}

	removed := 0
	err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip special directories
			if d.Name() == ".cloud_reserve" || d.Name() == ".cloud_delete" {
				return filepath.SkipDir
			}
			if d.Name() == "tmp_rotate" {
				if rmErr := os.RemoveAll(path); rmErr != nil {
					log.Printf("Failed to remove %s: %v", path, rmErr)
					return rmErr
				}
				log.Printf("Removed folder: %s", path)
				removed++
				// skip walking deeper inside deleted dir
				return filepath.SkipDir
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"removedDirs": removed,
	})
}
