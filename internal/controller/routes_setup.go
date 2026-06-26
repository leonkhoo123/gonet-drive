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

// SetupPublicAuthRoutes wires /api/login, /api/refresh, /api/mfa/verify, /api/logout.
func SetupPublicAuthRoutes(router *gin.Engine, cfg *config.CloudConfig, userService *service.UserService) {
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

// SetupMobileAuthRoutes wires /api/mobile/login, /api/mobile/refresh, /api/mobile/mfa/verify, /api/mobile/logout.
// These endpoints return tokens in the JSON body instead of setting httpOnly cookies.
func SetupMobileAuthRoutes(router *gin.Engine, cfg *config.CloudConfig, userService *service.UserService) {
	api := router.Group("/api/mobile")
	{
		api.POST("/login", middleware.LoginRateLimiter(), func(c *gin.Context) {
			userService.MobileLogin(c, cfg)
		})
		api.POST("/refresh", func(c *gin.Context) {
			userService.MobileRefresh(c, cfg)
		})
		api.POST("/mfa/verify", middleware.LoginRateLimiter(), func(c *gin.Context) {
			userService.MobileVerifyLoginMFA(c, cfg)
		})
		api.POST("/logout", func(c *gin.Context) {
			userService.MobileLogout(c, cfg)
		})
	}
}

// SetupPublicConfigRoutes wires /api/config/logo, /api/config/manifest.
func SetupPublicConfigRoutes(router *gin.Engine) {
	api := router.Group("/api")
	PublicConfigRoutes(api)
}

// SetupPublicShareRoutes wires /api/share/verify, /api/share/check-permission/:id.
func SetupPublicShareRoutes(router *gin.Engine, sharingService *service.SharingService) {
	PublicShareRoutes(router, sharingService)
}

// SetupShareFileRoutes wires /api/share/file/** with ShareAuthMiddleware.
func SetupShareFileRoutes(router *gin.Engine, shareRepo repository.SharingRepository) {
	ShareFileRoutes(router, shareRepo)
}

// SetupAuthenticatedRoutes wires all /api/user/** routes with JWTAuthMiddleware.
func SetupAuthenticatedRoutes(router *gin.Engine, cfg *config.CloudConfig, userService *service.UserService, sharingService *service.SharingService, audiobookService *service.AudiobookService, configRepo repository.CloudConfigRepository, pinnedFolderService *service.PinnedFolderService) {
	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(middleware.JWTAuthMiddleware(cfg))
	}
	{
		authRouter.GET("/ws", ws.WsHandler)
		authRouter.GET("/status", userStatusHandler)
		authRouter.GET("/mfa/setup", middleware.LoginRateLimiter(), func(c *gin.Context) { userService.SetupMFA(c, cfg) })
		authRouter.POST("/mfa/enable", middleware.LoginRateLimiter(), userService.EnableMFA)
		authRouter.GET("/me", newUserMeHandler(cfg, userService))

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
		PinnedFolderRoutes(authRouter, pinnedFolderService)

		adminRouter := authRouter.Group("/admin")
		adminRouter.Use(userService.AdminMiddleware())
		{
			adminRouter.GET("/users", userService.GetUsers)
			adminRouter.POST("/users", userService.CreateUser)
			adminRouter.POST("/users/:id/revoke", userService.RevokeSessions)
			adminRouter.DELETE("/users/:id", userService.DeleteUser)
			adminRouter.PUT("/config/logo", UpdateLogo)

			RegisterVideoIntegrityAdminRoutes(adminRouter, cfg)
		}
	}
}

// userStatusHandler godoc
// @Summary      Auth Status Check
// @Description  Check if the current request is properly authenticated.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {object}  map[string]interface{}
// @Router       /api/user/status [get]
func userStatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "authenticated"})
}

// userMeHandler godoc
// @Summary      Current User Info
// @Description  Get information about the currently authenticated user.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {object}  map[string]interface{}
// @Router       /api/user/me [get]
func newUserMeHandler(cfg *config.CloudConfig, userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// SetupAdminRoutes wires /api/user/admin/** routes with JWTAuthMiddleware + AdminMiddleware.
func SetupAdminRoutes(router *gin.Engine, cfg *config.CloudConfig, userService *service.UserService) {
	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(middleware.JWTAuthMiddleware(cfg))
	}
	adminRouter := authRouter.Group("/admin")
	adminRouter.Use(userService.AdminMiddleware())
	{
		adminRouter.GET("/users", userService.GetUsers)
		adminRouter.POST("/users", userService.CreateUser)
		adminRouter.POST("/users/:id/revoke", userService.RevokeSessions)
		adminRouter.DELETE("/users/:id", userService.DeleteUser)
		adminRouter.PUT("/config/logo", UpdateLogo)
		RegisterVideoIntegrityAdminRoutes(adminRouter, cfg)
	}
}

// SetupFileRoutes wires /api/user/files/** routes with JWTAuthMiddleware.
func SetupFileRoutes(router *gin.Engine, cfg *config.CloudConfig) {
	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(middleware.JWTAuthMiddleware(cfg))
	}
	FilesRoutes(authRouter, cfg)
}

// SetupConfigRoutes wires /api/config/** and /api/user/config/** routes.
func SetupConfigRoutes(router *gin.Engine, configRepo repository.CloudConfigRepository) {
	api := router.Group("/api")
	PublicConfigRoutes(api)

	authRouter := router.Group("/api/user")
	ConfigRoutes(authRouter, configRepo)
}

// SetupUserShareRoutes wires /api/share/** public and /api/user/share/** authenticated routes.
func SetupUserShareRoutes(router *gin.Engine, cfg *config.CloudConfig, sharingService *service.SharingService) {
	PublicShareRoutes(router, sharingService)

	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(middleware.JWTAuthMiddleware(cfg))
	}
	ShareRoutes(authRouter, sharingService)
}
