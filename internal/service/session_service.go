package service

import (
	"net/http"
	"time"

	"go-file-server/internal/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type SessionInfo struct {
	FamilyID   string `json:"family_id"`
	DeviceID   string `json:"device_id"`
	DeviceInfo string `json:"device_info"`
	IPAddress  string `json:"ip_address"`
	CreatedAt  string `json:"created_at"`
	ExpiresAt  string `json:"expires_at"`
}

// GetSessions returns active sessions for the current user.
// @Summary      Get Active Sessions
// @Description  List all active refresh token sessions for the authenticated user.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {array}   SessionInfo
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/user/me/sessions [get]
func (s *UserService) GetSessions(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessions, err := s.TokenRepo.GetActiveSessions(username)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	var sessionInfos []SessionInfo
	for _, s := range sessions {
		sessionInfos = append(sessionInfos, SessionInfo{
			FamilyID:   s.FamilyID,
			DeviceID:   s.DeviceID,
			DeviceInfo: s.DeviceInfo,
			IPAddress:  s.IPAddress,
			CreatedAt:  s.CreatedAt.Format(time.RFC3339),
			ExpiresAt:  s.ExpiresAt.Format(time.RFC3339),
		})
	}

	if sessionInfos == nil {
		sessionInfos = []SessionInfo{}
	}

	c.JSON(http.StatusOK, sessionInfos)
}

// RevokeSession revokes a specific session (family) by password confirmation.
// @Summary      Revoke Session
// @Description  Revoke a specific refresh token family. Requires password confirmation.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        family_id  path      string  true  "Session family ID"
// @Param        body       body      object{password=string}  true  "Password confirmation"
// @Success      200        {object}  map[string]interface{}
// @Failure      400        {object}  map[string]interface{}
// @Failure      401        {object}  map[string]interface{}
// @Router       /api/user/me/sessions/{family_id} [delete]
func (s *UserService) RevokeSession(c *gin.Context) {
	username := c.GetString("username")
	familyID := c.Param("family_id")

	if username == "" || familyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}

	user, err := s.UserRepo.GetByUsername(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}

	rowsAffected, err := s.TokenRepo.RevokeByUsernameAndFamilyID(username, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	middleware.RevokedSessionsCache.Set(familyID, true, 20*time.Minute)

	c.JSON(http.StatusOK, gin.H{"success": true})
}
