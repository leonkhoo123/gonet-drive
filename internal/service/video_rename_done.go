package service

import (
	"fmt"
	"go-file-server/internal/config"
	"go-file-server/internal/logger"
	"go-file-server/internal/util"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// VideoRenameDone renames a video file and moves it to a 'done' folder, with optional rotation.
// @Summary      Rename Video (Done)
// @Description  Move a video file to the 'done' subdirectory with optional rotation.
// @Tags         Media
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      object  true  "Request with path, newName, rotateAngle, opId"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/video/rename-done [post]
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

	logger.L.Debug("video rename requested", "opId", req.OpID, "path", req.Path, "newName", req.NewName, "angle", req.RotateAngle)

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
		logger.L.Debug("applying video rotation", "angle", req.RotateAngle, "file", newName)
		rotatedPath, err := util.AdjustVideoRotationTemp(c.Request.Context(), cfg.Server.FileRoot, srcPath, req.RotateAngle)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		srcPath = rotatedPath // continue with rename/move after rotation
	}

	// Target path
	destPath, err := util.ResolveDuplicatePath(doneDir, newName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"})
		return
	}
	destPath = filepath.Clean(destPath)
	srcPath = filepath.Clean(srcPath)
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
