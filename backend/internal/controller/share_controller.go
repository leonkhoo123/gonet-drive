package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"
	"github.com/leonkhoo123/gonet-auth/ratelimit"
	"go-file-server/internal/service"
)

var shareVerifyStore = NewMemoryRateLimiterStore(1, 5)
var shareVerifyLimiter = ratelimit.NewIPRateLimiter(shareVerifyStore)
var sharePINLimiter = NewSharePINLimiter()

func init() {
	// Wire the per-share-ID brute-force tracking into the verification endpoint
	service.SharePINAttemptRecorder = func(shareID string, success bool) {
		if success {
			sharePINLimiter.Reset(shareID)
		} else {
			sharePINLimiter.RecordFailure(shareID)
		}
	}
}

// SharePINRateLimitMiddleware checks per-share-ID brute-force protection.
func SharePINRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check for POST requests with JSON content type
		if c.Request.Method == http.MethodPost && c.ContentType() == "application/json" {
			// Read body and re-inject for downstream handlers
			body, err := c.GetRawData()
			if err == nil && len(body) > 0 {
				// Re-inject the body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewReader(body))

				var req struct {
					ID string `json:"id"`
				}
				if json.Unmarshal(body, &req) == nil && req.ID != "" {
					if sharePINLimiter.IsLockedOut(req.ID) {
						c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many failed attempts for this share, try again later"})
						c.Abort()
						return
					}
				}
			}
		}
		c.Next()
	}
}

// PublicShareRoutes handles the initial PIN verification routes that don't need authentication
func PublicShareRoutes(router *gin.Engine, sharingService *service.SharingService) {
	api := router.Group("/api/share")
	{
		api.POST("/verify", authgin.RateLimitMiddleware(shareVerifyLimiter), SharePINRateLimitMiddleware(), sharingService.VerifySharePINEndpoint)
		api.GET("/check-permission/:id", sharingService.CheckSharePermissionEndpoint)
	}
}

// ShareRoutes handles authenticated management of shares
func ShareRoutes(router *gin.RouterGroup, sharingService *service.SharingService) {
	// Share Management (requires regular JWT authentication)
	manage := router.Group("/share")
	{
		manage.POST("/create", sharingService.CreateShareEndpoint)
		manage.GET("/get-shares", sharingService.ListSharesEndpoint)
		manage.PUT("/:id/toggle-block", sharingService.ToggleShareBlockedEndpoint)
		manage.DELETE("/:id", sharingService.DeleteShareEndpoint)
	}
}
