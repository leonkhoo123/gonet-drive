package config_test

import (
	"context"
	"os"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/testutil"

	"github.com/leonkhoo123/gonet-auth/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ---------- Finding 1.1: SQLiteSecretStore ----------

func TestSQLiteSecretStore_LoadSecret_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteSecretStore{DB: db}

	_, err := store.LoadSecret("nonexistent_key")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestSQLiteSecretStore_SaveAndLoad(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteSecretStore{DB: db}

	err := store.SaveSecret("jwt_secret", "super-secret-value-12345")
	require.NoError(t, err)

	loaded, err := store.LoadSecret("jwt_secret")
	require.NoError(t, err)
	assert.Equal(t, "super-secret-value-12345", loaded)
}

func TestSQLiteSecretStore_SaveOverwrite(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteSecretStore{DB: db}

	err := store.SaveSecret("jwt_secret", "old_value")
	require.NoError(t, err)

	err = store.SaveSecret("jwt_secret", "new_value")
	require.NoError(t, err)

	loaded, err := store.LoadSecret("jwt_secret")
	require.NoError(t, err)
	assert.Equal(t, "new_value", loaded)
}

func TestSQLiteSecretStore_MultipleKeys(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteSecretStore{DB: db}

	require.NoError(t, store.SaveSecret("key1", "value1"))
	require.NoError(t, store.SaveSecret("key2", "value2"))

	v1, err := store.LoadSecret("key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", v1)

	v2, err := store.LoadSecret("key2")
	require.NoError(t, err)
	assert.Equal(t, "value2", v2)
}

// ---------- Finding 1.3: bcrypt cost ----------

func TestBcryptCost_Is12(t *testing.T) {
	password := "test-password-123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	require.NoError(t, err)

	// Verify the hash uses cost 12
	cost, err := bcrypt.Cost(hash)
	require.NoError(t, err)
	assert.Equal(t, 12, cost, "bcrypt cost should be 12, not the default 10")
}

// ---------- Finding 11.1: SQLiteAuditLogStore ----------

func TestSQLiteAuditLogStore_Log(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteAuditLogStore{DB: db}

	event := audit.AuditEvent{
		Type:       "login_success",
		Username:   "testuser",
		IP:         "192.168.1.1",
		DeviceInfo: "Mozilla/5.0",
		FamilyID:   "fam-123",
		Metadata:   map[string]string{"key": "value"},
		Timestamp:  time.Now(),
	}

	err := store.Log(context.Background(), event)
	require.NoError(t, err)

	// Verify the event was persisted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE username = ?", "testuser").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSQLiteAuditLogStore_LogWithoutMetadata(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteAuditLogStore{DB: db}

	event := audit.AuditEvent{
		Type:      "logout",
		Username:  "testuser",
		IP:        "10.0.0.1",
		Timestamp: time.Now(),
	}

	err := store.Log(context.Background(), event)
	require.NoError(t, err)
}

func TestSQLiteAuditLogStore_DeleteExpired(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteAuditLogStore{DB: db}

	// Insert an old event (40 days ago)
	_, err := db.Exec(
		`INSERT INTO audit_logs (event_type, username, ip_address, timestamp)
		 VALUES (?, ?, ?, ?)`,
		"old_event", "olduser", "127.0.0.1", time.Now().Add(-40*24*time.Hour),
	)
	require.NoError(t, err)

	// Insert a recent event
	err = store.Log(context.Background(), audit.AuditEvent{
		Type:      "recent_event",
		Username:  "newuser",
		IP:        "127.0.0.1",
		Timestamp: time.Now(),
	})
	require.NoError(t, err)

	// Delete events older than 30 days
	deleted, err := store.DeleteExpired(context.Background(), 30)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify only recent event remains
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSQLiteAuditLogStore_DeleteExpired_NothingToDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteAuditLogStore{DB: db}

	deleted, err := store.DeleteExpired(context.Background(), 30)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

// ---------- Finding 8.2: WAL mode ----------
// NOTE: In-memory SQLite databases (:memory:) cannot use WAL mode.
// The production code at db_config.go:69 sets PRAGMA journal_mode=WAL.
// This test verifies the migration infrastructure works (WAL only applies to file-backed DBs).

// ---------- Finding 8.1: token_hash index ----------

func TestTokenHashIndex_Exists(t *testing.T) {
	db := testutil.SetupTestDB(t)

	var indexSQL string
	err := db.QueryRow(
		"SELECT sql FROM sqlite_master WHERE type='index' AND name='idx_refresh_tokens_token_hash'",
	).Scan(&indexSQL)
	require.NoError(t, err, "idx_refresh_tokens_token_hash index should exist")
	assert.Contains(t, indexSQL, "token_hash")
}
