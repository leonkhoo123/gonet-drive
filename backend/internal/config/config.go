package config

import (
	"go-file-server/internal/assets"
	"go-file-server/internal/logger"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	AppEnv                     string
	DbDir                      string
	FileRoot                   string
	ListenAddr                 string
	Hostname                   string
	AllowedOrigins             []string
	VideoMode                  string
	ThumbnailMaxConcurrent     int
	ThumbnailGenerationTimeout time.Duration
	LogLevel                   string
}

type AuthConfig struct {
	AppJwt string // "ON" | "OFF"

	CookieShareJwt string        // per-share cookie name prefix (fixed default)
	ShareJwtMaxAge time.Duration // share link lifetime (fixed default)

	SecureMode                 bool
	AllowUnsafeUnprotectedMode bool
	TrustedProxyCIDRs          string
}

type AppDefaults struct {
	ServiceName     string
	UploadChunkSize string
	StorageLimit    string
}

type CloudConfig struct {
	Server   ServerConfig
	Auth     AuthConfig
	Defaults AppDefaults
}

var AppConfig *CloudConfig

func Load() *CloudConfig {
	envPath := resolveEnvPath()
	if err := godotenv.Load(envPath); err != nil {
		logger.L.Warn("no .env file found, using built-in defaults")
	}

	allowedOriginsStr := getEnv("ALLOWED_ORIGINS", "http://localhost:5173")
	var origins []string
	for _, o := range strings.Split(allowedOriginsStr, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	// Reject wildcard origins: unsafe when AllowCredentials is used (cookie auth).
	for _, origin := range origins {
		if origin == "*" || origin == "null" {
			logger.L.Fatal("ALLOWED_ORIGINS contains wildcard '*' or 'null' — unsafe with cookie auth; list explicit origins instead", "origin", origin)
		}
	}

	c := &CloudConfig{
		Server: ServerConfig{
			AppEnv:                     getEnv("APP_ENV", "local"),
			DbDir:                      getEnv("DB_DIR", ""),
			FileRoot:                   getEnv("WORK_DIR", "/app/data"), // default path
			ListenAddr:                 getEnv("LISTEN_ADDR", ":8080"),  // default internal port
			Hostname:                   getEnv("VIDEO_HOSTNAME", ""),    // optional override for public URL
			AllowedOrigins:             origins,
			VideoMode:                  getEnv("VIDEO_MODE", "normal"),
			ThumbnailMaxConcurrent:     getEnvInt("THUMBNAIL_MAX_CONCURRENT", 1),
			ThumbnailGenerationTimeout: getEnvDuration("THUMBNAIL_GENERATION_TIMEOUT", 30*time.Second),
			LogLevel:                   getEnv("LOG_LEVEL", "info"),
		},
		Auth: AuthConfig{
			AppJwt:                     getEnv("APP_JWT", ""),
			CookieShareJwt:             "shareJwt",
			ShareJwtMaxAge:             7 * 24 * time.Hour,
			SecureMode:                 getSecureMode(getEnv("APP_ENV", "local")),
			AllowUnsafeUnprotectedMode: getEnv("ALLOW_UNSAFE_UNPROTECTED_MODE", "") == "true",
			TrustedProxyCIDRs:          getEnv("TRUSTED_PROXY_CIDRS", ""),
		},
		Defaults: AppDefaults{
			ServiceName:     getEnv("DEFAULT_SERVICE_NAME", "GoNet Drive"),
			UploadChunkSize: getEnv("DEFAULT_UPLOAD_CHUNK_SIZE", "5"),
			StorageLimit:    getEnv("DEFAULT_STORAGE_LIMIT", "20480"),
		},
	}
	AppConfig = c
	// --- Logging the configuration ---

	logger.L.Info("starting application with configuration", "file_root", c.Server.FileRoot, "listen_addr", c.Server.ListenAddr, "log_level", c.Server.LogLevel)

	if _, err := os.Stat(c.Server.FileRoot); os.IsNotExist(err) {
		logger.L.Fatal("working directory does not exist", "path", c.Server.FileRoot)
	}

	initCloudReserve(c.Server.FileRoot)

	return c
}

func initCloudReserve(workDir string) {
	cloudReserveDir := filepath.Join(workDir, ".cloud_reserve")

	// Check and create .cloud_reserve folder
	if _, err := os.Stat(cloudReserveDir); os.IsNotExist(err) {
		err = os.Mkdir(cloudReserveDir, 0755)
		if err != nil {
			logger.L.Fatal("failed to create directory", "path", cloudReserveDir, "err", err)
		}
	}

	initLogo(cloudReserveDir)
}

func GetLogoPath() string {
	return filepath.Join(AppConfig.Server.FileRoot, ".cloud_reserve", "config", "icon", "logo.png")
}

func EnsureDefaultLogo() {
	logoPath := GetLogoPath()
	iconDir := filepath.Dir(logoPath)

	if _, err := os.Stat(iconDir); os.IsNotExist(err) {
		if err := os.MkdirAll(iconDir, 0755); err != nil {
			logger.L.Error("failed to create directory", "path", iconDir, "err", err)
			return
		}
	}

	if _, err := os.Stat(logoPath); os.IsNotExist(err) {
		if err := os.WriteFile(logoPath, assets.DefaultLogo, 0644); err != nil {
			logger.L.Warn("failed to write default logo", "path", logoPath, "err", err)
		} else {
			logger.L.Info("initialized default logo", "path", logoPath)
		}
	}
}

func initLogo(_ string) {
	EnsureDefaultLogo()
}

// getEnv returns the value from env or fallback if not found
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvDuration returns the parsed duration from env or fallback if not found or invalid
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		} else {
			logger.L.Warn("invalid duration config", "key", key, "value", v, "default", fallback)
		}
	}
	return fallback
}

// getEnvInt returns the parsed int from env or fallback if not found or invalid
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		} else {
			logger.L.Warn("invalid integer config", "key", key, "value", v, "default", fallback)
		}
	}
	return fallback
}

func resolveEnvPath() string {
	if _, err := os.Stat("../.env"); err == nil {
		return "../.env"
	}
	return ".env"
}

func getSecureMode(appEnv string) bool {
	if v := os.Getenv("SECURE_MODE"); v != "" {
		return v == "true" || v == "1"
	}
	return appEnv == "production"
}
