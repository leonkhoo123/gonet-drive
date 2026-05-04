package service

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"
	"go-file-server/internal/util"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tcolgate/mp3"
)

type AudiobookService struct {
	PathRepo     repository.AudioPathRepository
	ProgressRepo repository.AudiobookProgressRepository
}

func NewAudiobookService(pathRepo repository.AudioPathRepository, progressRepo repository.AudiobookProgressRepository) *AudiobookService {
	return &AudiobookService{
		PathRepo:     pathRepo,
		ProgressRepo: progressRepo,
	}
}

type AudioBookItem struct {
	Name         string    `json:"name"`
	TotalLength  float64   `json:"total_length"` // in seconds
	Size         int64     `json:"size"`
	ProgressTime float64   `json:"progress_time"`
	LastModified time.Time `json:"last_modified"`
	MediaType    string    `json:"mediaType"`
	Url          string    `json:"url,omitempty"`
}

type PathReq struct {
	Path      string `json:"path"`
	IsEnabled *bool  `json:"is_enabled"`
}

type ProgressReq struct {
	AudioBookName string  `json:"audiobook_name"`
	ProgressTime  float64 `json:"progress_time"`
}

func (s *AudiobookService) GetAudioBookPaths(c *gin.Context) {
	username := c.GetString("username")
	paths, err := s.PathRepo.GetByUsername(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if paths == nil {
		paths = []model.AudioPath{}
	}

	c.JSON(http.StatusOK, paths)
}

func (s *AudiobookService) AddAudioBookPath(c *gin.Context) {
	username := c.GetString("username")
	var req PathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	path := &model.AudioPath{
		Path:      req.Path,
		Username:  username,
		IsEnabled: isEnabled,
	}

	err := s.PathRepo.Create(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add path or duplicate path"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Path added successfully"})
}

func (s *AudiobookService) UpdateAudioBookPath(c *gin.Context) {
	username := c.GetString("username")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req PathReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.IsEnabled != nil {
		err := s.PathRepo.UpdateEnabled(id, username, *req.IsEnabled)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update path"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Path updated successfully"})
}

func (s *AudiobookService) DeleteAudioBookPath(c *gin.Context) {
	username := c.GetString("username")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	err = s.PathRepo.Delete(id, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete path"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Path deleted successfully"})
}

func (s *AudiobookService) ReportAudioBookProgress(c *gin.Context) {
	username := c.GetString("username")
	var req ProgressReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	progress := &model.AudiobookProgress{
		Username:      username,
		AudioBookName: req.AudioBookName,
		ProgressTime:  req.ProgressTime,
	}

	err := s.ProgressRepo.Upsert(progress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Progress updated successfully"})
}

// Estimate MP3 Duration more accurately using frame headers and VBR Xing headers if available
func estimateDuration(path string, size int64) float64 {
	f, err := os.Open(path)
	if err != nil {
		return float64(size) * 8 / 128000.0
	}
	defer f.Close()

	d := mp3.NewDecoder(f)
	var frame mp3.Frame
	skipped := 0

	err = d.Decode(&frame, &skipped)
	if err != nil {
		return float64(size) * 8 / 128000.0
	}

	frameReader := frame.Reader()
	buf, _ := io.ReadAll(frameReader)

	// Search for Xing or Info header
	idx := bytes.Index(buf, []byte("Xing"))
	if idx == -1 {
		idx = bytes.Index(buf, []byte("Info"))
	}

	// If a VBR header is found, extract total frames to calculate exact duration
	if idx != -1 && idx+8 <= len(buf) {
		flags := binary.BigEndian.Uint32(buf[idx+4 : idx+8])
		// Check if Frames field is present (Bit 0)
		if flags&0x01 != 0 && idx+12 <= len(buf) {
			frames := binary.BigEndian.Uint32(buf[idx+8 : idx+12])
			if frames > 0 {
				return float64(frames) * frame.Duration().Seconds()
			}
		}
	}

	audioBytes := size - int64(skipped)
	if audioBytes < 0 {
		audioBytes = size
	}

	// For CBR, calculate using the bitrate from the first frame
	bitrate := frame.Header().BitRate()
	if bitrate > 0 {
		return float64(audioBytes*8) / float64(bitrate)
	}

	// Fallback assumption
	return float64(size) * 8 / 128000.0
}

func (s *AudiobookService) ListAudioBooks(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetString("username")

		// Get all enabled paths
		rawPaths, err := s.PathRepo.GetEnabledPathsByUsername(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		var paths []string
		for _, p := range rawPaths {
			// Resolve absolute path using FileRoot
			fullPath := filepath.Join(cfg.Server.FileRoot, p)
			paths = append(paths, fullPath)
		}

		// Get user progress
		progressMap, err := s.ProgressRepo.GetByUsername(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		var audioBooks []AudioBookItem
		seen := make(map[string]bool)

		// Scan paths
		for _, dir := range paths {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue // Skip if dir not found
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if strings.HasPrefix(name, "ab_") && strings.HasSuffix(strings.ToLower(name), ".mp3") {
					info, err := entry.Info()
					if err != nil {
						continue
					}

					baseName := strings.TrimPrefix(name, "ab_")
					baseName = strings.TrimSuffix(baseName, ".mp3") // Or keep .mp3 depending on preference, removing 'ab_' as requested.
					baseName = strings.TrimSuffix(baseName, ".MP3")

					// keep the original filename for uniqueness or remove ab_
					audioBookName := strings.TrimPrefix(name, "ab_")

					if seen[audioBookName] {
						continue
					}
					seen[audioBookName] = true

					duration := estimateDuration(filepath.Join(dir, name), info.Size())

					prog, ok := progressMap[audioBookName]
					if !ok {
						prog = 0
					}

					audioBooks = append(audioBooks, AudioBookItem{
						Name:         audioBookName,
						TotalLength:  duration,
						Size:         info.Size(),
						ProgressTime: prog,
						LastModified: info.ModTime(),
						MediaType:    "audio_book",
					})
				}
			}
		}

		if audioBooks == nil {
			audioBooks = []AudioBookItem{} // Return empty array instead of null
		}

		c.JSON(http.StatusOK, audioBooks)
	}
}

func (s *AudiobookService) StreamAudioBook(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetString("username")
		name := c.Param("name") // e.g., xyz.mp3 (without ab_)

		if !util.IsSafePathComponent(name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid name"})
			return
		}

		targetFile := "ab_" + name

		// Find the file in enabled paths
		rawPaths, err := s.PathRepo.GetEnabledPathsByUsername(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		var foundPath string
		for _, p := range rawPaths {
			fullPath := filepath.Join(cfg.Server.FileRoot, p, targetFile)
			if _, err := os.Stat(fullPath); err == nil {
				foundPath = fullPath
				break
			}
		}

		if foundPath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Audiobook not found"})
			return
		}

		c.File(foundPath)
	}
}

