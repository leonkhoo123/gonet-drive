package controller_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/middleware"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupVideoIntegrityRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *service.UserService, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot

	viRepo := repository.NewSQLiteVideoIntegrityRepo(db)
	service.SetVideoIntegrityRepo(viRepo)

	userService, _, _ := testutil.SetupServices(t, db, workDir)

	middleware.ResetLoginLimiter()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, userService)
	controller.SetupAdminRoutes(router, cfg, userService)

	return router, cfg, userService, db
}

func TestVideoIntegrityScan_NoAuth(t *testing.T) {
	router, _, _, _ := setupVideoIntegrityRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/user/admin/video-integrity/scan", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	testutil.AssertAuthError(t, rec, http.StatusUnauthorized)
}

func TestVideoIntegrityScan_RegularUser(t *testing.T) {
	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "regularuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "regularuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodPost, "/api/user/admin/video-integrity/scan", nil, accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "admin access required", resp["error"])
}

func TestVideoIntegrityScan_Admin(t *testing.T) {
	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodPost, "/api/user/admin/video-integrity/scan", nil, accessCookie)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "integrity-scan", resp["opId"])
	assert.Equal(t, "started", resp["status"])
}

func TestVideoIntegrityScan_DoubleStart(t *testing.T) {
	// Test the concurrency gate at the HTTP level.
	// Pre-set the scan gate to "running" to verify the handler rejects the request.
	defer service.ResetIntegrityScanForTest()

	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	// Simulate a scan already in progress
	service.SetScanRunningForTest(true)

	rec := testutil.MakeAuthRequest(t, router, http.MethodPost, "/api/user/admin/video-integrity/scan", nil, accessCookie)
	assert.Equal(t, http.StatusConflict, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "already running")
}

func TestVideoIntegrityStatus_Admin(t *testing.T) {
	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/video-integrity/status", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["corrupt_count"])
	assert.Equal(t, false, resp["scan_running"])
}

func TestVideoIntegrityStatus_NoAuth(t *testing.T) {
	router, _, _, _ := setupVideoIntegrityRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/admin/video-integrity/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestVideoIntegrityList_Admin(t *testing.T) {
	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	// Populate some entries
	viRepo := service.GetVideoIntegrityRepo()
	require.NotNil(t, viRepo)
	require.NoError(t, viRepo.Upsert("abc", "/v.mp4", "corrupt_avcC", "avc1.000032"))
	require.NoError(t, viRepo.Upsert("def", "/w.mov", "corrupt_avcC", "avc1.000033"))

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/video-integrity/list", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["total"])

	entries, ok := resp["entries"].([]interface{})
	require.True(t, ok)
	assert.Len(t, entries, 2)

	// Verify enriched fields are present
	firstEntry, ok := entries[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, firstEntry["relative_path"])
	assert.NotEmpty(t, firstEntry["file_path"])
	assert.NotEmpty(t, firstEntry["detected_at"])
	assert.NotEmpty(t, firstEntry["last_checked_at"])
	assert.NotEmpty(t, firstEntry["mime_codec_string"])
	// relative_path should be a cleaned path without root prefix
	relPath, _ := firstEntry["relative_path"].(string)
	assert.True(t, len(relPath) > 0, "relative_path should not be empty")
	assert.Equal(t, "corrupt_avcC", firstEntry["issue_type"])
}

func TestVideoIntegrityList_Empty(t *testing.T) {
	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/video-integrity/list", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total"])
}

func TestVideoIntegrityScan_StopWhenRunning(t *testing.T) {
	defer service.ResetIntegrityScanForTest()
	service.SetScanRunningForTest(true)

	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	body := bytes.NewBuffer([]byte(`{"action":"stop"}`))
	rec := testutil.MakeAuthRequest(t, router, http.MethodPost, "/api/user/admin/video-integrity/scan", body, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "integrity-scan", resp["opId"])
	assert.Equal(t, "stopping", resp["status"])
}

func TestVideoIntegrityScan_StopWhenNotRunning(t *testing.T) {
	defer service.ResetIntegrityScanForTest()

	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	body := bytes.NewBuffer([]byte(`{"action":"stop"}`))
	rec := testutil.MakeAuthRequest(t, router, http.MethodPost, "/api/user/admin/video-integrity/scan", body, accessCookie)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "no scan")
}

func TestVideoIntegrityScan_StartWithBody(t *testing.T) {
	// Empty body (no action field) should still start a scan
	defer service.ResetIntegrityScanForTest()

	router, _, _, db := setupVideoIntegrityRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	body := bytes.NewBuffer([]byte(`{}`))
	rec := testutil.MakeAuthRequest(t, router, http.MethodPost, "/api/user/admin/video-integrity/scan", body, accessCookie)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "started", resp["status"])
}
