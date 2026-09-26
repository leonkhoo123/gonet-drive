package controller_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/repository"
	"go-file-server/internal/testutil"

	"github.com/leonkhoo123/gonet-auth/audit"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuditRouter(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)
	controller.SetupAuthenticatedRoutes(router, cfg, authInstance, authCfg, userService, nil, nil, nil, nil, repository.NewSQLiteAuditLogRepo(db))

	return router, db
}

func seedControllerAuditEvent(t *testing.T, db *sql.DB, eventType, username string) {
	t.Helper()
	store := &config.SQLiteAuditLogStore{DB: db}
	require.NoError(t, store.Log(context.Background(), audit.AuditEvent{
		Type:      audit.AuditEventType(eventType),
		Username:  username,
		IP:        "10.0.0.1",
		Timestamp: time.Now().UTC(),
	}))
}

type auditLogsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Logs     []map[string]interface{} `json:"logs"`
		Total    int                      `json:"total"`
		Page     int                      `json:"page"`
		PageSize int                      `json:"page_size"`
	} `json:"data"`
}

func TestAuditLogs_NoToken(t *testing.T) {
	router, _ := setupAuditRouter(t)

	req := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/audit-logs", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, req.Code)
}

func TestAuditLogs_NonAdmin(t *testing.T) {
	router, db := setupAuditRouter(t)
	testutil.CreateTestUser(t, db, "regularuser", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "regularuser", "pass123")
	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/audit-logs", nil, accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuditLogs_Admin_ReturnsEntries(t *testing.T) {
	router, db := setupAuditRouter(t)
	testutil.CreateTestUser(t, db, "auditadmin", "pass123", "admin")

	seedControllerAuditEvent(t, db, "session_revoked", "audit-target-a")
	seedControllerAuditEvent(t, db, "session_revoked", "audit-target-b")
	seedControllerAuditEvent(t, db, "token_compromise", "audit-target-a")

	accessCookie := testutil.LoginAndGetCookie(t, router, "auditadmin", "pass123")
	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/audit-logs", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp auditLogsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp.Status)
	assert.GreaterOrEqual(t, resp.Data.Total, 3)
	assert.GreaterOrEqual(t, len(resp.Data.Logs), 3)
}

func TestAuditLogs_Admin_FilterByEventType(t *testing.T) {
	router, db := setupAuditRouter(t)
	testutil.CreateTestUser(t, db, "auditadmin", "pass123", "admin")

	seedControllerAuditEvent(t, db, "session_revoked", "filter-user")
	seedControllerAuditEvent(t, db, "session_revoked", "filter-user")
	seedControllerAuditEvent(t, db, "token_compromise", "filter-user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "auditadmin", "pass123")
	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/audit-logs?event_type=session_revoked", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp auditLogsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Data.Total)
	for _, entry := range resp.Data.Logs {
		assert.Equal(t, "session_revoked", entry["event_type"])
	}
}

func TestAuditLogs_Admin_FilterByUsername(t *testing.T) {
	router, db := setupAuditRouter(t)
	testutil.CreateTestUser(t, db, "auditadmin", "pass123", "admin")

	seedControllerAuditEvent(t, db, "session_revoked", "needle-user")
	seedControllerAuditEvent(t, db, "token_compromise", "needle-user")
	seedControllerAuditEvent(t, db, "session_revoked", "other-user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "auditadmin", "pass123")
	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/audit-logs?username=needle", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp auditLogsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Data.Total)
}

func TestAuditLogs_Admin_Pagination(t *testing.T) {
	router, db := setupAuditRouter(t)
	testutil.CreateTestUser(t, db, "auditadmin", "pass123", "admin")

	for i := 0; i < 5; i++ {
		seedControllerAuditEvent(t, db, "session_revoked", "paged-user")
	}

	accessCookie := testutil.LoginAndGetCookie(t, router, "auditadmin", "pass123")
	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/audit-logs?page=2&page_size=2&username=paged-user", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp auditLogsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 5, resp.Data.Total)
	assert.Equal(t, 2, resp.Data.Page)
	assert.Equal(t, 2, resp.Data.PageSize)
	assert.Len(t, resp.Data.Logs, 2)
}
