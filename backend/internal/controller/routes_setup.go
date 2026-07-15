package controller

import (
	"errors"
	"html"
	"net"
	"net/http"
	"strings"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/ws"

	gonetauth "github.com/leonkhoo123/gonet-auth"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"
	"github.com/leonkhoo123/gonet-auth/auth"
	"github.com/leonkhoo123/gonet-auth/ratelimit"

	"github.com/gin-gonic/gin"
)

var (
	loginStore       = NewMemoryRateLimiterStore(1, 5)
	mobileLoginStore = NewMemoryRateLimiterStore(1, 5)
	mobileMfaStore   = NewMemoryRateLimiterStore(1, 5)
	refreshStore     = NewMemoryRateLimiterStore(5, 10)
	setupStore       = NewMemoryRateLimiterStore(1, 5)
	logoutStore      = NewMemoryRateLimiterStore(5, 10)

	loginLimiter       = ratelimit.NewIPRateLimiter(loginStore)
	mobileLoginLimiter = ratelimit.NewIPRateLimiter(mobileLoginStore)
	mobileMfaLimiter   = ratelimit.NewIPRateLimiter(mobileMfaStore)
	refreshLimiter     = ratelimit.NewIPRateLimiter(refreshStore)
	setupLimiter       = ratelimit.NewIPRateLimiter(setupStore)
	logoutLimiter      = ratelimit.NewIPRateLimiter(logoutStore)
)

// allLimiters tracks all rate limiters for trusted-proxy configuration.
// shareVerifyLimiter (share_controller.go) is added at init time.
var allLimiters = []*ratelimit.IPRateLimiter{
	loginLimiter, mobileLoginLimiter, mobileMfaLimiter, refreshLimiter, setupLimiter, logoutLimiter,
}

// allStores tracks all rate limiter stores for background cleanup.
var allStores = []*MemoryRateLimiterStore{
	loginStore, mobileLoginStore, mobileMfaStore, refreshStore, setupStore, logoutStore,
}

// init registers shareVerifyLimiter (declared in share_controller.go) for
// trusted-proxy configuration to avoid circular initialization.
func init() {
	allLimiters = append(allLimiters, shareVerifyLimiter)
}

// ResetLoginLimiterForTest resets the rate limiters to a fresh state.
// This is intended for use in tests to avoid cross-test interference.
func ResetLoginLimiterForTest() {
	loginStore = NewMemoryRateLimiterStore(1, 5)
	mobileLoginStore = NewMemoryRateLimiterStore(1, 5)
	mobileMfaStore = NewMemoryRateLimiterStore(1, 5)
	refreshStore = NewMemoryRateLimiterStore(5, 10)
	setupStore = NewMemoryRateLimiterStore(1, 5)
	logoutStore = NewMemoryRateLimiterStore(5, 10)

	loginLimiter = ratelimit.NewIPRateLimiter(loginStore)
	mobileLoginLimiter = ratelimit.NewIPRateLimiter(mobileLoginStore)
	mobileMfaLimiter = ratelimit.NewIPRateLimiter(mobileMfaStore)
	refreshLimiter = ratelimit.NewIPRateLimiter(refreshStore)
	setupLimiter = ratelimit.NewIPRateLimiter(setupStore)
	logoutLimiter = ratelimit.NewIPRateLimiter(logoutStore)

	shareVerifyStore = NewMemoryRateLimiterStore(1, 5)
	shareVerifyLimiter = ratelimit.NewIPRateLimiter(shareVerifyStore)

	allStores = []*MemoryRateLimiterStore{loginStore, mobileLoginStore, mobileMfaStore, refreshStore, setupStore, logoutStore}
	allLimiters = []*ratelimit.IPRateLimiter{loginLimiter, mobileLoginLimiter, mobileMfaLimiter, refreshLimiter, setupLimiter, logoutLimiter, shareVerifyLimiter}
}

// StartLimiterCleanup starts background cleanup for all rate limiter stores (Finding 6.1).
func StartLimiterCleanup() {
	interval := 5 * time.Minute
	for _, store := range allStores {
		store.StartCleanupScheduler(interval)
	}
	shareVerifyStore.StartCleanupScheduler(interval)
}

