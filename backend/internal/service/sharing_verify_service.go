package service

import (
	"database/sql"
	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
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

// SharePINAttemptRecorder is an optional callback to record PIN verification outcomes.
// Set by the controller layer to avoid circular imports.
var SharePINAttemptRecorder func(shareID string, success bool)

// VerifySharePINEndpoint handles POST /api/share/verify.
// @Summary      Verify Share PIN
// @Description  Verify a share link ID and PIN. On success, a share JWT cookie is set for subsequent access.
// @Tags         Share
// @Accept       json
// @Produce      json
// @Param        body  body      VerifyShareRequest  true  "Share ID and PIN"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      410   {object}  map[string]interface{}
// @Router       /api/share/verify [post]
func (s *SharingService) VerifySharePINEndpoint(c *gin.Context) {
	var req VerifyShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Err(c, http.StatusBadRequest, "Invalid request")
		return
	}

	share, err := s.ShareRepo.GetByID(req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			httpx.Err(c, http.StatusNotFound, "Share not found")
			return
		}
		httpx.Err(c, http.StatusInternalServerError, "Database error")
		return
	}

	if share.Blocked {
		httpx.Err(c, http.StatusForbidden, "Share is blocked")
		return
	}

	if !share.IsNeverExpires() && time.Now().After(share.ExpiresAt) {
		httpx.Err(c, http.StatusGone, "Share link has expired")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(share.PinHash), []byte(req.Pin))
	if err != nil {
		if SharePINAttemptRecorder != nil {
			SharePINAttemptRecorder(req.ID, false)
		}
		httpx.Err(c, http.StatusUnauthorized, "Invalid PIN")
		return
	}

	if SharePINAttemptRecorder != nil {
		SharePINAttemptRecorder(req.ID, true)
	}

	tokenDuration := config.AppConfig.Auth.ShareJwtMaxAge
	if !share.IsNeverExpires() {
		timeUntilExpiry := time.Until(share.ExpiresAt)
		if timeUntilExpiry < tokenDuration {
			tokenDuration = timeUntilExpiry
		}
	}

	tokenString, err := middleware.GenerateShareJWT(s.JWT, req.ID, share.Path, share.Authority, tokenDuration)
	if err != nil {
		httpx.Err(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	util.SetShareJwt(c, config.AppConfig, tokenString, int(tokenDuration.Seconds()), req.ID)

	httpx.OK(c, http.StatusOK, gin.H{
		"message":   "Verification successful",
		"authority": share.Authority,
	})
}

// CheckSharePermissionEndpoint handles GET /api/share/check-permission/:id.
// @Summary      Check Share Permission
// @Description  Verify that a valid share JWT exists for the given share ID and the share is still active.
// @Tags         Share
// @Produce      json
// @Param        id   path      string  true  "Share ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/share/check-permission/{id} [get]
func (s *SharingService) CheckSharePermissionEndpoint(c *gin.Context) {
	requestedShareID := c.Param("id")
	if requestedShareID == "" {
		httpx.Err(c, http.StatusBadRequest, "Missing share ID")
		return
	}

	tokenStr, err := util.GetShareJwt(c, config.AppConfig, requestedShareID)
	if err != nil {
		httpx.Err(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	claims, err := middleware.ValidateShareToken(s.JWT, tokenStr)
	if err != nil {
		httpx.Err(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if claims.ShareID != requestedShareID {
		httpx.Err(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	share, err := s.ShareRepo.GetByID(requestedShareID)
	if err != nil {
		httpx.Err(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if share.Blocked {
		httpx.Err(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if !share.IsNeverExpires() && time.Now().After(share.ExpiresAt) {
		httpx.Err(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	httpx.OK(c, http.StatusOK, gin.H{
		"message":   "Permission verified",
		"authority": claims.Authority,
	})
}