func (s *AudiobookService) AudioBookFileList(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		relPath := c.Query("path")
		if relPath == "" {
			relPath = "/.audio_book"
		}

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

		entries, err := os.ReadDir(fullPath)
		if err != nil {
			// Try to handle missing directory gracefully if it is the default
			if relPath == "/.audio_book" {
				c.JSON(http.StatusOK, gin.H{
					"path":  cleanPath,
					"items": []AudioBookItem{},
				})
				return
			}
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		username := c.GetString("username")
		// Get user progress
		progressMap, err := s.ProgressRepo.GetByUsername(username)
		if err != nil {
			progressMap = make(map[string]float64)
		}

		var items []AudioBookItem

		for _, e := range entries {
			if e.Name() == ".cloud_reserve" || e.Name() == ".cloud_delete" || strings.HasPrefix(e.Name(), ".") {
				continue
			}

			info, err := e.Info()
			if err != nil {
				continue
			}

			if e.IsDir() {
				items = append(items, AudioBookItem{
					Name:         e.Name(),
					Size:         info.Size(),
					LastModified: info.ModTime(),
					MediaType:    "dir",
				})
				continue
			}

			name := e.Name()
			if strings.HasPrefix(name, "ab_") {
				lowerName := strings.ToLower(name)
				if !strings.HasSuffix(lowerName, ".mp3") && !strings.HasSuffix(lowerName, ".m4a") && !strings.HasSuffix(lowerName, ".wav") && !strings.HasSuffix(lowerName, ".flac") && !strings.HasSuffix(lowerName, ".ogg") && !strings.HasSuffix(lowerName, ".aac") {
					continue
				}

				audioBookName := strings.TrimPrefix(name, "ab_")
				// some files might have extension stripped in DB? In ListAudioBooks we only trim "ab_". DB query uses audioBookName exactly.
				// wait, ListAudioBooks strips "ab_" but keeps ".mp3" for `audioBookName`?
				// "audioBookName := strings.TrimPrefix(name, "ab_")" is what ListAudioBooks uses.

				duration := estimateDuration(filepath.Join(fullPath, name), info.Size())

				prog, ok := progressMap[audioBookName]
				if !ok {
					prog = 0
				}

				baseURL := cfg.Server.Hostname
				if baseURL == "" {
					baseURL = util.GetBaseURL(c.Request)
				}
				url := fmt.Sprintf("%s/api/user/music/play/file%s", baseURL, filepath.Join(cleanPath, name))

				items = append(items, AudioBookItem{
					Name:         name,
					TotalLength:  duration,
					Size:         info.Size(),
					ProgressTime: prog,
					LastModified: info.ModTime(),
					MediaType:    "audio_book",
					Url:          url,
				})
			}
		}

		if items == nil {
			items = []AudioBookItem{}
		}

		// Sort: dirs first, then by name
		sort.Slice(items, func(i, j int) bool {
			if items[i].MediaType == "dir" && items[j].MediaType != "dir" {
				return true
			}
			if items[i].MediaType != "dir" && items[j].MediaType == "dir" {
				return false
			}
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		})

		c.JSON(http.StatusOK, gin.H{
			"path":  cleanPath,
			"items": items,
		})
	}
}
