package service

import (
	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/util"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *UserService) Logout(c *gin.Context, cfg *config.CloudConfig) {
	// Revoke refresh token in DB if provided
	if refreshToken, err := util.GetRefreshToken(c, cfg); err == nil {
		hashedRefreshToken := middleware.HashToken(refreshToken)
		if rt, err := s.TokenRepo.GetByTokenHash(hashedRefreshToken); err == nil {
			middleware.RevokedSessionsCache.Set(rt.FamilyID, true, 20*time.Minute)
			s.TokenRepo.RevokeByFamilyID(rt.FamilyID)
		}
	}

	// Clear access_token
	util.ClearAccessToken(c, cfg)

	// Clear refresh_token
	util.ClearRefreshToken(c, cfg)

	// Clear legacy token just in case
	util.ClearLegacyToken(c, cfg.Auth.TokenName)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
