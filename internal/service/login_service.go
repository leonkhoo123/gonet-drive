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
	"golang.org/x/crypto/bcrypt"
)

var dummyBcryptHash string

func init() {
	// Generate a dummy hash with the default cost to mitigate timing attacks on login
	hash, _ := bcrypt.GenerateFromPassword([]byte("dummy_password"), bcrypt.DefaultCost)
	dummyBcryptHash = string(hash)
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
}

func (s *UserService) Login(c *gin.Context, cfg *config.CloudConfig) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := s.UserRepo.GetByUsername(req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			// Mitigate timing attack by performing a dummy bcrypt comparison
			bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(req.Password))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	} else {
		// Verify bcrypt password
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
	}

	if user.MFAEnabled {
		// Issue pre-auth cookie
		token, err := middleware.GenerateAccessToken(req.Username, user.TokenVersion, cfg, true, "") // Using AT as temporary Pre-Auth token
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		util.SetMfaPendingToken(c, cfg, token)

		c.JSON(http.StatusOK, gin.H{"message": "MFA required", "mfa_required": true})
		return
	}

	familyID := uuid.New().String()

	// generate Access Token
	token, err := middleware.GenerateAccessToken(req.Username, user.TokenVersion, cfg, false, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// Generate Refresh Token
	refreshToken := middleware.GenerateRefreshToken()
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

	// Create Access Cookie
	util.SetAccessToken(c, cfg, token)

	// Create Refresh Cookie
	util.SetRefreshToken(c, cfg, refreshToken)

	mfaSetupRequired := !user.MFAEnabled && user.MFAMandatory
	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "mfa_required": false, "mfa_setup_required": mfaSetupRequired})
}