// ConfigureTrustedProxies sets trusted proxy CIDRs on all rate limiters (Finding 6.2).
// In production, an empty or fully-invalid CIDR list is fatal because the limiter
// cannot safely trust X-Forwarded-For without configured proxies.
func ConfigureTrustedProxies(cidrs []string, appEnv string) {
	if len(cidrs) == 0 {
		if appEnv == "production" {
			logger.L.Fatal("TRUSTED_PROXY_CIDRS required in production — rate limiter cannot trust X-Forwarded-For without configured proxies")
		}
		logger.L.Warn("no TRUSTED_PROXY_CIDRS configured — X-Forwarded-For / X-Real-IP are trusted from all sources; configure proxy CIDRs to restrict")
		return
	}
	var valid []string
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			logger.L.Warn("skipping invalid TRUSTED_PROXY_CIDRS entry", "cidr", cidr, "error", err.Error())
			continue
		}
		valid = append(valid, cidr)
	}
	if len(valid) == 0 {
		if appEnv == "production" {
			logger.L.Fatal("TRUSTED_PROXY_CIDRS required in production — all configured entries were invalid")
		}
		logger.L.Warn("all TRUSTED_PROXY_CIDRS entries were invalid — no trusted proxies configured")
		return
	}
	for _, limiter := range allLimiters {
		if err := limiter.SetTrustedProxies(valid...); err != nil {
			logger.L.Warn("failed to set trusted proxies on limiter", "error", err.Error())
		}
	}
}

// sanitizeDeviceInfo HTML-escapes and truncates the User-Agent string (Finding 4.4).
func sanitizeDeviceInfo(raw string) string {
	s := html.EscapeString(raw)
	if len(s) > 512 {
		s = s[:512]
	}
	return s
}

// SanitizeDeviceInfo is the exported test-accessible alias for sanitizeDeviceInfo.
var SanitizeDeviceInfo = sanitizeDeviceInfo

// SecurityHeadersMiddleware sets common HTTP security headers (Finding 7.1).
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Server", "")
		c.Next()
	}
}

// SetupPublicAuthRoutes wires the public web auth endpoints using the library's
// ready-made Gin handlers (login, refresh, MFA verify/recovery, logout) plus the
// first-run admin provisioning + setup status endpoints.
func SetupPublicAuthRoutes(router *gin.Engine, cfg *config.CloudConfig, authInstance *auth.Auth, authCfg *gonetauth.AuthConfig) {
	h := authgin.NewHandlers(authInstance, authCfg)

	api := router.Group("/api")
	{
		api.POST("/login", authgin.RateLimitMiddleware(loginLimiter), h.Login())
		api.POST("/refresh", authgin.RateLimitMiddleware(refreshLimiter), h.Refresh())
		api.POST("/mfa/verify", authgin.RateLimitMiddleware(loginLimiter), h.MFAVerify())
		api.POST("/mfa/recovery", authgin.RateLimitMiddleware(loginLimiter), h.MFARecovery())
		api.POST("/logout", authgin.RateLimitMiddleware(logoutLimiter), h.Logout())

		// First-run admin provisioning + setup status. The path is kept as
		// /api/setup/* to minimize frontend churn while using library handlers.
		api.POST("/setup/admin", authgin.RateLimitMiddleware(setupLimiter), h.ProvisionAdmin())
		api.GET("/setup/status", h.SetupStatus())
	}
}

