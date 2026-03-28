package service

import (
	"database/sql"
	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/util"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type VerifyShareRequest struct {
	ID  string `json:"id" binding:"required"`
	Pin string `json:"pin" binding:"required"`
}

// VerifySharePINEndpoint handles POST /api/share/verify
func (s *SharingService) VerifySharePINEndpoint(c *gin.Context) {
	var req VerifyShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	share, err := s.ShareRepo.GetByID(req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Share not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if share.Blocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Share is blocked"})
		return
	}

	if time.Now().After(share.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "Share link has expired"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(share.PinHash), []byte(req.Pin))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid PIN"})
		return
	}

	tokenDuration := config.AppConfig.Auth.ShareJwtMaxAge
	timeUntilExpiry := time.Until(share.ExpiresAt)
	if timeUntilExpiry < tokenDuration {
		tokenDuration = timeUntilExpiry
	}

	secret := config.AppConfig.Auth.JwtSecret
	tokenString, err := middleware.GenerateShareJWT(req.ID, share.Path, share.Authority, tokenDuration, secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	util.SetShareJwt(c, config.AppConfig, tokenString, int(tokenDuration.Seconds()), req.ID)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Verification successful",
		"authority": share.Authority,
	})
}

// CheckSharePermissionEndpoint handles GET /api/share/check-permission/:id
func (s *SharingService) CheckSharePermissionEndpoint(c *gin.Context) {
	requestedShareID := c.Param("id")
	if requestedShareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing share ID"})
		return
	}

	secret := config.AppConfig.Auth.JwtSecret
	tokenStr, err := util.GetShareJwt(c, config.AppConfig, requestedShareID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing share token"})
		return
	}

	token, err := middleware.ValidateShareTokenString(tokenStr, secret)
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired share token"})
		return
	}

	claims, ok := token.Claims.(*middleware.ShareTokenClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid share token claims"})
		return
	}

	if claims.ShareID != requestedShareID {
		c.JSON(http.StatusForbidden, gin.H{"error": "token does not match requested share ID"})
		return
	}

	share, err := s.ShareRepo.GetByID(requestedShareID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Share not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if share.Blocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Share is blocked"})
		return
	}

	if time.Now().After(share.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "Share link has expired"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Permission verified",
		"authority": claims.Authority,
	})
}
