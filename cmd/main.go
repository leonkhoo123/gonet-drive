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
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "go-file-server/docs"
	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/storage"
	"go-file-server/internal/ws"
	"go-file-server/ui"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.Load()

	// Initialize database
	config.InitDB(cfg.Server.FileRoot)
	defer config.DB.Close()

	// Initialize storage manager background scan
	storage.InitStorageManager(cfg.Server.FileRoot)

	router := gin.New()
	router.Use(gin.Logger())
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

	repo := repository.NewSQLiteUserRepo(config.DB)
	tokenRepo := repository.NewSQLiteRefreshTokenRepo(config.DB)
	userService := service.NewUserService(repo, tokenRepo)

	shareRepo := repository.NewSQLiteSharingRepo(config.DB)
	sharingService := service.NewSharingService(shareRepo, cfg.Server.FileRoot)

	audioPathRepo := repository.NewSQLiteAudioPathRepo(config.DB)
	audioProgressRepo := repository.NewSQLiteAudiobookProgressRepo(config.DB)
	audiobookService := service.NewAudiobookService(audioPathRepo, audioProgressRepo)

	configRepo := repository.NewSQLiteCloudConfigRepo(config.DB)

	// Start WebSocket manager
	go ws.Manager.Start()

	// Start sequential file operation worker
	service.StartFileOperationWorker()

	// Public routes
	controller.SetupPublicAuthRoutes(router, cfg, userService)
	controller.SetupMobileAuthRoutes(router, cfg, userService)
	controller.SetupPublicConfigRoutes(router)
	controller.SetupPublicShareRoutes(router, sharingService)
	controller.SetupShareFileRoutes(router, shareRepo)

	// Authenticated routes
	controller.SetupAuthenticatedRoutes(router, cfg, userService, sharingService, audiobookService, configRepo)

	distFS, err := fs.Sub(ui.ReactFiles, "dist")
	if err != nil {
		log.Fatalf("failed to create sub filesystem for ui dist: %v", err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
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
		log.Printf("Starting server on %s", cfg.Server.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting gracefully")
}

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
