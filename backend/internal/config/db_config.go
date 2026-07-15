package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-file-server/database"
	"go-file-server/internal/logger"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leonkhoo123/gonet-auth/audit"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseConfig struct {
	ServiceName     string `json:"service_name"`
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
	// Use DB_DIR environment variable for the database location, default to /app/db
	configDir := AppConfig.Server.DbDir
	if configDir == "" {
		if AppConfig.Server.AppEnv == "dev" {
			configDir = "../db"
		} else {
			configDir = "/app/db"
		}
	}

	// Check and create config folder
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			logger.L.Fatal("failed to create DB directory", "path", configDir, "err", err)
		}
	}

	// Database path
	dbPath := filepath.Join(configDir, "config.db") + "?_busy_timeout=5000"

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.L.Fatal("failed to connect to database", "err", err)
	}

	// Enable WAL mode for better concurrency (Finding 8.2)
	if _, err := DB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		logger.L.Warn("failed to set WAL journal mode", "err", err)
	}

	// Run migrations
	if err := database.RunMigrations(DB); err != nil {
		logger.L.Fatal("database migrations failed", "err", err)
	}

	// Clean up deprecated configs
	_, _ = DB.Exec("DELETE FROM cloud_config WHERE config_name = 'allowed_origin'")

	// Insert default config if table is empty
	insertDefaultQuery := `
	INSERT OR IGNORE INTO cloud_config (config_name, config_type, config_unit, config_value)
	VALUES 
	('service_name', 'string', null, ?),
	('upload_chunk_size', 'int', 'MB', ?),
	('storage_limit', 'int', 'MB', ?);`

	for i := 0; i < 5; i++ {
		_, err = DB.Exec(insertDefaultQuery, AppConfig.Defaults.ServiceName, AppConfig.Defaults.UploadChunkSize, AppConfig.Defaults.StorageLimit)
		if err == nil {
			break
		}
		logger.L.Warn("failed to insert default config", "attempt", i+1, "max_attempts", 5, "err", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		logger.L.Fatal("failed to insert default config after retries", "err", err)
	}

	// Refresh cache at startup
	if err := RefreshCloudConfigCache(); err != nil {
		logger.L.Warn("failed to load cloud config cache", "err", err)

		uploadChunkSize, _ := strconv.ParseInt(AppConfig.Defaults.UploadChunkSize, 10, 64)
		storageLimit, _ := strconv.ParseInt(AppConfig.Defaults.StorageLimit, 10, 64)

		AppCloudConfig = &DatabaseConfig{
			ServiceName:     AppConfig.Defaults.ServiceName,
			UploadChunkSize: uploadChunkSize * 1024 * 1024,
			StorageLimit:    storageLimit * 1024 * 1024,
		}
	}
}

// SQLiteSecretStore persists secrets in the app_settings table.
type SQLiteSecretStore struct {
	DB *sql.DB
}

func (s *SQLiteSecretStore) LoadSecret(key string) (string, error) {
	var value string
	err := s.DB.QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", os.ErrNotExist
	}
	return value, err
}

func (s *SQLiteSecretStore) SaveSecret(key, value string) error {
	_, err := s.DB.Exec(
		`INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		key, value,
	)
	return err
}

// SQLiteAuditLogStore persists audit events in the audit_logs table.
type SQLiteAuditLogStore struct {
	DB *sql.DB
}

func (s *SQLiteAuditLogStore) Log(_ context.Context, event audit.AuditEvent) error {
	metadataJSON := ""
	if event.Metadata != nil {
		b, err := json.Marshal(event.Metadata)
		if err != nil {
			logger.L.Warn("failed to marshal audit log metadata", "err", err)
		} else {
			metadataJSON = string(b)
		}
	}
	_, err := s.DB.Exec(
		`INSERT INTO audit_logs (event_type, username, ip_address, device_info, family_id, metadata, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(event.Type), event.Username, event.IP, event.DeviceInfo, event.FamilyID, metadataJSON, event.Timestamp,
	)
	return err
}

func (s *SQLiteAuditLogStore) DeleteExpired(_ context.Context, retentionDays int) (int64, error) {
	result, err := s.DB.Exec("DELETE FROM audit_logs WHERE timestamp < datetime('now', ?)", fmt.Sprintf("-%d days", retentionDays))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
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
		ServiceName:     AppConfig.Defaults.ServiceName,
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
			c.ServiceName = valStr
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
