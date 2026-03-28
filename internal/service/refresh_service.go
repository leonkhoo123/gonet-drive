package service

import (
	"database/sql"
	"net/http"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/model"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *UserService) RefreshToken(c *gin.Context, cfg *config.CloudConfig) {
	refreshToken, err := util.GetRefreshToken(c, cfg)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no refresh token"})
		return
	}

	hashedRefreshToken := middleware.HashToken(refreshToken)

	rt, err := s.TokenRepo.GetByTokenHash(hashedRefreshToken)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if time.Now().After(rt.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token expired"})
		return
	}

	if rt.IsRevoked {
		// REUSE DETECTED! Invalidate the entire family
		s.TokenRepo.RevokeByFamilyID(rt.FamilyID)

		middleware.RevokedSessionsCache.Set(rt.FamilyID, true, 20*time.Minute)

		// We could also increment token_version to invalidate all access tokens immediately
		s.UserRepo.IncrementTokenVersion(rt.Username)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "token compromised, please log in again"})
		return
	}

	// Token is valid. Revoke this one to prevent reuse, and issue a new one in the same family
	err = s.TokenRepo.RevokeByID(rt.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Get user's current token version
	user, err := s.UserRepo.GetByUsername(rt.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	newAccessToken, err := middleware.GenerateAccessToken(rt.Username, user.TokenVersion, cfg, false, rt.FamilyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	newRefreshToken := middleware.GenerateRefreshToken()
	hashedNewRefreshToken := middleware.HashToken(newRefreshToken)
	newTokenID := uuid.New().String()
	newExpiresAt := time.Now().Add(cfg.Auth.RefreshTokenMaxAge)
	newIPAddress := c.ClientIP()

	newRt := &model.RefreshToken{
		ID:         newTokenID,
		Username:   rt.Username,
		TokenHash:  hashedNewRefreshToken,
		FamilyID:   rt.FamilyID,
		DeviceID:   rt.DeviceID,
		DeviceInfo: rt.DeviceInfo,
		IPAddress:  newIPAddress,
		ExpiresAt:  newExpiresAt,
	}

	err = s.TokenRepo.Create(newRt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store new refresh token"})
		return
	}

	// Set cookies
	util.SetAccessToken(c, cfg, newAccessToken)
	util.SetRefreshToken(c, cfg, newRefreshToken)

	c.JSON(http.StatusOK, gin.H{"message": "Token refreshed successfully"})
}
