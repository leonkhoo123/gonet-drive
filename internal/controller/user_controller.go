package controller

import (
	"net/http"

	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/ws"

	"github.com/gin-gonic/gin"
)

// PublicUserRoutes registers all user routes that don't require authentication
func PublicUserRoutes(router *gin.Engine, cfg *config.CloudConfig, userService *service.UserService) {
	api := router.Group("/api")
	{
		api.POST("/login", middleware.LoginRateLimiter(), func(c *gin.Context) {
			userService.Login(c, cfg)
		})
		api.POST("/refresh", func(c *gin.Context) {
			userService.RefreshToken(c, cfg)
		})
		api.POST("/mfa/verify", middleware.LoginRateLimiter(), func(c *gin.Context) {
			userService.VerifyLoginMFA(c, cfg)
		})
		api.POST("/logout", func(c *gin.Context) {
			userService.Logout(c, cfg)
		})
	}
}

// UserRoutes registers all user routes that require authentication
func UserRoutes(router *gin.Engine, cfg *config.CloudConfig, userService *service.UserService, sharingService *service.SharingService, audiobookService *service.AudiobookService, configRepo repository.CloudConfigRepository) {
	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(middleware.JWTAuthMiddleware(cfg))
	} // --- else local bypass jwt ---
	{
		// WebSocket route
		authRouter.GET("/ws", ws.WsHandler)

		authRouter.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "authenticated"})
		})

		authRouter.GET("/mfa/setup", middleware.LoginRateLimiter(), func(c *gin.Context) { userService.SetupMFA(c, cfg) })
		authRouter.POST("/mfa/enable", middleware.LoginRateLimiter(), userService.EnableMFA)

		authRouter.GET("/me", func(c *gin.Context) {
			username := c.GetString("username")
			user, err := userService.UserRepo.GetByUsername(username)

			role := "user"
			mfaEnabled := false
			mfaMandatory := false

			if err == nil {
				role = user.Role
				mfaEnabled = user.MFAEnabled
				mfaMandatory = user.MFAMandatory
			}

			superAdminUser := cfg.Auth.AdminUser
			isSuperAdmin := username == superAdminUser
			mfaSetupRequired := !mfaEnabled && mfaMandatory
			c.JSON(http.StatusOK, gin.H{
				"username":           username,
				"role":               role,
				"is_super_admin":     isSuperAdmin,
				"mfa_setup_required": mfaSetupRequired,
			})
		})

		authRouter.GET("/me/sessions", userService.GetSessions)
		authRouter.DELETE("/me/sessions/:family_id", userService.RevokeSession)

		VideoRoutes(authRouter, cfg)
		PhotoRoutes(authRouter, cfg)
		MusicRoutes(authRouter, cfg)
		AudioBookRoutes(authRouter, cfg, audiobookService)
		DocumentRoutes(authRouter, cfg)
		FilesRoutes(authRouter, cfg)
		ConfigRoutes(authRouter, configRepo)
		ShareRoutes(authRouter, sharingService)

		adminRouter := authRouter.Group("/admin")
		adminRouter.Use(userService.AdminMiddleware())
		{
			adminRouter.GET("/users", userService.GetUsers)
			adminRouter.POST("/users", userService.CreateUser)
			adminRouter.POST("/users/:id/revoke", userService.RevokeSessions)
			adminRouter.DELETE("/users/:id", userService.DeleteUser)
		}
	}
}
