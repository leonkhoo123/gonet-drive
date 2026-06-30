package controller

import (
	"go-file-server/internal/service"
	"github.com/leonkhoo123/gonet-auth/ratelimit"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

	"github.com/gin-gonic/gin"
)

var shareVerifyLimiter = ratelimit.NewIPRateLimiter(5, 5)

// PublicShareRoutes handles the initial PIN verification routes that don't need authentication
func PublicShareRoutes(router *gin.Engine, sharingService *service.SharingService) {
	api := router.Group("/api/share")
	{
		api.POST("/verify", authgin.RateLimitMiddleware(shareVerifyLimiter), sharingService.VerifySharePINEndpoint)
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