// SetupMobileAuthRoutes wires /api/mobile/login, /api/mobile/refresh, /api/mobile/mfa/verify, /api/mobile/logout.
// These endpoints return tokens in the JSON body instead of setting httpOnly cookies.
// SetupMobileAuthRoutes wires /api/mobile/* with custom handlers.
// gonet-auth does not (yet) ship a first-class mobile auth flow, so these
// routes use custom inline handlers that call authInstance.Login/Refresh/etc.
// directly and return tokens in the response body instead of cookies.
func SetupMobileAuthRoutes(router *gin.Engine, cfg *config.CloudConfig, authInstance *auth.Auth, authCfg *gonetauth.AuthConfig) {
	api := router.Group("/api/mobile")
	{
		api.POST("/login", authgin.RateLimitMiddleware(mobileLoginLimiter), func(c *gin.Context) {
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
				req.DeviceID, sanitizeDeviceInfo(c.Request.UserAgent()), c.ClientIP(),
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
			case gonetauth.LoginAccountLocked:
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "account locked due to too many failed attempts"})
			case gonetauth.LoginMaxSessionsReached:
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "maximum sessions reached"})
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
					logger.L.Warn("mobile token refresh failed", "err", err)
					c.JSON(http.StatusUnauthorized, gin.H{"error": "token refresh failed"})
				}
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"access_token":  result.AccessToken,
				"refresh_token": result.RefreshToken,
			})
		})

		api.POST("/mfa/verify", authgin.RateLimitMiddleware(mobileMfaLimiter), func(c *gin.Context) {
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

			result, err := authInstance.VerifyMFA(c.Request.Context(), req.TempToken, req.Code, c.GetHeader("X-Device-Id"), sanitizeDeviceInfo(c.Request.UserAgent()), c.ClientIP())
			if err != nil {
				status := http.StatusUnauthorized
				errMsg := "MFA verification failed"
				switch {
				case errors.Is(err, gonetauth.ErrMFALockedOut):
					status = http.StatusTooManyRequests
					errMsg = "too many failed attempts, try again later"
				}
				logger.L.Warn("mobile MFA verification failed", "err", err)
				c.JSON(status, gin.H{"error": errMsg})
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
				authInstance.LogoutFromClaims(c.Request.Context(), claims)
			} else {
				logger.L.Warn("mobile logout: failed to extract bearer token", "err", err)
			}

			// Accept optional refresh_token in body for individual revocation (Finding 4.5)
			var req struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
				if _, err := authInstance.Logout(c.Request.Context(), req.RefreshToken, c.ClientIP()); err != nil {
					logger.L.Warn("mobile logout: refresh token revocation failed", "err", err)
				}
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
func SetupShareFileRoutes(router *gin.Engine, shareRepo repository.SharingRepository, authInstance *auth.Auth) {
	ShareFileRoutes(router, shareRepo, authInstance.JWT)
}

var mfaBypassPaths = []string{"/api/user/me", "/api/user/mfa/setup", "/api/user/mfa/confirm", "/api/logout"}

// SetupAuthenticatedRoutes wires all /api/user/** routes with JWTAuthMiddleware.
func SetupAuthenticatedRoutes(router *gin.Engine, cfg *config.CloudConfig, authInstance *auth.Auth, authCfg *gonetauth.AuthConfig, userService *service.UserService, sharingService *service.SharingService, audiobookService *service.AudiobookService, configRepo repository.CloudConfigRepository, pinnedFolderService *service.PinnedFolderService) {
	h := authgin.NewHandlers(authInstance, authCfg)

	authRouter := router.Group("/api/user")
	// The library middleware short-circuits when JWTOff is set, so it is always
	// attached (no cfg.Auth.AppJwt guard needed).
	authRouter.Use(authgin.JWTAuthMiddleware(authInstance, mfaBypassPaths))
	{
		authRouter.GET("/ws", ws.WsHandler)
		authRouter.GET("/status", userStatusHandler)
		authRouter.POST("/mfa/setup", authgin.RateLimitMiddleware(loginLimiter), h.MFASetup())
		authRouter.POST("/mfa/confirm", authgin.RateLimitMiddleware(loginLimiter), h.MFAConfirm())
		authRouter.GET("/me", userMeHandler())

		authRouter.GET("/me/sessions", h.GetSessions())
		authRouter.POST("/me/sessions/revoke", h.RevokeSession())

		VideoRoutes(authRouter, cfg)
		PhotoRoutes(authRouter, cfg)
		MusicRoutes(authRouter, cfg)
		AudioBookRoutes(authRouter, cfg, audiobookService)
		DocumentRoutes(authRouter, cfg)
		FilesRoutes(authRouter, cfg)
		ShareRoutes(authRouter, sharingService)
		PinnedFolderRoutes(authRouter, pinnedFolderService)

		adminRouter := authRouter.Group("/admin")
		adminRouter.Use(authgin.AdminMiddleware(authInstance))
		{
			adminRouter.GET("/users", h.GetUsers())
			adminRouter.POST("/users", h.CreateUser())
			adminRouter.DELETE("/users/:id", h.DeleteUser())
			adminRouter.POST("/users/:id/revoke-all", h.RevokeAllSessions())
			adminRouter.PUT("/config/logo", UpdateLogo)
			ConfigRoutes(adminRouter, configRepo)

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
	httpx.OK(c, http.StatusOK, gin.H{"message": "authenticated"})
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
func userMeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetString(authgin.KeyUsername)
		role := c.GetString(authgin.KeyRole)
		httpx.OK(c, http.StatusOK, gin.H{
			"username": username,
			"role":     role,
		})
	}
}
