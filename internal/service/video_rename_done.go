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

func VideoRenameDone(c *gin.Context, cfg *config.CloudConfig) {
	var req struct {
		Path        string `json:"path"`
		NewName     string `json:"newName"`
		RotateAngle int    `json:"rotateAngle"`
		OpID        string `json:"opId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	log.Printf("[OpID: %s] VideoRenameDone: path=%s, newName=%s, angle=%d", req.OpID, req.Path, req.NewName, req.RotateAngle)

	srcPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access forbidden: " + err.Error()})
		return
	}

	newName, err := util.SanitizeFilename(req.NewName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename: " + err.Error()})
		return
	}

	parentDir := filepath.Dir(srcPath)
	doneDir := filepath.Join(parentDir, "done")

	// Create "done" folder if not exists
	if err := os.MkdirAll(doneDir, 0777); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create done folder"})
		return
	}

	if req.RotateAngle != 0 {
		log.Printf("Will add [%d°] to [%s]", req.RotateAngle, newName)
		rotatedPath, err := util.AdjustVideoRotationTemp(cfg.Server.FileRoot, srcPath, req.RotateAngle)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		srcPath = rotatedPath // continue with rename/move after rotation
	}

	// Target path
	destPath := util.ResolveDuplicatePath(doneDir, newName)
	// Rename (move)
	if err := os.Rename(srcPath, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to move file: %v", err),
		})
		return
	}

	// Optional: simulate small delay before returning JSON
	time.Sleep(500 * time.Millisecond)

	c.JSON(http.StatusOK, gin.H{
		"message":   "File moved to done folder successfully",
		"original":  req.Path,
		"new_name":  newName,
		"new_path":  filepath.ToSlash(destPath),
		"timestamp": time.Now(),
	})
}
