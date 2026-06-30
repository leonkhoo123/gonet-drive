package controller

import (
	"errors"
	"net/http"

	"go-file-server/internal/config"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/ws"

	gonetauth "github.com/leonkhoo123/gonet-auth"
	"github.com/leonkhoo123/gonet-auth/auth"
	"github.com/leonkhoo123/gonet-auth/ratelimit"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

	"github.com/gin-gonic/gin"
)

var loginLimiter = ratelimit.NewIPRateLimiter(1, 5)
var refreshLimiter = ratelimit.NewIPRateLimiter(5, 10)

// ResetLoginLimiterForTest resets the rate limiters to a fresh state.
// This is intended for use in tests to avoid cross-test interference.
func ResetLoginLimiterForTest() {
	loginLimiter = ratelimit.NewIPRateLimiter(1, 5)
	refreshLimiter = ratelimit.NewIPRateLimiter(5, 10)
	shareVerifyLimiter = ratelimit.NewIPRateLimiter(1, 5)
}

// SetupPublicAuthRoutes wires /api/login, /api/refresh, /api/mfa/verify, /api/logout.
func SetupPublicAuthRoutes(router *gin.Engine, cfg *config.CloudConfig, authInstance *auth.Auth, authCfg *gonetauth.AuthConfig) {
	api := router.Group("/api")
	{
		api.POST("/login", authgin.RateLimitMiddleware(loginLimiter), func(c *gin.Context) {
			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
				DeviceID string `json:"device_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
				return
			}

			result, err := authInstance.Login(
				c.Request.Context(),
				req.Username, req.Password,
				req.DeviceID, c.Request.UserAgent(), c.ClientIP(),
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
				return
			}

			switch result.Code {
			case gonetauth.LoginAuthenticated:
				authgin.SetAccessToken(c, authCfg, result.AccessToken)
				authgin.SetRefreshToken(c, authCfg, result.RefreshToken)
				resp := gin.H{"message": "Login successful", "mfa_required": false}
				if result.MFASetupRequired {
					resp["mfa_setup_required"] = true
				}
				c.JSON(http.StatusOK, resp)
			case gonetauth.LoginMFARequired:
				authgin.SetMfaPendingToken(c, authCfg, result.PreAuthToken)
				c.JSON(http.StatusOK, gin.H{"message": "MFA required", "mfa_required": true})
			case gonetauth.LoginInvalidCredentials:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			}
		})

		api.POST("/refresh", authgin.RateLimitMiddleware(refreshLimiter), func(c *gin.Context) {
			refreshToken, err := authgin.GetRefreshToken(c, authCfg)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "no refresh token"})
				return
			}

			result, err := authInstance.RefreshToken(c.Request.Context(), refreshToken, c.ClientIP())
			if err != nil {
				switch {
				case errors.Is(err, gonetauth.ErrTokenCompromised):
					authgin.ClearAccessToken(c, authCfg)
					authgin.ClearRefreshToken(c, authCfg)
					c.JSON(http.StatusUnauthorized, gin.H{"error": "token compromised, please log in again"})
				default:
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				}
				return
			}

			authgin.SetAccessToken(c, authCfg, result.AccessToken)
			authgin.SetRefreshToken(c, authCfg, result.RefreshToken)
			c.JSON(http.StatusOK, gin.H{"message": "Token refreshed successfully"})
		})

		api.POST("/mfa/verify", authgin.RateLimitMiddleware(loginLimiter), func(c *gin.Context) {
			var req struct {
				Code     string `json:"code"`
				DeviceID string `json:"device_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
				return
			}

			preAuthToken, err := authgin.GetMfaPendingToken(c, authCfg)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "missing pre-auth token"})
				return
			}

			result, err := authInstance.VerifyMFA(c.Request.Context(), preAuthToken, req.Code, req.DeviceID, c.Request.UserAgent(), c.ClientIP())
			if err != nil {
				status := http.StatusUnauthorized
				switch {
				case errors.Is(err, gonetauth.ErrMFALockedOut):
					status = http.StatusTooManyRequests
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			authgin.ClearMfaPendingToken(c, authCfg)
			authgin.SetAccessToken(c, authCfg, result.AccessToken)
			authgin.SetRefreshToken(c, authCfg, result.RefreshToken)
			c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
		})

		api.POST("/logout", func(c *gin.Context) {
			refreshToken, _ := authgin.GetRefreshToken(c, authCfg)
			result, err := authInstance.Logout(refreshToken)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed"})
				return
			}
			for _, cookie := range result.ClearedCookies {
				c.SetCookie(cookie, "", -1, "/", "", authCfg.SecureMode, true)
			}
			c.JSON(http.StatusOK, gin.H{"success": true})
		})
	}
}

