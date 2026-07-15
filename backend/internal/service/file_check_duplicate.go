package service

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

type CheckDuplicateReq struct {
	Sources []string `json:"sources" binding:"required"`
	DestDir string   `json:"destDir" binding:"required"`
}

type FileDetail struct {
	Name       string    `json:"name"`
	IsDir      bool      `json:"isDir"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type DuplicateItem struct {
	Source FileDetail `json:"source"`
	Target FileDetail `json:"target"`
}

// CheckDuplicates checks for name collisions when copying/moving files to a destination.
// @Summary      Check Duplicates
// @Description  Check for name collision between source and destination paths.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      CheckDuplicateReq  true  "Check duplicates request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/check-duplicates [post]
func CheckDuplicates(c *gin.Context, cfg *config.CloudConfig) {
	var req CheckDuplicateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Err(c, http.StatusBadRequest, "invalid request")
		return
	}

	safeSources, err := util.SanitizeRepoPaths(cfg.Server.FileRoot, req.Sources)
	if err != nil {
		httpx.Err(c, http.StatusForbidden, "Access forbidden: "+err.Error())
		return
	}

	safeDestDir, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.DestDir)
	if err != nil {
		httpx.Err(c, http.StatusForbidden, "Access forbidden: "+err.Error())
		return
	}

	var duplicates []DuplicateItem

	for _, src := range safeSources {
		baseName := filepath.Base(src)
		destPath := filepath.Join(safeDestDir, baseName)

		destInfo, errDest := os.Stat(destPath)
		srcInfo, errSrc := os.Stat(src)

		if errDest == nil && errSrc == nil {
			// Item exists in both source and destination
			duplicates = append(duplicates, DuplicateItem{
				Source: FileDetail{
					Name:       baseName,
					IsDir:      srcInfo.IsDir(),
					Size:       srcInfo.Size(),
					ModifiedAt: srcInfo.ModTime(),
				},
				Target: FileDetail{
					Name:       baseName,
					IsDir:      destInfo.IsDir(),
					Size:       destInfo.Size(),
					ModifiedAt: destInfo.ModTime(),
				},
			})
		}
	}

	if duplicates == nil {
		duplicates = make([]DuplicateItem, 0)
	}

	httpx.OK(c, http.StatusOK, gin.H{
		"hasDuplicates": len(duplicates) > 0,
		"duplicates":    duplicates,
	})
}

type UploadFileDetail struct {
	Path       string    `json:"path"` // relative path including filename
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type CheckUploadDuplicateReq struct {
	Files   []UploadFileDetail `json:"files" binding:"required"`
	DestDir string             `json:"destDir" binding:"required"`
}

// CheckUploadDuplicates checks for name collisions when uploading files.
// @Summary      Check Upload Duplicates
// @Description  Check for file name collision when uploading files to a destination.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      CheckUploadDuplicateReq  true  "Check upload duplicates request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/check-upload-duplicates [post]
func CheckUploadDuplicates(c *gin.Context, cfg *config.CloudConfig) {
	var req CheckUploadDuplicateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Err(c, http.StatusBadRequest, "invalid request")
		return
	}

	safeDestDir, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.DestDir)
	if err != nil {
		httpx.Err(c, http.StatusForbidden, "Access forbidden: "+err.Error())
		return
	}

	var duplicates []DuplicateItem

	for _, file := range req.Files {
		// Verify relative path is safe within destination directory
		destPath, err := util.SanitizeRepoPath(safeDestDir, file.Path)
		if err != nil {
			continue // Skip invalid paths
		}

		destInfo, errDest := os.Stat(destPath)

		if errDest == nil {
			// Item exists in destination
			duplicates = append(duplicates, DuplicateItem{
				Source: FileDetail{
					Name:       filepath.Base(file.Path),
					IsDir:      false,
					Size:       file.Size,
					ModifiedAt: file.ModifiedAt,
				},
				Target: FileDetail{
					Name:       filepath.Base(destPath),
					IsDir:      destInfo.IsDir(),
					Size:       destInfo.Size(),
					ModifiedAt: destInfo.ModTime(),
				},
			})
		}
	}

	if duplicates == nil {
		duplicates = make([]DuplicateItem, 0)
	}

	httpx.OK(c, http.StatusOK, gin.H{
		"hasDuplicates": len(duplicates) > 0,
		"duplicates":    duplicates,
	})
}
