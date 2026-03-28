package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	FileRoot       string
	ListenAddr     string
	Hostname       string
	AllowedOrigins []string
}

type AuthConfig struct {
	JwtSecret          string
	TokenName          string
	CookieAccessToken  string
	CookieRefreshToken string
	CookieMfaPending   string
	CookieShareJwt     string

	AccessTokenMaxAge  time.Duration
	RefreshTokenMaxAge time.Duration
	MfaPendingMaxAge   time.Duration
	ShareJwtMaxAge     time.Duration
}

type AppDefaults struct {
	ServiceName     string
	AllowedOrigin   string
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
	// Try to load .env from current working directory
	envPath := filepath.Join(".", ".env")
	if err := godotenv.Load(envPath); err != nil {
		fmt.Println("⚠️  No .env file found, using built-in defaults")
	}

	allowedOriginsStr := getEnv("ALLOWED_ORIGINS", "http://192.168.1.9:30334,http://localhost:5173,http://localhost:4173,http://192.168.1.10:30334,http://leon-server.tail571fb3.ts.net:30334,http://leonserver.cc:30334")
	var origins []string
	for _, o := range strings.Split(allowedOriginsStr, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	c := &CloudConfig{
		Server: ServerConfig{
			FileRoot:       getEnv("WORK_DIR", "/app/data"), // default path
			ListenAddr:     getEnv("LISTEN_ADDR", ":8080"),  // default internal port
			Hostname:       getEnv("VIDEO_HOSTNAME", ""),    // optional override for public URL
			AllowedOrigins: origins,
		},
		Auth: AuthConfig{
			JwtSecret:          getEnv("APP_JWTSECRET", ""),
			TokenName:          getEnv("TOKEN_NAME", "file_server_token"),
			CookieAccessToken:  getEnv("COOKIE_ACCESS_TOKEN", "access_token"),
			CookieRefreshToken: getEnv("COOKIE_REFRESH_TOKEN", "refresh_token"),
			CookieMfaPending:   getEnv("COOKIE_MFA_PENDING", "mfa_pending"),
			CookieShareJwt:     getEnv("COOKIE_SHARE_JWT", "shareJwt"),
			AccessTokenMaxAge:  getEnvDuration("ACCESS_TOKEN_MAX_AGE", 15*time.Minute),
			RefreshTokenMaxAge: getEnvDuration("REFRESH_TOKEN_MAX_AGE", 7*24*time.Hour),
			MfaPendingMaxAge:   getEnvDuration("MFA_PENDING_MAX_AGE", 5*time.Minute),
			ShareJwtMaxAge:     getEnvDuration("SHARE_JWT_MAX_AGE", 7*24*time.Hour),
		},
		Defaults: AppDefaults{
			ServiceName:     getEnv("DEFAULT_SERVICE_NAME", "My Cloud Server"),
			AllowedOrigin:   getEnv("DEFAULT_ALLOWED_ORIGIN", "*"),
			UploadChunkSize: getEnv("DEFAULT_UPLOAD_CHUNK_SIZE", "5"),
			StorageLimit:    getEnv("DEFAULT_STORAGE_LIMIT", "20480"),
		},
	}
	AppConfig = c
	// --- Logging the configuration ---

	log.Println("--- Starting application with configuration ---")
	log.Printf("FileRoot:   %s", c.Server.FileRoot)
	log.Printf("ListenAddr: %s", c.Server.ListenAddr)
	log.Println("---------------------------------------------")

	if c.Auth.JwtSecret == "" {
		log.Fatalf("Missing JWT secret .env")
	}

	if _, err := os.Stat(c.Server.FileRoot); os.IsNotExist(err) {
		log.Fatalf("Working directory does not exist (%s)\n", c.Server.FileRoot)
	}

	return c
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
			fmt.Printf("⚠️  Invalid duration for %s: %v, using default %v\n", key, v, fallback)
		}
	}
	return fallback
}
