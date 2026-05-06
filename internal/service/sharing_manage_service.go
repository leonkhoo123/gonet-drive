package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"go-file-server/internal/model"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CreateShareRequest struct {
	Path        string `json:"path" binding:"required"`
	Description string `json:"description" binding:"required"`
	ExpiresIn   int    `json:"expires_in_hours" binding:"required"`
	Authority   string `json:"authority"`
}

func generateRandomID() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	return strings.ReplaceAll(base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), "-", "")
}

func generateRandomPin() string {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "123456"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func (s *SharingService) CreateShareEndpoint(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.Authority != "modify" {
		req.Authority = "view"
	}

	// Validate expires_in_hours: -1 = never, >= 1 = hours
	if req.ExpiresIn != -1 && req.ExpiresIn < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expires_in_hours must be -1 (never) or >= 1"})
		return
	}

	pin := generateRandomPin()

	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash PIN"})
		return
	}

	id := generateRandomID()
	var expiresAt time.Time
	if req.ExpiresIn == -1 {
		expiresAt = model.NeverExpires
	} else {
		expiresAt = time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
	}
	createdAt := time.Now()

	share := &model.SharingInfo{
		ID:          id,
		Path:        req.Path,
		PinHash:     string(hash),
		ExpiresAt:   expiresAt,
		Blocked:     false,
		Authority:   req.Authority,
		Username:    username.(string),
		Description: req.Description,
		CreatedAt:   createdAt,
	}

	if err := s.ShareRepo.Create(share); err != nil {
		log.Printf("Failed to insert share link: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Share link created successfully",
		"share":   share,
		"pin":     pin,
	})
}

func (s *SharingService) ListSharesEndpoint(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	shares, err := s.ShareRepo.ListByUsername(username.(string))
	if err != nil {
		log.Printf("Failed to query shares: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if shares == nil {
		shares = []model.SharingInfo{}
	}

	for i := range shares {
		fullPath, err := util.SanitizeRepoPath(s.BaseDir, shares[i].Path)
		if err == nil {
			if stat, statErr := os.Stat(fullPath); statErr == nil {
				shares[i].IsDir = stat.IsDir()
			} else {
				log.Printf("Failed to stat share path: %s, error: %v", fullPath, statErr)
			}
		} else {
			log.Printf("Failed to sanitize share path: %s, error: %v", shares[i].Path, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"shares": shares})
}

func (s *SharingService) ToggleShareBlockedEndpoint(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	share, err := s.ShareRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if share.Username != username.(string) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	newBlocked := !share.Blocked
	if err := s.ShareRepo.UpdateBlockedStatus(id, username.(string), newBlocked); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blocked status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated", "blocked": newBlocked})
}

func (s *SharingService) DeleteShareEndpoint(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	rowsAffected, err := s.ShareRepo.Delete(id, username.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found or already deleted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Share link deleted"})
}
