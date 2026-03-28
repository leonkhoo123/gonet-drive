package service

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go-file-server/internal/config"
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

func CheckDuplicates(c *gin.Context, cfg *config.CloudConfig) {
	var req CheckDuplicateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	safeSources, err := util.SanitizeRepoPaths(cfg.Server.FileRoot, req.Sources)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access forbidden: " + err.Error()})
		return
	}

	safeDestDir, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.DestDir)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access forbidden: " + err.Error()})
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

	c.JSON(http.StatusOK, gin.H{
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

func CheckUploadDuplicates(c *gin.Context, cfg *config.CloudConfig) {
	var req CheckUploadDuplicateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	safeDestDir, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.DestDir)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access forbidden: " + err.Error()})
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

	c.JSON(http.StatusOK, gin.H{
		"hasDuplicates": len(duplicates) > 0,
		"duplicates":    duplicates,
	})
}
