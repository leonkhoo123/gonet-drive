package service

import (
	"database/sql"
	"net/http"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// MobileLogin authenticates a user and returns tokens in the JSON response body.
// Mirrors Login() but does not set cookies.
func (s *UserService) MobileLogin(c *gin.Context, cfg *config.CloudConfig) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Prefer X-Device-Id header if body device_id is empty
	if req.DeviceID == "" {
		req.DeviceID = c.GetHeader("X-Device-Id")
	}

	user, err := s.UserRepo.GetByUsername(req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(req.Password))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if user.MFAEnabled {
		token, err := middleware.GenerateAccessToken(req.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, true, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"temp_token":         token,
			"mfa_required":       true,
			"message":            "MFA required",
		})
		return
	}

	familyID := uuid.New().String()

	accessToken, err := middleware.GenerateAccessToken(req.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	refreshToken, err := middleware.GenerateMobileRefreshToken(req.Username, familyID, req.DeviceID, cfg.Auth.RefreshTokenMaxAge, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	hashedRefreshToken := middleware.HashToken(refreshToken)
	tokenID := uuid.New().String()
	expiresAt := time.Now().Add(cfg.Auth.RefreshTokenMaxAge)

	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	rt := &model.RefreshToken{
		ID:         tokenID,
		Username:   req.Username,
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

	mfaSetupRequired := !user.MFAEnabled && user.MFAMandatory
	c.JSON(http.StatusOK, gin.H{
		"access_token":       accessToken,
		"refresh_token":      refreshToken,
		"mfa_required":       false,
		"mfa_setup_required": mfaSetupRequired,
		"message":            "Login successful",
	})
}

// MobileRefresh exchanges a refresh token for a new access token pair.
// Mirrors RefreshToken() but accepts the refresh token in the request body.
func (s *UserService) MobileRefresh(c *gin.Context, cfg *config.CloudConfig) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token required"})
		return
	}

	hashedRefreshToken := middleware.HashToken(req.RefreshToken)

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
		s.TokenRepo.RevokeByFamilyID(rt.FamilyID)
		middleware.RevokedSessionsCache.Set(rt.FamilyID, true, 20*time.Minute)
		s.UserRepo.IncrementTokenVersion(rt.Username)
		middleware.ClearUserRoleCache(rt.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token compromised, please log in again"})
		return
	}

	err = s.TokenRepo.RevokeByID(rt.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	user, err := s.UserRepo.GetByUsername(rt.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	newAccessToken, err := middleware.GenerateAccessToken(rt.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, rt.FamilyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	newRefreshToken, err := middleware.GenerateMobileRefreshToken(rt.Username, rt.FamilyID, rt.DeviceID, cfg.Auth.RefreshTokenMaxAge, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	})
}

// MobileVerifyLoginMFA completes mobile login by validating a TOTP code.
// Mirrors VerifyLoginMFA() but accepts the pre-auth token in the request body.
func (s *UserService) MobileVerifyLoginMFA(c *gin.Context, cfg *config.CloudConfig) {
	var req struct {
		Code      string `json:"code"`
		TempToken string `json:"temp_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.TempToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "temp_token required"})
		return
	}

	token, err := middleware.ValidateTokenString(req.TempToken, cfg)
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
			mfaFailedCache.Set(attemptsKey, 1, 15*time.Minute)
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

	mfaFailedCache.Delete(attemptsKey)

	if jti != "" {
		mfaFailedCache.Set("used_jti_"+jti, true, 15*time.Minute)
	}

	deviceID := c.GetHeader("X-Device-Id")

	familyID := uuid.New().String()

	newAccessToken, err := middleware.GenerateAccessToken(username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken, err := middleware.GenerateMobileRefreshToken(username, familyID, deviceID, cfg.Auth.RefreshTokenMaxAge, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

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
		DeviceID:   deviceID,
		DeviceInfo: userAgent,
		IPAddress:  ipAddress,
		ExpiresAt:  expiresAt,
	}

	if err := s.TokenRepo.Create(rt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": refreshToken,
		"message":       "MFA verified",
	})
}

// MobileLogout invalidates the current mobile session.
// Extracts the family_id from the Bearer token and revokes the entire family.
func (s *UserService) MobileLogout(c *gin.Context, cfg *config.CloudConfig) {
	claims, err := middleware.ExtractAndValidateBearerAccess(c, cfg)
	if err == nil && claims.FamilyID != "" {
		middleware.RevokedSessionsCache.Set(claims.FamilyID, true, 20*time.Minute)
		s.TokenRepo.RevokeByFamilyID(claims.FamilyID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}
