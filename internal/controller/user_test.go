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
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *service.UserService, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, _ := testutil.SetupServices(t, db, workDir)

	middleware.ResetLoginLimiter()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, userService)

	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(middleware.JWTAuthMiddleware(cfg))
	}
	authRouter.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "authenticated"})
	})
	authRouter.GET("/me", func(c *gin.Context) {
		username := c.GetString("username")
		user, err := userService.UserRepo.GetByUsername(username)
		role := "user"
		mfaEnabled := false
		mfaMandatory := false
		if err == nil {
			role = user.Role
			mfaEnabled = user.MFAEnabled
			mfaMandatory = user.MFAMandatory
		}
		superAdminUser := cfg.Auth.AdminUser
		isSuperAdmin := username == superAdminUser
		mfaSetupRequired := !mfaEnabled && mfaMandatory
		c.JSON(http.StatusOK, gin.H{
			"username":           username,
			"role":               role,
			"is_super_admin":     isSuperAdmin,
			"mfa_setup_required": mfaSetupRequired,
		})
	})
	authRouter.GET("/me/sessions", userService.GetSessions)
	authRouter.DELETE("/me/sessions/:family_id", userService.RevokeSession)

	return router, cfg, userService, db
}

func TestGetMe_Success(t *testing.T) {
	router, _, _, db := setupUserRouter(t)
	testutil.CreateTestUser(t, db, "meuser", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "meuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/me", nil, accessCookie)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "meuser", resp["username"])
	assert.Equal(t, "user", resp["role"])
	assert.Equal(t, false, resp["is_super_admin"])
	assert.Equal(t, false, resp["mfa_setup_required"])
}

func TestGetMe_Unauthenticated(t *testing.T) {
	router, _, _, _ := setupUserRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	testutil.AssertAuthError(t, rec, http.StatusUnauthorized)
}

func TestGetStatus_Authenticated(t *testing.T) {
	router, _, _, db := setupUserRouter(t)
	testutil.CreateTestUser(t, db, "statususer", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "statususer", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/status", nil, accessCookie)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "authenticated", resp["message"])
}

func TestGetStatus_Unauthenticated(t *testing.T) {
	router, _, _, _ := setupUserRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	testutil.AssertAuthError(t, rec, http.StatusUnauthorized)
}

func TestGetSessions_Authenticated(t *testing.T) {
	router, _, _, db := setupUserRouter(t)
	testutil.CreateTestUser(t, db, "sessuser", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "sessuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/me/sessions", nil, accessCookie)
	assert.Equal(t, http.StatusOK, rec.Code)

	var sessions []map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sessions))
	assert.NotNil(t, sessions, "sessions response should be a valid JSON array")
	assert.GreaterOrEqual(t, len(sessions), 1, "should have at least one active session after login")
	assert.NotEmpty(t, sessions[0]["family_id"], "session should have a family_id")
}

func TestRevokeSession_OwnSession(t *testing.T) {
	router, _, _, db := setupUserRouter(t)
	testutil.CreateTestUser(t, db, "revokeusr", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "revokeusr", "pass123")

	sessionsRec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/me/sessions", nil, accessCookie)
	require.Equal(t, http.StatusOK, sessionsRec.Code)

	var sessions []map[string]interface{}
	require.NoError(t, json.Unmarshal(sessionsRec.Body.Bytes(), &sessions))
	require.GreaterOrEqual(t, len(sessions), 1)
	familyID := sessions[0]["family_id"].(string)
	require.NotEmpty(t, familyID)

	revokeBody := map[string]string{"password": "pass123"}
	jsonBody, err := json.Marshal(revokeBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/me/sessions/"+familyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["success"])
}

func TestRevokeSession_NotOwn(t *testing.T) {
	router, _, _, db := setupUserRouter(t)

	testutil.CreateTestUser(t, db, "userA", "passA", "user")
	testutil.CreateTestUser(t, db, "userB", "passB", "user")

	accessCookieA := testutil.LoginAndGetCookie(t, router, "userA", "passA")
	accessCookieB := testutil.LoginAndGetCookie(t, router, "userB", "passB")

	sessionsRec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/me/sessions", nil, accessCookieB)
	require.Equal(t, http.StatusOK, sessionsRec.Code)

	var sessionsB []map[string]interface{}
	require.NoError(t, json.Unmarshal(sessionsRec.Body.Bytes(), &sessionsB))
	require.GreaterOrEqual(t, len(sessionsB), 1)
	familyIDB := sessionsB[0]["family_id"].(string)
	require.NotEmpty(t, familyIDB)

	revokeBody := map[string]string{"password": "passA"}
	jsonBody, err := json.Marshal(revokeBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/me/sessions/"+familyIDB, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(accessCookieA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "session not found", resp["error"])

	statusRec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/me/sessions", nil, accessCookieB)
	assert.Equal(t, http.StatusOK, statusRec.Code)

	var sessionsAfter []map[string]interface{}
	require.NoError(t, json.Unmarshal(statusRec.Body.Bytes(), &sessionsAfter))
	assert.GreaterOrEqual(t, len(sessionsAfter), 1)
	assert.Equal(t, familyIDB, sessionsAfter[0]["family_id"])
}
