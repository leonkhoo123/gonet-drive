package controller_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/testutil"

	"github.com/leonkhoo123/gonet-auth/auth"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func setupAuthRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *auth.Auth, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	_, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)

	return router, cfg, authInstance, db
}

func setupAuthRouterWithAuthd(t *testing.T) (*gin.Engine, *config.CloudConfig, *auth.Auth, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	_, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

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

	return router, cfg, authInstance, db
}

func loginJSON(t *testing.T, router *gin.Engine, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]string{
		"username": username,
		"password": password,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func loginJSONWithIP(t *testing.T, router *gin.Engine, username, password, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]string{
		"username": username,
		"password": password,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func getCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func getCookieValue(rec *httptest.ResponseRecorder, name string) string {
	c := getCookie(rec, name)
	if c == nil {
		return ""
	}
	return c.Value
}

// ---------- 3.1 Login Tests ----------

func TestLogin_Success(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "loginuser", "pass123", "user")

	rec := loginJSON(t, router, "loginuser", "pass123")
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Login successful", resp["message"])
	assert.Equal(t, false, resp["mfa_required"])

	accessCookie := getCookie(rec, "access_token")
	require.NotNil(t, accessCookie, "access_token cookie should be set")
	assert.NotEmpty(t, accessCookie.Value)
	assert.True(t, accessCookie.HttpOnly)
	assert.Equal(t, "/", accessCookie.Path)

	refreshCookie := getCookie(rec, "refresh_token")
	assert.NotNil(t, refreshCookie, "refresh_token cookie should be set")
	assert.NotEmpty(t, refreshCookie.Value)
}

func TestLogin_InvalidPassword(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "loginuser2", "correctpass", "user")

	rec := loginJSON(t, router, "loginuser2", "wrongpass")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid credentials", resp["error"])
}

func TestLogin_UserNotFound(t *testing.T) {
	router, _, _, _ := setupAuthRouter(t)

	rec := loginJSON(t, router, "nonexistent", "anypass")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid credentials", resp["error"])
}

