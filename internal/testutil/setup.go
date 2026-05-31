package testutil

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"go-file-server/database"
	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/state"

	"github.com/google/uuid"
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
			AppJwt:             "ON",
			JwtSecret:          "test-secret-key-for-testing-only",
			AdminUser:          "admin",
			AdminPass:          "admin123",
			TokenName:          "file_server_token",
			CookieAccessToken:  "access_token",
			CookieRefreshToken: "refresh_token",
			CookieMfaPending:   "mfa_pending",
			CookieShareJwt:     "shareJwt",
			AccessTokenMaxAge:  15 * time.Minute,
			RefreshTokenMaxAge: 7 * 24 * time.Hour,
			MfaPendingMaxAge:   5 * time.Minute,
			ShareJwtMaxAge:     7 * 24 * time.Hour,
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

	middleware.RevokedSessionsCache.Flush()
	state.ShareStatusCache.Flush()

	t.Cleanup(func() {
		config.DB = nil
		config.AppConfig = nil
		config.AppCloudConfig = nil
		middleware.RevokedSessionsCache.Flush()
		state.ShareStatusCache.Flush()
		db.Close()
	})

	return db
}

// SetupServices constructs the three core service instances from the real constructors.
func SetupServices(t *testing.T, db *sql.DB, workDir string) (*service.UserService, *service.SharingService, repository.CloudConfigRepository) {
	t.Helper()

	userRepo := repository.NewSQLiteUserRepo(db)
	tokenRepo := repository.NewSQLiteRefreshTokenRepo(db)
	userService := service.NewUserService(userRepo, tokenRepo)

	shareRepo := repository.NewSQLiteSharingRepo(db)
	sharingService := service.NewSharingService(shareRepo, workDir)

	configRepo := repository.NewSQLiteCloudConfigRepo(db)

	return userService, sharingService, configRepo
}

// CreateTestUser inserts a user with a bcrypt-hashed password. Returns the created user and raw password.
func CreateTestUser(t *testing.T, db *sql.DB, username, password, role string) *model.User {
	t.Helper()

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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
