package service

import (
	"fmt"
	"net/http"

	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"

	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

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
	username, exists := c.Get(authgin.KeyUsername)
	if !exists {
		httpx.Err(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req pinAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Err(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	count, err := s.Repo.Count(username.(string))
	if err != nil {
		logger.L.Error("failed to count pinned folders", "err", err)
		httpx.Err(c, http.StatusInternalServerError, "Database error")
		return
	}
	if count >= maxPinnedFolders {
		httpx.Err(c, http.StatusBadRequest, fmt.Sprintf("Maximum %d pinned folders allowed", maxPinnedFolders))
		return
	}

	if err := s.Repo.Add(username.(string), req.Path); err != nil {
		logger.L.Error("failed to add pinned folder", "err", err)
		httpx.Err(c, http.StatusInternalServerError, "Database error")
		return
	}

	httpx.Msg(c, http.StatusOK, "Folder pinned")
}

func (s *PinnedFolderService) Remove(c *gin.Context) {
	username, exists := c.Get(authgin.KeyUsername)
	if !exists {
		httpx.Err(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	path := c.Query("path")
	if path == "" {
		httpx.Err(c, http.StatusBadRequest, "path query parameter is required")
		return
	}

	rows, err := s.Repo.Remove(username.(string), path)
	if err != nil {
		logger.L.Error("failed to remove pinned folder", "err", err)
		httpx.Err(c, http.StatusInternalServerError, "Database error")
		return
	}

	if rows == 0 {
		httpx.Err(c, http.StatusNotFound, "Pinned folder not found")
		return
	}

	httpx.Msg(c, http.StatusOK, "Folder unpinned")
}

func (s *PinnedFolderService) List(c *gin.Context) {
	username, exists := c.Get(authgin.KeyUsername)
	if !exists {
		httpx.Err(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	folders, err := s.Repo.List(username.(string))
	if err != nil {
		logger.L.Error("failed to list pinned folders", "err", err)
		httpx.Err(c, http.StatusInternalServerError, "Database error")
		return
	}
	if folders == nil {
		folders = []model.PinnedFolder{}
	}

	httpx.OK(c, http.StatusOK, gin.H{"pins": folders})
}

func (s *PinnedFolderService) Reorder(c *gin.Context) {
	username, exists := c.Get(authgin.KeyUsername)
	if !exists {
		httpx.Err(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req pinReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Err(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if err := s.Repo.Reorder(username.(string), req.Paths); err != nil {
		logger.L.Error("failed to reorder pinned folders", "err", err)
		httpx.Err(c, http.StatusInternalServerError, "Database error")
		return
	}

	httpx.Msg(c, http.StatusOK, "Pinned folders reordered")
}
