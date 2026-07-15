// @title           GoNet Drive API
// @version         1.0
// @description     GoNet Drive is a self-hosted cloud file storage and sharing service.
// @termsOfService  https://example.com/terms

// @contact.name   GoNet Drive Support
// @contact.url    https://github.com

// @host      localhost:3333

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the JWT access token.

// @securityDefinitions.apikey  CookieAuth
// @in                          cookie
// @name                        access_token
// @description                 JWT access token stored as a cookie.

package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/h2non/bimg"
	"github.com/patrickmn/go-cache"

	_ "go-file-server/docs"
	authpkg "go-file-server/internal/auth"
	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/repository"
	"go-file-server/internal/schedule"
	"go-file-server/internal/service"
	"go-file-server/internal/state"
	"go-file-server/internal/storage"
	"go-file-server/internal/ws"
	"go-file-server/ui"

	gonetauth "github.com/leonkhoo123/gonet-auth"
	"github.com/leonkhoo123/gonet-auth/auth"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.Load()

	// Initialize structured logger
	logger.Init(parseLogLevel(cfg.Server.LogLevel), cfg.Server.AppEnv)

	// Verify required external binaries exist
	verifyExternalBinaries()

	// Set libvips memory limit to prevent OOM on super-large photos
	// 256 MB is safe for a 2 GB pod while handling any realistic photo size
	bimg.VipsCacheSetMaxMem(256 * 1024 * 1024)
	bimg.VipsCacheSetMax(100)

	// Initialize database
	config.InitDB(cfg.Server.FileRoot)
	defer config.DB.Close()

	// Initialize storage manager background scan
	storage.InitStorageManager(cfg.Server.FileRoot)

	// Security headers middleware (Finding 7.1)
	router := gin.New()
	router.Use(controller.SecurityHeadersMiddleware())
	router.Use(logger.RequestIDMiddleware())
	router.Use(logger.RequestLoggerMiddleware())
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Share-Id", "X-Device-Id"},
	}))

	// Swagger UI (only in dev)
	if cfg.Server.AppEnv == "dev" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	router.GET("/api/health", healthHandler)

	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") && c.Request.URL.Path != "/api/health" {
			state.RecordAPIRequest()
		}
		c.Next()
	})

	repo := repository.NewSQLiteUserRepo(config.DB)
	tokenRepo := repository.NewSQLiteRefreshTokenRepo(config.DB)
	userService := service.NewUserService(repo, tokenRepo, nil)

	shareRepo := repository.NewSQLiteSharingRepo(config.DB)

	audioPathRepo := repository.NewSQLiteAudioPathRepo(config.DB)
	audioProgressRepo := repository.NewSQLiteAudiobookProgressRepo(config.DB)
	audiobookService := service.NewAudiobookService(audioPathRepo, audioProgressRepo)

	configRepo := repository.NewSQLiteCloudConfigRepo(config.DB)

	thumbnailRepo := repository.NewSQLiteThumbnailRepo(config.DB)
	service.SetThumbnailRepo(thumbnailRepo)

	videoIntegrityRepo := repository.NewSQLiteVideoIntegrityRepo(config.DB)
	service.SetVideoIntegrityRepo(videoIntegrityRepo)

	pinnedFolderRepo := repository.NewSQLitePinnedFolderRepo(config.DB)
	pinnedFolderService := service.NewPinnedFolderService(pinnedFolderRepo)

	// Start WebSocket manager
	go ws.Manager.Start()

	// Start sequential file operation worker
	service.StartFileOperationWorker()

	// Build gonet-auth configuration via the builder (Phase 0).
	// The library manages/rotates its own signing secret via the SecretStore;
	// session limit, lockout, and audit retention use library defaults.
	authCfg := gonetauth.NewDefaultConfig().
		WithSecretStore(&config.SQLiteSecretStore{DB: config.DB}).
		WithSecureMode(cfg.Auth.SecureMode).
		WithMFA(cfg.Defaults.ServiceName, 5, 15*time.Minute).
		WithJWTOff(cfg.Auth.AppJwt == "OFF")
	// Same-origin default is SameSiteStrict. If the frontend is served from a
	// different origin in prod, add .WithSameSite(http.SameSiteNoneMode).

	// JWTOff guardrails (Finding 5.2): require explicit opt-in, never in prod.
	if authCfg.JWTOff {
		if cfg.Server.AppEnv == "production" {
			logger.L.Fatal("APP_JWT=OFF is not allowed in production")
		}
		if !cfg.Auth.AllowUnsafeUnprotectedMode {
			logger.L.Fatal("APP_JWT=OFF requires ALLOW_UNSAFE_UNPROTECTED_MODE=true")
		}
		logger.L.Warn("!!! JWTOff mode is enabled — ALL authentication is bypassed !!!")
	}

	// AuditLogStore (Finding 11.1)
	auditLogStore := &config.SQLiteAuditLogStore{DB: config.DB}

	// Structured logFn for gonet-auth
	authLogFn := func(msg string, keyvals ...any) {
		if len(keyvals) == 0 {
			return
		}
		logMsg, ok := keyvals[0].(string)
		if !ok {
			logMsg = fmt.Sprint(keyvals[0])
		}
		switch msg {
		case "debug":
			logger.L.Debug(logMsg, keyvals[1:]...)
		case "info":
			logger.L.Info(logMsg, keyvals[1:]...)
		case "warn":
			logger.L.Warn(logMsg, keyvals[1:]...)
		case "error":
			logger.L.Error(logMsg, keyvals[1:]...)
		}
	}

	userStore := &authpkg.SQLiteUserStore{Repo: repo}
	tokenStore := &authpkg.SQLiteTokenStore{Repo: tokenRepo}

	// NewAuth with functional options. Each cache factory returns an
	// independent cache instance (WithCacheFactory is required).
	authInstance := auth.NewAuth(authCfg, userStore, tokenStore,
		auth.WithMFA(userStore),     // SQLiteUserStore implements UserMFAStore
		auth.WithLockout(userStore), // …and UserLockoutStore
		auth.WithCacheFactory(func() gonetauth.CacheStore {
			return cache.New(20*time.Minute, 30*time.Minute)
		}),
		auth.WithMFAFailedCacheFactory(func() gonetauth.CacheStore {
			return cache.New(15*time.Minute, 30*time.Minute)
		}),
		auth.WithMFAJTICacheFactory(func() gonetauth.CacheStore {
			return cache.New(20*time.Minute, 30*time.Minute)
		}),
		auth.WithLogFn(authLogFn),
		auth.WithAuditLog(auditLogStore),
	)
	userService.AuthInstance = authInstance

	// Share links are signed with the library-managed JWT secret.
	sharingService := service.NewSharingService(shareRepo, cfg.Server.FileRoot, authInstance.JWT)

	// Start() owns token cleanup, audit-log cleanup, and key rotation.
	// It also runs the OnFirstRun callback when no admin exists.
	if err := authInstance.Start(); err != nil {
		logger.L.Fatal("auth initialization failed", "error", err)
	}
	defer authInstance.Shutdown()

	// Start daily thumbnail maintenance scheduler (runs at 4:30 AM)
	schedule.StartThumbnailMaintenanceScheduler(cfg.Server.FileRoot, thumbnailRepo)

	// Configure trusted proxy CIDRs for rate limiters (Finding 6.2)
	var trustedProxyCIDRs []string
	if cfg.Auth.TrustedProxyCIDRs != "" {
		for _, cidr := range strings.Split(cfg.Auth.TrustedProxyCIDRs, ",") {
			if cidr = strings.TrimSpace(cidr); cidr != "" {
				trustedProxyCIDRs = append(trustedProxyCIDRs, cidr)
			}
		}
	}
	controller.ConfigureTrustedProxies(trustedProxyCIDRs, cfg.Server.AppEnv)

	// Start rate limiter cleanup schedulers (Finding 6.1)
	controller.StartLimiterCleanup()

	// Public routes
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)
	controller.SetupMobileAuthRoutes(router, cfg, authInstance, authCfg)
	controller.SetupPublicConfigRoutes(router)
	controller.SetupPublicShareRoutes(router, sharingService)
	controller.SetupShareFileRoutes(router, shareRepo, authInstance)

	// Authenticated routes
	controller.SetupAuthenticatedRoutes(router, cfg, authInstance, authCfg, userService, sharingService, audiobookService, configRepo, pinnedFolderService)

	distFS, err := fs.Sub(ui.ReactFiles, "dist")
	if err != nil {
		logger.L.Fatal("failed to create sub filesystem", "err", err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/api") {
			httpx.Abort(c, http.StatusNotFound, "API route not found")
			return
		}

		if _, err := fs.Stat(distFS, strings.TrimPrefix(path, "/")); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	srv := &http.Server{
		Addr:    cfg.Server.ListenAddr,
		Handler: router,
	}

	// graceful shutdown
	go func() {
		logger.L.Info("starting server", "addr", cfg.Server.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Fatal("server start failed", "err", err)
		}
	}()

	// wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.L.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.L.Fatal("forced shutdown", "err", err)
	}

	logger.L.Info("server exited gracefully")
}

// SecurityHeadersMiddleware sets common HTTP security headers (Finding 7.1).
// Exported from controller package for testability. See controller.SecurityHeadersMiddleware.

// healthHandler godoc
// @Summary      Health Check
// @Description  Get service health status and configuration info.
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/health [get]
func healthHandler(c *gin.Context) {
	cloudConfig := config.AppCloudConfig
	if cloudConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cloud config not available"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":            "OK",
		"service_name":      cloudConfig.ServiceName,
		"upload_chunk_size": cloudConfig.UploadChunkSize,
		"video_mode":        config.AppConfig.Server.VideoMode,
	})
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func verifyExternalBinaries() {
	for _, bin := range []string{"prlimit", "ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			logger.L.Fatal("required binary not found in PATH", "binary", bin, "err", err)
		}
	}
	logger.L.Debug("required external binaries verified", "binaries", []string{"prlimit", "ffmpeg", "ffprobe"})
}
