package service

import (
	"fmt"
	"go-file-server/internal/config"
	"go-file-server/internal/storage"
	"go-file-server/internal/util"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func FileList(c *gin.Context, cfg *config.CloudConfig) {
	relPath := c.Query("path") // e.g. /, /videos, /videos/travel/2024
	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, relPath)
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	cleanPath, _ := filepath.Rel(cfg.Server.FileRoot, fullPath)
	if cleanPath == "." {
		cleanPath = "/"
	} else {
		cleanPath = "/" + filepath.ToSlash(cleanPath)
	}

	displayPath := cleanPath

	// Auto-create .cloud_delete directory if requested and it doesn't exist
	if cleanPath == "/.cloud_delete" {
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			os.MkdirAll(fullPath, 0755)
		}
	}

	sortBy := c.DefaultQuery("sort", "name") // name | size | modified
	order := c.DefaultQuery("order", "asc")  // asc | desc

	showHiddenStr := c.DefaultQuery("showHidden", "false")
	showHidden := showHiddenStr == "true"

	// Find shared paths
	sharedPaths := make(map[string]bool)
	if config.DB != nil {
		likePattern := displayPath
		if displayPath == "/" {
			likePattern = "/%"
		} else {
			likePattern = displayPath + "/%"
		}

		rows, err := config.DB.Query(
			`SELECT path FROM sharing_info WHERE expires_at > ? AND path LIKE ?`,
			time.Now(), likePattern,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p string
				if err := rows.Scan(&p); err == nil {
					sharedPaths[p] = true
				}
			}
		}
	}

	entries, err := os.ReadDir(fullPath)
	isSingleFile := false
	if err != nil {
		info, statErr := os.Stat(fullPath)
		if statErr == nil && !info.IsDir() {
			entries = []os.DirEntry{fs.FileInfoToDirEntry(info)}
			isSingleFile = true
		} else {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	}

	var result []gin.H
	var fileCount, folderCount int

	for _, e := range entries {
		if e.Name() == ".cloud_reserve" || e.Name() == ".cloud_delete" {
			continue
		}
		if !showHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}

		if e.IsDir() {
			folderCount++
		} else {
			fileCount++
		}

		info, _ := e.Info()
		url := ""
		mediaType := ""

		itemPath := ""
		if isSingleFile {
			itemPath = displayPath
		} else {
			if displayPath == "/" {
				itemPath = "/" + e.Name()
			} else {
				itemPath = displayPath + "/" + e.Name()
			}
		}

		if isVideoFile(info.Name()) {
			baseURL := cfg.Server.Hostname
			if baseURL == "" {
				baseURL = util.GetBaseURL(c.Request)
			}
			url = fmt.Sprintf("%s/api/user/video/play/file%s", baseURL, itemPath)
			mediaType = "video"
		} else if isPhotoFile(info.Name()) {
			baseURL := cfg.Server.Hostname
			if baseURL == "" {
				baseURL = util.GetBaseURL(c.Request)
			}
			url = fmt.Sprintf("%s/api/user/photo/play/file%s", baseURL, itemPath)
			mediaType = "photo"
		} else if isMusicFile(info.Name()) {
			baseURL := cfg.Server.Hostname
			if baseURL == "" {
				baseURL = util.GetBaseURL(c.Request)
			}
			url = fmt.Sprintf("%s/api/user/music/play/file%s", baseURL, itemPath)
			mediaType = "music"
		} else if isPdfFile(info.Name()) {
			baseURL := cfg.Server.Hostname
			if baseURL == "" {
				baseURL = util.GetBaseURL(c.Request)
			}
			url = fmt.Sprintf("%s/api/user/document/read/file%s", baseURL, itemPath)
			mediaType = "pdf"
		} else if isTextFile(info.Name()) {
			baseURL := cfg.Server.Hostname
			if baseURL == "" {
				baseURL = util.GetBaseURL(c.Request)
			}
			url = fmt.Sprintf("%s/api/user/document/read/file%s", baseURL, itemPath)
			mediaType = "text_documents"
		}

		result = append(result, gin.H{
			"name":       e.Name(),
			"type":       map[bool]string{true: "dir", false: "file"}[e.IsDir()],
			"size":       info.Size(),
			"modified":   info.ModTime(),
			"url":        url,
			"media_type": mediaType,
			"path":       itemPath,
			"isShared":   sharedPaths[itemPath],
		})
	}

	sortMethod := ""
	ordering := "asc"
	// --- Sorting ---
	sort.Slice(result, func(i, j int) bool {
		isDirI := result[i]["type"].(string) == "dir"
		isDirJ := result[j]["type"].(string) == "dir"

		// --- Always show folders first ---
		if isDirI != isDirJ {
			return isDirI // folders first
		}

		var less bool
		switch sortBy {
		case "size":
			// Only compare size if both are files
			if !isDirI && !isDirJ {
				less = result[i]["size"].(int64) < result[j]["size"].(int64)
			} else {
				// Folders: sort by name
				less = strings.ToLower(result[i]["name"].(string)) < strings.ToLower(result[j]["name"].(string))
			}
			sortMethod = "size"

		case "modified":
			less = result[i]["modified"].(time.Time).Before(result[j]["modified"].(time.Time))
			sortMethod = "modified"

		default: // name
			less = strings.ToLower(result[i]["name"].(string)) < strings.ToLower(result[j]["name"].(string))
			sortMethod = "name"
		}

		if order == "desc" {
			ordering = "desc"
			return !less
		}
		return less
	})

	var limit int64 = 0
	if config.AppCloudConfig != nil {
		limit = config.AppCloudConfig.StorageLimit
	}

	response := gin.H{
		"path":           displayPath,
		"sort":           sortMethod,
		"order":          ordering,
		"items":          result,
		"file_count":     fileCount,
		"folder_count":   folderCount,
		"count":          len(result),
		"is_single_file": isSingleFile,
		"storage": gin.H{
			"used":  storage.GetUsage(),
			"limit": limit,
		},
	}

	c.JSON(http.StatusOK, response)
}

// helper function to detect video files
func isVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".mp4", ".mkv", ".mov", ".avi", ".webm":
		return true
	default:
		return false
	}
}

// helper function to detect photo files
func isPhotoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".heic":
		return true
	default:
		return false
	}
}

// helper function to detect music files
func isMusicFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".mp3", ".flac", ".wav", ".ogg", ".m4a", ".aac", ".wma":
		return true
	default:
		return false
	}
}

// helper function to detect text document files
func isTextFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".txt", ".md", ".markdown", ".json", ".yaml", ".yml", ".xml", ".csv", ".ini", ".conf", ".cfg", ".log", ".sh", ".bash", ".zsh", ".js", ".jsx", ".ts", ".tsx", ".go", ".py", ".c", ".cpp", ".h", ".hpp", ".java", ".html", ".css", ".sql", ".dockerfile", "dockerfile", "makefile", ".bat", ".ps1":
		return true
	default:
		// Also allow specific filenames without extension
		lowerName := strings.ToLower(name)
		if lowerName == "dockerfile" || lowerName == "makefile" || lowerName == "caddyfile" {
			return true
		}
		return false
	}
}

// helper function to detect pdf files
func isPdfFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".pdf":
		return true
	default:
		return false
	}
}