func TestLogin_MissingBody(t *testing.T) {
	router, _, _, _ := setupAuthRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader([]byte(`{invalid}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_MFARequired(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	user := testutil.CreateTestUser(t, db, "mfauser", "pass123", "user")

	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", "JBSWY3DPEHPK3PXP", user.ID)
	require.NoError(t, err)

	rec := loginJSON(t, router, "mfauser", "pass123")
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "MFA required", resp["message"])
	assert.Equal(t, true, resp["mfa_required"])

	mfaPendingCookie := getCookie(rec, "mfa_pending")
	require.NotNil(t, mfaPendingCookie, "mfa_pending cookie should be set")
	assert.NotEmpty(t, mfaPendingCookie.Value)
}

// ---------- 3.1 Refresh Tests ----------

func TestRefresh_Success(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "refreshuser", "pass123", "user")

	loginRec := loginJSON(t, router, "refreshuser", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	refreshCookie := getCookie(loginRec, "refresh_token")
	require.NotNil(t, refreshCookie)

	req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	req.AddCookie(refreshCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Token refreshed successfully", resp["message"])

	newAccessCookie := getCookie(rec, "access_token")
	require.NotNil(t, newAccessCookie)
	assert.NotEmpty(t, newAccessCookie.Value)

	newRefreshCookie := getCookie(rec, "refresh_token")
	require.NotNil(t, newRefreshCookie)
	assert.NotEmpty(t, newRefreshCookie.Value)
}

func TestRefresh_NoCookie(t *testing.T) {
	router, _, _, _ := setupAuthRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	testutil.AssertAuthError(t, rec, http.StatusUnauthorized)
}

func TestRefresh_Revoked(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "refreshuser2", "pass123", "user")

	loginRec := loginJSON(t, router, "refreshuser2", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	refreshCookie := getCookie(loginRec, "refresh_token")
	require.NotNil(t, refreshCookie)

	req1 := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	req1.AddCookie(refreshCookie)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	req2.AddCookie(refreshCookie)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &resp))
	assert.Equal(t, "token compromised, please log in again", resp["error"])
}

func TestRefresh_RotatesToken(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "refreshuser3", "pass123", "user")

	loginRec := loginJSON(t, router, "refreshuser3", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	oldRefreshCookie := getCookie(loginRec, "refresh_token")
	require.NotNil(t, oldRefreshCookie)
	oldAccessCookie := getCookie(loginRec, "access_token")
	require.NotNil(t, oldAccessCookie)

	req1 := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	req1.AddCookie(oldRefreshCookie)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	newRefreshCookie := getCookie(rec1, "refresh_token")
	require.NotNil(t, newRefreshCookie)
	assert.NotEqual(t, oldRefreshCookie.Value, newRefreshCookie.Value, "refresh token should be rotated")

	newAccessCookie := getCookie(rec1, "access_token")
	require.NotNil(t, newAccessCookie)
	assert.NotEqual(t, oldAccessCookie.Value, newAccessCookie.Value, "access token should be rotated")
}

func TestRefresh_ReuseDetection(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "refreshuser4", "pass123", "user")

	loginRec := loginJSON(t, router, "refreshuser4", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	originalRefreshCookie := getCookie(loginRec, "refresh_token")
	require.NotNil(t, originalRefreshCookie)
	originalRefreshValue := originalRefreshCookie.Value

	req1 := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	req1.AddCookie(originalRefreshCookie)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	oldCookie := &http.Cookie{
		Name:  "refresh_token",
		Value: originalRefreshValue,
		Path:  "/",
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
	req2.AddCookie(oldCookie)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &resp))
	assert.Equal(t, "token compromised, please log in again", resp["error"])
}

// ---------- 3.1 Logout Tests ----------

func TestLogout_Success(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "logoutuser", "pass123", "user")

	loginRec := loginJSON(t, router, "logoutuser", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	refreshCookie := getCookie(loginRec, "refresh_token")
	require.NotNil(t, refreshCookie)

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req.AddCookie(refreshCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["success"])

	accessCookie := getCookie(rec, "access_token")
	require.NotNil(t, accessCookie)
	assert.Equal(t, -1, accessCookie.MaxAge)

	refreshCookie2 := getCookie(rec, "refresh_token")
	require.NotNil(t, refreshCookie2)
	assert.Equal(t, -1, refreshCookie2.MaxAge)
}

func TestLogout_NoSession(t *testing.T) {
	router, _, _, _ := setupAuthRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["success"])
}

func TestLogout_RevokesTokens(t *testing.T) {
	router, _, _, db := setupAuthRouterWithAuthd(t)
	testutil.CreateTestUser(t, db, "logoutuser2", "pass123", "user")

	loginRec := loginJSON(t, router, "logoutuser2", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	accessCookie := getCookie(loginRec, "access_token")
	require.NotNil(t, accessCookie)
	refreshCookie := getCookie(loginRec, "refresh_token")
	require.NotNil(t, refreshCookie)

	reqLogout := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	reqLogout.AddCookie(refreshCookie)
	recLogout := httptest.NewRecorder()
	router.ServeHTTP(recLogout, reqLogout)
	require.Equal(t, http.StatusOK, recLogout.Code)

	reqStatus := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	reqStatus.AddCookie(accessCookie)
	recStatus := httptest.NewRecorder()
	router.ServeHTTP(recStatus, reqStatus)

	assert.Equal(t, http.StatusUnauthorized, recStatus.Code)
}

// ---------- 3.3 Rate Limiter Tests ----------

func TestLoginRateLimiter_Burst(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "ratelimituser", "pass123", "user")

	for i := 0; i < 5; i++ {
		rec := loginJSON(t, router, "ratelimituser", "pass123")
		require.NotEqual(t, http.StatusTooManyRequests, rec.Code, "request %d should not be rate-limited", i+1)
	}

	rec := loginJSON(t, router, "ratelimituser", "pass123")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "too many requests", resp["error"])
}

func TestLoginRateLimiter_Recovery(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "ratelimituser2", "pass123", "user")

	for i := 0; i < 5; i++ {
		rec := loginJSON(t, router, "ratelimituser2", "pass123")
		require.NotEqual(t, http.StatusTooManyRequests, rec.Code, "request %d should not be rate-limited", i+1)
	}

	rec := loginJSON(t, router, "ratelimituser2", "pass123")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// rate limiter uses hardcoded rate.Every(1*time.Second) with burst=5.
	// Wait slightly longer than 1s for a new token to refill.
	time.Sleep(1100 * time.Millisecond)

	recAfter := loginJSON(t, router, "ratelimituser2", "pass123")
	assert.NotEqual(t, http.StatusTooManyRequests, recAfter.Code, "should recover after waiting")
}

func TestLoginRateLimiter_DifferentIPs(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)
	testutil.CreateTestUser(t, db, "ratelimituser3", "pass123", "user")

	for i := 0; i < 5; i++ {
		rec := loginJSONWithIP(t, router, "ratelimituser3", "pass123", "10.0.0.1:1234")
		require.NotEqual(t, http.StatusTooManyRequests, rec.Code, "request %d should not be rate-limited", i+1)
	}

	rec := loginJSONWithIP(t, router, "ratelimituser3", "pass123", "10.0.0.1:1234")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	recOther := loginJSONWithIP(t, router, "ratelimituser3", "pass123", "10.0.0.2:1234")
	assert.NotEqual(t, http.StatusTooManyRequests, recOther.Code, "different IP should not be rate-limited")
}


