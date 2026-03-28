package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/util"
)

type PropertiesReq struct {
	Sources []string `json:"sources" binding:"required"`
}

type PropertiesContains struct {
	Files   int `json:"files"`
	Folders int `json:"folders"`
}

type PropertiesResponse struct {
	Type           string             `json:"type"` // "file", "directory", or "multiple"
	Name           *string            `json:"name"` // null if multiple
	Location       string             `json:"location"`
	TotalSizeBytes int64              `json:"totalSizeBytes"`
	Contains       PropertiesContains `json:"contains"`
	CreatedAt      *time.Time         `json:"createdAt"`
	ModifiedAt     *time.Time         `json:"modifiedAt"`
	AccessedAt     *time.Time         `json:"accessedAt"`
}

func GetFileProperties(req PropertiesReq, cfg *config.CloudConfig) (*PropertiesResponse, error) {
	safeSources, err := util.SanitizeRepoPaths(cfg.Server.FileRoot, req.Sources)
	if err != nil {
		return nil, err
	}

	if len(safeSources) == 0 {
		return nil, fmt.Errorf("no sources provided")
	}

	res := &PropertiesResponse{
		Contains: PropertiesContains{},
	}

	// Calculate location based on the original request paths
	relParentDir := filepath.Dir(req.Sources[0])
	res.Location = relParentDir

	for _, src := range req.Sources {
		if filepath.Dir(src) != relParentDir {
			res.Location = "Multiple Locations"
			break
		}
	}

	if len(safeSources) == 1 {
		safePath := safeSources[0]
		info, err := os.Stat(safePath)
		if err != nil {
			return nil, err
		}

		name := info.Name()
		res.Name = &name

		if info.IsDir() {
			res.Type = "directory"
		} else {
			res.Type = "file"
		}

		modTime := info.ModTime()
		res.ModifiedAt = &modTime

		if info.IsDir() {
			size, files, folders := getDirStats(safePath)
			res.TotalSizeBytes = size
			res.Contains.Files = files
			res.Contains.Folders = folders
		} else {
			res.TotalSizeBytes = info.Size()
		}
	} else {
		res.Type = "multiple"
		res.Name = nil

		for _, safePath := range safeSources {
			info, err := os.Stat(safePath)
			if err != nil {
				return nil, err
			}

			if info.IsDir() {
				size, files, folders := getDirStats(safePath)
				res.TotalSizeBytes += size
				res.Contains.Files += files
				res.Contains.Folders += folders + 1
			} else {
				res.TotalSizeBytes += info.Size()
				res.Contains.Files += 1
			}
		}
	}

	return res, nil
}

func getDirStats(dirPath string) (size int64, files int, folders int) {
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if path == dirPath {
			return nil
		}
		if info.IsDir() {
			folders++
		} else {
			files++
			size += info.Size()
		}
		return nil
	})
	return
}
