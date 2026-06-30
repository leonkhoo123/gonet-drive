package service

import (
	"net/http"

	"go-file-server/internal/config"

	"github.com/leonkhoo123/gonet-auth/mfa"

	"github.com/gin-gonic/gin"
)

// SetupMFA generates a new TOTP secret for the current user.
// @Summary      Setup MFA
// @Description  Generate a TOTP secret and return the secret key and provisioning URL. Requires authentication.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/user/mfa/setup [get]
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

	secret, url, err := mfa.GenerateSecret(cfg.Defaults.ServiceName, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP secret"})
		return
	}

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

// EnableMFA enables MFA for the current user after verifying a TOTP code.
// @Summary      Enable MFA
// @Description  Enable MFA after setting up a TOTP secret via /api/user/mfa/setup.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      MFAVerifyRequest  true  "TOTP code"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /api/user/mfa/enable [post]
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

	if !mfa.ValidateCode(*user.MFASecret, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}

	if err := s.UserRepo.EnableMFA(username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "MFA enabled successfully"})
}
