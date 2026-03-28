package config

import (
	"database/sql"
	"fmt"
	"go-file-server/database"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type DatabaseConfig struct {
	TitleName       string `json:"title_name"`
	AllowedOrigin   string `json:"allowed_origin"`
	UploadChunkSize int64  `json:"upload_chunk_size"`
	StorageLimit    int64  `json:"storage_limit"`
}

var DB *sql.DB
var AppCloudConfig *DatabaseConfig

func RefreshCloudConfigCache() error {
	cfg, err := GetCloudConfig()
	if err != nil {
		return err
	}
	AppCloudConfig = cfg
	return nil
}

func InitDB(workDir string) {
	cloudReserveDir := filepath.Join(workDir, ".cloud_reserve")

	// Check and create .cloud_reserve folder
	if _, err := os.Stat(cloudReserveDir); os.IsNotExist(err) {
		err = os.Mkdir(cloudReserveDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create %s directory: %v", cloudReserveDir, err)
		}
	}

	// Use DB_DIR environment variable for the database location, default to /app/db
	configDir := os.Getenv("DB_DIR")
	if configDir == "" {
		if os.Getenv("APP_ENV") == "local" {
			configDir = "./db"
		} else {
			configDir = "/app/db"
		}
	}

	// Check and create config folder
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create %s directory: %v", configDir, err)
		}
	}

	// Database path
	dbPath := filepath.Join(configDir, "config.db") + "?_busy_timeout=5000&nolock=1"

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.RunMigrations(DB); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Insert default config if table is empty
	insertDefaultQuery := `
	INSERT OR IGNORE INTO cloud_config (config_name, config_type, config_unit, config_value)
	VALUES 
	('service_name', 'string', null, ?),
	('allowed_origin', 'string', null, ?),
	('upload_chunk_size', 'int', 'MB', ?),
	('storage_limit', 'int', 'MB', ?);`

	for i := 0; i < 5; i++ {
		_, err = DB.Exec(insertDefaultQuery, AppConfig.Defaults.ServiceName, AppConfig.Defaults.AllowedOrigin, AppConfig.Defaults.UploadChunkSize, AppConfig.Defaults.StorageLimit)
		if err == nil {
			break
		}
		log.Printf("Failed to insert default config (attempt %d/5): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to insert default config after retries: %v", err)
	}

	// Refresh cache at startup
	if err := RefreshCloudConfigCache(); err != nil {
		log.Printf("Warning: Failed to load cloud config cache: %v", err)

		uploadChunkSize, _ := strconv.ParseInt(AppConfig.Defaults.UploadChunkSize, 10, 64)
		storageLimit, _ := strconv.ParseInt(AppConfig.Defaults.StorageLimit, 10, 64)

		AppCloudConfig = &DatabaseConfig{
			TitleName:       AppConfig.Defaults.ServiceName,
			AllowedOrigin:   AppConfig.Defaults.AllowedOrigin,
			UploadChunkSize: uploadChunkSize * 1024 * 1024,
			StorageLimit:    storageLimit * 1024 * 1024,
		}
	}

	bootstrapAdmin()
}

func bootstrapAdmin() {
	adminUser := os.Getenv("ADMIN_USER")
	adminPass := os.Getenv("ADMIN_PASS")

	if adminUser == "" || adminPass == "" {
		return
	}

	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", adminUser).Scan(&exists)
	if err != nil {
		log.Printf("Failed to check if admin user exists: %v", err)
		return
	}

	if !exists {
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash admin password: %v", err)
			return
		}

		adminID := uuid.New().String()
		_, err = DB.Exec(`
			INSERT INTO users (id, username, password_hash, role, mfa_mandatory)
			VALUES (?, ?, ?, 'superadmin', 1)
		`, adminID, adminUser, string(hashedPass))
		if err != nil {
			log.Printf("Failed to create admin user: %v", err)
		} else {
			log.Printf("Successfully bootstrapped superadmin user: %s", adminUser)
		}
	}
}

func GetCloudConfig() (*DatabaseConfig, error) {
	if DB == nil {
		return nil, sql.ErrNoRows
	}

	query := `SELECT config_name, config_type, config_unit, config_value FROM cloud_config WHERE is_enabled = 1 AND is_deleted = 0`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	uploadChunkSize, _ := strconv.ParseInt(AppConfig.Defaults.UploadChunkSize, 10, 64)
	storageLimit, _ := strconv.ParseInt(AppConfig.Defaults.StorageLimit, 10, 64)

	// Default config
	c := DatabaseConfig{
		TitleName:       AppConfig.Defaults.ServiceName,
		AllowedOrigin:   AppConfig.Defaults.AllowedOrigin,
		UploadChunkSize: uploadChunkSize * 1024 * 1024, // Convert MB to bytes
		StorageLimit:    storageLimit * 1024 * 1024,    // Convert MB to bytes
	}

	for rows.Next() {
		var name, ctype string
		var unit, value sql.NullString
		if err := rows.Scan(&name, &ctype, &unit, &value); err != nil {
			return nil, err
		}

		if !value.Valid {
			continue
		}
		valStr := value.String

		switch name {
		case "service_name":
			c.TitleName = valStr
		case "allowed_origin":
			c.AllowedOrigin = valStr
		case "upload_chunk_size", "storage_limit":
			var size int64
			fmt.Sscanf(valStr, "%d", &size)
			if unit.Valid {
				switch strings.ToLower(unit.String) {
				case "kb":
					size *= 1024
				case "mb":
					size *= 1024 * 1024
				case "gb":
					size *= 1024 * 1024 * 1024
				case "byte", "bytes", "b":
					// already in bytes
				}
			}
			if name == "upload_chunk_size" {
				c.UploadChunkSize = size
			} else {
				c.StorageLimit = size
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &c, nil
}
