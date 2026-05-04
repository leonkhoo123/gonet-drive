package service

import (
	"fmt"
	"go-file-server/internal/config"
	"go-file-server/internal/util"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

func VideoDisqualified(c *gin.Context, cfg *config.CloudConfig) {
	var req struct {
		Path string `json:"path"`
		OpID string `json:"opId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	log.Printf("[OpID: %s] VideoDisqualified: path=%s", req.OpID, req.Path)

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'path' field"})
		return
	}

	// Sanitize source path
	srcPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access forbidden: " + err.Error()})
		return
	}
	parentDir := filepath.Dir(srcPath)
	disqualifiedDir := filepath.Join(parentDir, "disqualified")

	// Create disqualified folder if not exists
	if err := os.MkdirAll(disqualifiedDir, 0777); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create disqualified folder"})
		return
	}

	// Destination file path
	// destPath := filepath.Join(disqualifiedDir, filepath.Base(srcPath))

	destPath, err := util.ResolveDuplicatePath(disqualifiedDir, filepath.Base(srcPath))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	// Move (rename) file
	if err := os.Rename(srcPath, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to move file: %v", err),
		})
		return
	}

	time.Sleep(500 * time.Millisecond) // sleep for 0.5 seconds

	c.JSON(http.StatusOK, gin.H{
		"message":  "File moved to disqualified folder successfully",
		"original": req.Path,
		"new_path": filepath.ToSlash(filepath.Join(filepath.Dir(req.Path), "disqualified", filepath.Base(req.Path))),
	})
}
