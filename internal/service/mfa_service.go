package service

import (
	"net/http"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/model"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/pquerna/otp/totp"
)

var mfaFailedCache = cache.New(15*time.Minute, 30*time.Minute)

func (s *UserService) SetupMFA(c *gin.Context, cfg *config.CloudConfig) {
	username := c.GetString("username")

	user, err := s.UserRepo.GetByUsername(username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found or DB error", "details": err.Error()})
		return
	}
	if user.MFAEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA already enabled"})
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      cfg.Auth.TokenName,
		AccountName: username,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP secret"})
		return
	}

	secret := key.Secret()
	url := key.URL()

	if err := s.UserRepo.UpdateMFASecret(username, secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"url":    url,
	})
}

type MFAVerifyRequest struct {
	Code     string `json:"code"`
	DeviceID string `json:"device_id"`
}

func (s *UserService) EnableMFA(c *gin.Context) {
	username := c.GetString("username")

	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := s.UserRepo.GetByUsername(username)
	if err != nil || user.MFAEnabled || user.MFASecret == nil || *user.MFASecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	valid := totp.Validate(req.Code, *user.MFASecret)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}

	if err := s.UserRepo.EnableMFA(username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "MFA enabled successfully"})
}

func (s *UserService) VerifyLoginMFA(c *gin.Context, cfg *config.CloudConfig) {
	// The user should have an mfa_pending cookie, which is just an access token but we must validate it carefully
	tokenStr, err := util.GetMfaPendingToken(c, cfg)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing pre-auth token"})
		return
	}

	token, err := middleware.ValidateTokenString(tokenStr, cfg)
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid pre-auth token"})
		return
	}

	claims, ok := token.Claims.(*middleware.AccessTokenClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	if !claims.IsPreAuth {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid pre-auth token"})
		return
	}

	jti := claims.ID
	if jti != "" {
		if _, used := mfaFailedCache.Get("used_jti_" + jti); used {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "pre-auth token already used"})
			return
		}
	}

	username := claims.Username

	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := s.UserRepo.GetByUsername(username)
	if err != nil || !user.MFAEnabled || user.MFASecret == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	cacheKey := "mfa_lock_" + username
	if _, locked := mfaFailedCache.Get(cacheKey); locked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is locked due to too many failed attempts. Try again later."})
		return
	}

	attemptsKey := "mfa_attempts_" + username

	valid := totp.Validate(req.Code, *user.MFASecret)
	if !valid {
		attempts, err := mfaFailedCache.IncrementInt(attemptsKey, 1)
		if err != nil {
			mfaFailedCache.Set(attemptsKey, 1, cache.DefaultExpiration)
			attempts = 1
		}

		if attempts >= 5 {
			mfaFailedCache.Set(cacheKey, true, 15*time.Minute)
			mfaFailedCache.Delete(attemptsKey)
			c.JSON(http.StatusForbidden, gin.H{"error": "Too many failed attempts. Account locked for 15 minutes."})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}

	// Success, reset failed attempts
	mfaFailedCache.Delete(attemptsKey)

	if jti != "" {
		mfaFailedCache.Set("used_jti_"+jti, true, 15*time.Minute)
	}

	familyID := uuid.New().String()

	newAccessToken, err := middleware.GenerateAccessToken(username, user.TokenVersion, cfg, false, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken := middleware.GenerateRefreshToken()
	hashedRefreshToken := middleware.HashToken(refreshToken)
	tokenID := uuid.New().String()
	expiresAt := time.Now().Add(cfg.Auth.RefreshTokenMaxAge)

	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	rt := &model.RefreshToken{
		ID:         tokenID,
		Username:   username,
		TokenHash:  hashedRefreshToken,
		FamilyID:   familyID,
		DeviceID:   req.DeviceID,
		DeviceInfo: userAgent,
		IPAddress:  ipAddress,
		ExpiresAt:  expiresAt,
	}

	if err := s.TokenRepo.Create(rt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	util.ClearMfaPendingToken(c, cfg)
	util.SetAccessToken(c, cfg, newAccessToken)
	util.SetRefreshToken(c, cfg, refreshToken)

	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}
