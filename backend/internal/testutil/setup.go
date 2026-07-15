package testutil

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"go-file-server/database"
	authpkg "go-file-server/internal/auth"
	"go-file-server/internal/config"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/state"

	gonetauth "github.com/leonkhoo123/gonet-auth"
	"github.com/leonkhoo123/gonet-auth/auth"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var testGlobalMu sync.Mutex

// TestConfig builds a minimal CloudConfig suitable for testing.
func TestConfig(t *testing.T) *config.CloudConfig {
	t.Helper()

	workDir := t.TempDir()

	cfg := &config.CloudConfig{
		Server: config.ServerConfig{
			AppEnv:                     "local",
			FileRoot:                   workDir,
			ListenAddr:                 ":0",
			AllowedOrigins:             []string{"*"},
			ThumbnailMaxConcurrent:     2,
			ThumbnailGenerationTimeout: 30 * time.Second,
		},
		Auth: config.AuthConfig{
			AppJwt:         "ON",
			CookieShareJwt: "shareJwt",
			ShareJwtMaxAge: 7 * 24 * time.Hour,
			SecureMode:     false,
		},
		Defaults: config.AppDefaults{
			ServiceName:     "GoNet Drive Test",
			UploadChunkSize: "5",
			StorageLimit:    "20480",
		},
	}

	return cfg
}

// SetupTestDB creates an in-memory SQLite database, runs all migrations,
// seeds default cloud_config rows, and sets the package-level globals.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	testGlobalMu.Lock()
	t.Cleanup(testGlobalMu.Unlock)

	db, err := sql.Open("sqlite3", ":memory:?_busy_timeout=5000")
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}

	// A ":memory:" database is scoped to a single connection. The gonet-auth
	// audit service writes from a background goroutine concurrently with the
	// request path, which would otherwise make database/sql open a second
	// (empty) in-memory database. Pin the pool to one connection so every
	// query hits the same schema-loaded database.
	db.SetMaxOpenConns(1)

	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	seedDefaults := `
	INSERT OR IGNORE INTO cloud_config (config_name, config_type, config_unit, config_value)
	VALUES 
	('service_name', 'string', null, 'GoNet Drive Test'),
	('upload_chunk_size', 'int', 'MB', '5'),
	('storage_limit', 'int', 'MB', '20480');
	`
	if _, err := db.Exec(seedDefaults); err != nil {
		t.Fatalf("failed to seed default cloud config: %v", err)
	}

	cfg := TestConfig(t)

	config.DB = db
	config.AppConfig = cfg
	config.AppCloudConfig = &config.DatabaseConfig{
		ServiceName:     "GoNet Drive Test",
		UploadChunkSize: 5 * 1024 * 1024,
		StorageLimit:    20480 * 1024 * 1024,
	}

	state.ShareStatusCache.Flush()

	t.Cleanup(func() {
		config.DB = nil
		config.AppConfig = nil
		config.AppCloudConfig = nil
		state.ShareStatusCache.Flush()
		db.Close()
	})

	return db
}

// SetupServices constructs the core service instances and the gonet-auth Auth instance.
func SetupServices(t *testing.T, db *sql.DB, workDir string) (*service.UserService, *service.SharingService, repository.CloudConfigRepository, *auth.Auth, *gonetauth.AuthConfig) {
	t.Helper()

	cfg := config.AppConfig

	userRepo := repository.NewSQLiteUserRepo(db)
	tokenRepo := repository.NewSQLiteRefreshTokenRepo(db)

	shareRepo := repository.NewSQLiteSharingRepo(db)
	configRepo := repository.NewSQLiteCloudConfigRepo(db)

	// Builder-based config mirrors cmd/main.go. The in-memory SQLite DB already
	// has app_settings, so SQLiteSecretStore lets the library auto-generate a
	// test signing secret — no JwtSecret literal.
	authCfg := gonetauth.NewDefaultConfig().
		WithSecretStore(&config.SQLiteSecretStore{DB: db}).
		WithMFA("GoNet Drive Test", 5, 15*time.Minute).
		WithSecureMode(cfg.Auth.SecureMode).
		WithJWTOff(cfg.Auth.AppJwt == "OFF")

	userStore := &authpkg.SQLiteUserStore{Repo: userRepo}
	tokenStore := &authpkg.SQLiteTokenStore{Repo: tokenRepo}

	authInstance := auth.NewAuth(authCfg, userStore, tokenStore,
		auth.WithMFA(userStore),
		auth.WithLockout(userStore),
		auth.WithCacheFactory(func() gonetauth.CacheStore {
			return cache.New(20*time.Minute, 30*time.Minute)
		}),
		auth.WithMFAFailedCacheFactory(func() gonetauth.CacheStore {
			return cache.New(15*time.Minute, 30*time.Minute)
		}),
		auth.WithMFAJTICacheFactory(func() gonetauth.CacheStore {
			return cache.New(20*time.Minute, 30*time.Minute)
		}),
		auth.WithAuditLog(&config.SQLiteAuditLogStore{DB: db}),
	)

	require.NoError(t, authInstance.Start())
	t.Cleanup(authInstance.Shutdown)

	sharingService := service.NewSharingService(shareRepo, workDir, authInstance.JWT)
	userService := service.NewUserService(userRepo, tokenRepo, authInstance)

	return userService, sharingService, configRepo, authInstance, authCfg
}

// CreateTestUser inserts a user with a bcrypt-hashed password. Returns the created user and raw password.
func CreateTestUser(t *testing.T, db *sql.DB, username, password, role string) *model.User {
	t.Helper()

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	require.NoError(t, err)

	now := time.Now()
	user := &model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hashedPass),
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
		TokenVersion: 1,
	}

	_, err = db.Exec(`
		INSERT INTO users (id, username, password_hash, role, mfa_secret, mfa_enabled, mfa_mandatory, storage_quota, created_at, updated_at, failed_attempts, locked_until, token_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, user.Role, user.MFASecret, user.MFAEnabled, user.MFAMandatory,
		user.StorageQuota, user.CreatedAt, user.UpdatedAt, user.FailedAttempts, user.LockedUntil, user.TokenVersion,
	)
	require.NoError(t, err)

	return user
}
