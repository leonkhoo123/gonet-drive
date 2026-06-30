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
	"go-file-server/internal/testutil"

	"github.com/leonkhoo123/gonet-auth/auth"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupBypassRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *auth.Auth, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	cfg.Auth.AppJwt = "OFF"
	workDir := cfg.Server.FileRoot
	userService, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)

	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(authgin.JWTAuthMiddleware(authInstance, []string{"/api/user/me", "/api/user/mfa/setup", "/api/user/mfa/enable", "/api/logout"}))
	}
	authRouter.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "authenticated"})
	})
	authRouter.GET("/me", func(c *gin.Context) {
		username := c.GetString("username")
		c.JSON(http.StatusOK, gin.H{"username": username})
	})

	adminRouter := authRouter.Group("/admin")
	adminRouter.Use(authgin.AdminMiddleware(authInstance))
	adminRouter.GET("/users", userService.GetUsers)

	return router, cfg, authInstance, db
}

func TestJWTBypass_NoTokenAccess(t *testing.T) {
	router, _, _, _ := setupBypassRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestJWTBypass_NoTokenMe(t *testing.T) {
	router, _, _, _ := setupBypassRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	_, ok := resp["username"]
	assert.True(t, ok, "response should contain username field")
}

func TestJWTBypass_AuthRoutesStillWork(t *testing.T) {
	router, _, _, db := setupBypassRouter(t)
	testutil.CreateTestUser(t, db, "bypassuser", "pass123", "user")

	body := map[string]string{"username": "bypassuser", "password": "pass123"}
	jsonBody, _ := json.Marshal(body)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	assert.Equal(t, http.StatusOK, loginRec.Code)

	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	assert.Equal(t, "Login successful", loginResp["message"])

	hasAccess := false
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "access_token" {
			hasAccess = true
		}
	}
	assert.True(t, hasAccess, "access_token cookie should still be set in bypass mode")
}

func TestJWTBypass_RefreshWorks(t *testing.T) {
	router, _, _, db := setupBypassRouter(t)
	testutil.CreateTestUser(t, db, "bypassuser2", "pass123", "user")

	body := map[string]string{"username": "bypassuser2", "password": "pass123"}
	jsonBody, _ := json.Marshal(body)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	refreshCookie := getCookie(loginRec, "refresh_token")
	require.NotNil(t, refreshCookie, "refresh_token cookie should be set on login")

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	router.ServeHTTP(refreshRec, refreshReq)

	assert.Equal(t, http.StatusOK, refreshRec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(refreshRec.Body.Bytes(), &resp))
	assert.Equal(t, "Token refreshed successfully", resp["message"])

	newAccessCookie := getCookie(refreshRec, "access_token")
	assert.NotNil(t, newAccessCookie, "access_token should be set after refresh")
	assert.NotEmpty(t, newAccessCookie.Value)
}

func TestJWTBypass_LogoutWorks(t *testing.T) {
	router, _, _, db := setupBypassRouter(t)
	testutil.CreateTestUser(t, db, "bypassuser3", "pass123", "user")

	body := map[string]string{"username": "bypassuser3", "password": "pass123"}
	jsonBody, _ := json.Marshal(body)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	refreshCookie := getCookie(loginRec, "refresh_token")
	require.NotNil(t, refreshCookie)

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	logoutReq.AddCookie(refreshCookie)
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)

	assert.Equal(t, http.StatusOK, logoutRec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(logoutRec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["success"])

	accessCookie := getCookie(logoutRec, "access_token")
	require.NotNil(t, accessCookie)
	assert.Equal(t, -1, accessCookie.MaxAge)

	refreshCookie2 := getCookie(logoutRec, "refresh_token")
	require.NotNil(t, refreshCookie2)
	assert.Equal(t, -1, refreshCookie2.MaxAge)
}

func TestJWTBypass_AdminRoutes(t *testing.T) {
	router, _, _, _ := setupBypassRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/admin/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, "admin routes should require auth even in bypass mode")
}
