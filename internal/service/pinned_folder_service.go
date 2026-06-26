package service

import (
	"fmt"
	"net/http"

	"go-file-server/internal/logger"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"

	"github.com/gin-gonic/gin"
)

const maxPinnedFolders = 10

type PinnedFolderService struct {
	Repo repository.PinnedFolderRepository
}

func NewPinnedFolderService(repo repository.PinnedFolderRepository) *PinnedFolderService {
	return &PinnedFolderService{Repo: repo}
}

type pinAddRequest struct {
	Path string `json:"path" binding:"required"`
}

type pinReorderRequest struct {
	Paths []string `json:"paths" binding:"required"`
}

func (s *PinnedFolderService) Add(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req pinAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	count, err := s.Repo.Count(username.(string))
	if err != nil {
		logger.L.Error("failed to count pinned folders", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count >= maxPinnedFolders {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Maximum %d pinned folders allowed", maxPinnedFolders)})
		return
	}

	if err := s.Repo.Add(username.(string), req.Path); err != nil {
		logger.L.Error("failed to add pinned folder", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder pinned"})
}

func (s *PinnedFolderService) Remove(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query parameter is required"})
		return
	}

	rows, err := s.Repo.Remove(username.(string), path)
	if err != nil {
		logger.L.Error("failed to remove pinned folder", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pinned folder not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder unpinned"})
}

func (s *PinnedFolderService) List(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	folders, err := s.Repo.List(username.(string))
	if err != nil {
		logger.L.Error("failed to list pinned folders", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if folders == nil {
		folders = []model.PinnedFolder{}
	}

	c.JSON(http.StatusOK, gin.H{"pins": folders})
}

func (s *PinnedFolderService) Reorder(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req pinReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := s.Repo.Reorder(username.(string), req.Paths); err != nil {
		logger.L.Error("failed to reorder pinned folders", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pinned folders reordered"})
}