// SetupMobileAuthRoutes wires /api/mobile/login, /api/mobile/refresh, /api/mobile/mfa/verify, /api/mobile/logout.
// These endpoints return tokens in the JSON body instead of setting httpOnly cookies.
func SetupMobileAuthRoutes(router *gin.Engine, cfg *config.CloudConfig, authInstance *auth.Auth, authCfg *gonetauth.AuthConfig) {
	api := router.Group("/api/mobile")
	{
		api.POST("/login", authgin.RateLimitMiddleware(loginLimiter), func(c *gin.Context) {
			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
				DeviceID string `json:"device_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
				return
			}
			if req.DeviceID == "" {
				req.DeviceID = c.GetHeader("X-Device-Id")
			}

			result, err := authInstance.Login(
				c.Request.Context(),
				req.Username, req.Password,
				req.DeviceID, c.Request.UserAgent(), c.ClientIP(),
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
				return
			}

			switch result.Code {
			case gonetauth.LoginAuthenticated:
				resp := gin.H{
					"access_token":  result.AccessToken,
					"refresh_token": result.RefreshToken,
					"mfa_required":  false,
				}
				if result.MFASetupRequired {
					resp["mfa_setup_required"] = true
				}
				c.JSON(http.StatusOK, resp)
			case gonetauth.LoginMFARequired:
				c.JSON(http.StatusOK, gin.H{
					"temp_token":   result.PreAuthToken,
					"mfa_required": true,
					"message":      "MFA required",
				})
			case gonetauth.LoginInvalidCredentials:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			}
		})

		api.POST("/refresh", authgin.RateLimitMiddleware(refreshLimiter), func(c *gin.Context) {
			var req struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token required"})
				return
			}

			result, err := authInstance.RefreshToken(c.Request.Context(), req.RefreshToken, c.ClientIP())
			if err != nil {
				switch {
				case errors.Is(err, gonetauth.ErrTokenCompromised):
					c.JSON(http.StatusUnauthorized, gin.H{"error": "token compromised, please log in again"})
				default:
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				}
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"access_token":  result.AccessToken,
				"refresh_token": result.RefreshToken,
			})
		})

		api.POST("/mfa/verify", authgin.RateLimitMiddleware(loginLimiter), func(c *gin.Context) {
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

			result, err := authInstance.VerifyMFA(c.Request.Context(), req.TempToken, req.Code, c.GetHeader("X-Device-Id"), c.Request.UserAgent(), c.ClientIP())
			if err != nil {
				status := http.StatusUnauthorized
				switch {
				case errors.Is(err, gonetauth.ErrMFALockedOut):
					status = http.StatusTooManyRequests
				}
				c.JSON(status, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"access_token":  result.AccessToken,
				"refresh_token": result.RefreshToken,
				"message":       "MFA verified",
			})
		})

		api.POST("/logout", func(c *gin.Context) {
			if claims, err := authInstance.JWT.ExtractBearerToken(c.Request); err == nil {
				authInstance.LogoutFromClaims(claims)
			}
			c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
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

var mfaBypassPaths = []string{"/api/user/me", "/api/user/mfa/setup", "/api/user/mfa/enable", "/api/logout"}

// SetupAuthenticatedRoutes wires all /api/user/** routes with JWTAuthMiddleware.
func SetupAuthenticatedRoutes(router *gin.Engine, cfg *config.CloudConfig, authInstance *auth.Auth, authCfg *gonetauth.AuthConfig, userService *service.UserService, sharingService *service.SharingService, audiobookService *service.AudiobookService, configRepo repository.CloudConfigRepository, pinnedFolderService *service.PinnedFolderService) {
	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(authgin.JWTAuthMiddleware(authInstance, mfaBypassPaths))
	}
	{
		authRouter.GET("/ws", ws.WsHandler)
		authRouter.GET("/status", userStatusHandler)
		authRouter.GET("/mfa/setup", authgin.RateLimitMiddleware(loginLimiter), func(c *gin.Context) { userService.SetupMFA(c, cfg) })
		authRouter.POST("/mfa/enable", authgin.RateLimitMiddleware(loginLimiter), userService.EnableMFA)
		authRouter.GET("/me", newUserMeHandler(cfg, userService))

		authRouter.GET("/me/sessions", func(c *gin.Context) {
			sessions, err := authInstance.GetSessions(c.GetString("username"))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
				return
			}
			c.JSON(http.StatusOK, sessions)
		})
		authRouter.DELETE("/me/sessions/:family_id", func(c *gin.Context) {
			var req struct {
				Password string `json:"password"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
				return
			}
			err := authInstance.RevokeSession(c.Request.Context(), c.GetString("username"), c.Param("family_id"), req.Password)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

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
		adminRouter.Use(authgin.AdminMiddleware(authInstance))
		{
			adminRouter.GET("/users", userService.GetUsers)
			adminRouter.POST("/users", userService.CreateUser)
			adminRouter.POST("/users/:id/revoke", func(c *gin.Context) {
				id := c.Param("id")
				currentUsername := c.GetString("username")
				superAdminUser := cfg.Auth.AdminUser

				targetUser, err := userService.UserRepo.GetByID(id)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
					return
				}

				if targetUser.Username == superAdminUser {
					c.JSON(http.StatusForbidden, gin.H{"error": "cannot revoke super admin"})
					return
				}
				if targetUser.Role == "superadmin" && currentUsername != superAdminUser {
					c.JSON(http.StatusForbidden, gin.H{"error": "only the main superadmin can revoke other superadmins"})
					return
				}
				if targetUser.Role == "admin" && currentUsername != superAdminUser {
					c.JSON(http.StatusForbidden, gin.H{"error": "only super admin can revoke an admin"})
					return
				}

				if err := userService.UserRepo.IncrementTokenVersionByID(id); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke access tokens"})
					return
				}
				userService.TokenRepo.RevokeByUsername(targetUser.Username)
				authInstance.ClearUserRoleCache(targetUser.Username)

				c.JSON(http.StatusOK, gin.H{"success": true})
			})
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
