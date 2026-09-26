package repository_test

import (
	"context"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/repository"
	"go-file-server/internal/testutil"

	"github.com/leonkhoo123/gonet-auth/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedAuditLog(t *testing.T, store *config.SQLiteAuditLogStore, eventType, username, ip string, ts time.Time) {
	t.Helper()
	err := store.Log(context.Background(), audit.AuditEvent{
		Type:      audit.AuditEventType(eventType),
		Username:  username,
		IP:        ip,
		Timestamp: ts,
	})
	require.NoError(t, err)
}

func TestAuditLogRepo_List(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteAuditLogStore{DB: db}
	repo := repository.NewSQLiteAuditLogRepo(db)

	now := time.Now().UTC().Truncate(time.Second)
	seedAuditLog(t, store, "login_success", "alice", "10.0.0.1", now.Add(-2*time.Hour))
	seedAuditLog(t, store, "login_failure", "bob", "10.0.0.2", now.Add(-1*time.Hour))
	seedAuditLog(t, store, "logout", "alice", "10.0.0.1", now)

	entries, total, err := repo.List(repository.AuditLogFilter{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, entries, 3)

	// Newest first.
	assert.Equal(t, "logout", entries[0].EventType)
	assert.Equal(t, "alice", entries[0].Username)
	assert.Equal(t, "10.0.0.1", entries[0].IPAddress)
	assert.Equal(t, "login_failure", entries[1].EventType)
	assert.Equal(t, "login_success", entries[2].EventType)

	// Timestamp round-trips through the driver.
	assert.True(t, entries[0].Timestamp.Equal(now), "timestamp %s != %s", entries[0].Timestamp, now)
}

func TestAuditLogRepo_FilterByEventType(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteAuditLogStore{DB: db}
	repo := repository.NewSQLiteAuditLogRepo(db)

	now := time.Now().UTC()
	seedAuditLog(t, store, "login_success", "alice", "10.0.0.1", now)
	seedAuditLog(t, store, "login_failure", "bob", "10.0.0.2", now)

	entries, total, err := repo.List(repository.AuditLogFilter{EventType: "login_failure", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, entries, 1)
	assert.Equal(t, "bob", entries[0].Username)
}

func TestAuditLogRepo_FilterByUsernameSubstring(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteAuditLogStore{DB: db}
	repo := repository.NewSQLiteAuditLogRepo(db)

	now := time.Now().UTC()
	seedAuditLog(t, store, "login_success", "alice", "10.0.0.1", now)
	seedAuditLog(t, store, "login_success", "alicia", "10.0.0.3", now)
	seedAuditLog(t, store, "login_success", "bob", "10.0.0.2", now)

	entries, total, err := repo.List(repository.AuditLogFilter{Username: "ali", Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, entries, 2)
}

func TestAuditLogRepo_Pagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	store := &config.SQLiteAuditLogStore{DB: db}
	repo := repository.NewSQLiteAuditLogRepo(db)

	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 5; i++ {
		seedAuditLog(t, store, "login_success", "user", "10.0.0.1", base.Add(time.Duration(i)*time.Minute))
	}

	page1, total, err := repo.List(repository.AuditLogFilter{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page1, 2)

	page2, total, err := repo.List(repository.AuditLogFilter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, page2, 2)

	// Pages don't overlap.
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestAuditLogRepo_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteAuditLogRepo(db)

	entries, total, err := repo.List(repository.AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.NotNil(t, entries)
	assert.Empty(t, entries)
}
