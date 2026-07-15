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

	gonetauth "github.com/leonkhoo123/gonet-auth"
	"github.com/leonkhoo123/gonet-auth/auth"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMobileAuthRouter(t *testing.T) (*gin.Engine, *gonetauth.AuthConfig, *auth.Auth, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	_, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()
	controller.SetupMobileAuthRoutes(router, cfg, authInstance, authCfg)

	return router, authCfg, authInstance, db
}

func setupMobileAuthRouterWithProtected(t *testing.T) (*gin.Engine, *gonetauth.AuthConfig, *auth.Auth, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	_, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()
	controller.SetupMobileAuthRoutes(router, cfg, authInstance, authCfg)

	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(authgin.JWTAuthMiddleware(authInstance, []string{"/api/user/me", "/api/user/mfa/setup", "/api/user/mfa/confirm", "/api/logout"}))
	}
	authRouter.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "authenticated"})
	})

	return router, authCfg, authInstance, db
}

func mobileLogin(t *testing.T, router *gin.Engine, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	return mobileLoginWithDevice(t, router, username, password, "")
}

func mobileLoginWithDevice(t *testing.T, router *gin.Engine, username, password, deviceID string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]string{
		"username": username,
		"password": password,
	}
	if deviceID != "" {
		body["device_id"] = deviceID
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if deviceID != "" {
		req.Header.Set("X-Device-Id", deviceID)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func makeMobileBearerRequest(t *testing.T, router *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// ---------- Mobile Login Tests ----------

func TestMobileLogin_Success(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	testutil.CreateTestUser(t, db, "mobileuser", "pass123", "user")

	rec := mobileLogin(t, router, "mobileuser", "pass123")
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["mfa_required"])
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])

	// Verify no cookies are set (mobile returns tokens in body, not cookies)
	for _, c := range rec.Result().Cookies() {
		assert.NotEqual(t, "access_token", c.Name, "access_token cookie should not be set")
		assert.NotEqual(t, "refresh_token", c.Name, "refresh_token cookie should not be set")
	}
}

func TestMobileLogin_WithDeviceID(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	testutil.CreateTestUser(t, db, "mobiledevuser", "pass123", "user")

	rec := mobileLoginWithDevice(t, router, "mobiledevuser", "pass123", "android-phone-123")
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

func TestMobileLogin_InvalidPassword(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	testutil.CreateTestUser(t, db, "mobileuser2", "correctpass", "user")

	rec := mobileLogin(t, router, "mobileuser2", "wrongpass")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid credentials", resp["error"])
}

func TestMobileLogin_UserNotFound(t *testing.T) {
	router, _, _, _ := setupMobileAuthRouter(t)

	rec := mobileLogin(t, router, "nonexistent", "anypass")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid credentials", resp["error"])
}

func TestMobileLogin_MissingBody(t *testing.T) {
	router, _, _, _ := setupMobileAuthRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/mobile/login", bytes.NewReader([]byte(`{invalid}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMobileLogin_MFARequired(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	user := testutil.CreateTestUser(t, db, "mobilemfauser", "pass123", "user")

	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", "JBSWY3DPEHPK3PXP", user.ID)
	require.NoError(t, err)

	rec := mobileLogin(t, router, "mobilemfauser", "pass123")
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "MFA required", resp["message"])
	assert.Equal(t, true, resp["mfa_required"])
	assert.NotEmpty(t, resp["temp_token"])
	assert.Empty(t, resp["access_token"])
	assert.Empty(t, resp["refresh_token"])
}

func TestMobileLogin_MFA_Mandatory_NotSetup(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	user := testutil.CreateTestUser(t, db, "mobilemfamand", "pass123", "user")

	_, err := db.Exec("UPDATE users SET mfa_mandatory = 1 WHERE id = ?", user.ID)
	require.NoError(t, err)

	rec := mobileLogin(t, router, "mobilemfamand", "pass123")
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["mfa_setup_required"])
	assert.NotEmpty(t, resp["access_token"])
}

func TestMobileLogin_RateLimited(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	testutil.CreateTestUser(t, db, "mobileratelimit", "pass123", "user")

	for i := 0; i < 5; i++ {
		rec := mobileLogin(t, router, "mobileratelimit", "pass123")
		require.NotEqual(t, http.StatusTooManyRequests, rec.Code, "request %d should not be rate-limited", i+1)
	}

	rec := mobileLogin(t, router, "mobileratelimit", "pass123")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

// ---------- Mobile Refresh Tests ----------

func TestMobileRefresh_Success(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	testutil.CreateTestUser(t, db, "mobilerefresh", "pass123", "user")

	loginRec := mobileLogin(t, router, "mobilerefresh", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	refreshToken := loginResp["refresh_token"].(string)
	require.NotEmpty(t, refreshToken)

	body := map[string]string{"refresh_token": refreshToken}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

func TestMobileRefresh_TokenRotation(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	testutil.CreateTestUser(t, db, "mobilerefreshrot", "pass123", "user")

	loginRec := mobileLogin(t, router, "mobilerefreshrot", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	oldAccess := loginResp["access_token"].(string)
	oldRefresh := loginResp["refresh_token"].(string)

	body := map[string]string{"refresh_token": oldRefresh}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEqual(t, oldAccess, resp["access_token"], "access token should be rotated")
	assert.NotEqual(t, oldRefresh, resp["refresh_token"], "refresh token should be rotated")
}

func TestMobileRefresh_NoRefreshToken(t *testing.T) {
	router, _, _, _ := setupMobileAuthRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/mobile/refresh", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMobileRefresh_InvalidToken(t *testing.T) {
	router, _, _, _ := setupMobileAuthRouter(t)

	body := map[string]string{"refresh_token": "invalid-token-value"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMobileRefresh_ReuseDetection(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	testutil.CreateTestUser(t, db, "mobilerefreshreuse", "pass123", "user")

	loginRec := mobileLogin(t, router, "mobilerefreshreuse", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	refreshToken := loginResp["refresh_token"].(string)

	body := map[string]string{"refresh_token": refreshToken}
	jsonBody, _ := json.Marshal(body)

	// First refresh — should succeed
	req1 := httptest.NewRequest(http.MethodPost, "/api/mobile/refresh", bytes.NewReader(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusOK, rec1.Code)

	// Reuse the same old refresh token — should detect compromise
	req2 := httptest.NewRequest(http.MethodPost, "/api/mobile/refresh", bytes.NewReader(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &resp))
	assert.Equal(t, "token compromised, please log in again", resp["error"])
}

// ---------- Mobile MFA Verify Tests ----------

func TestMobileMFAVerify_Success(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	user := testutil.CreateTestUser(t, db, "mobilemfaverify", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, err)

	loginRec := mobileLogin(t, router, "mobilemfaverify", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	assert.Equal(t, true, loginResp["mfa_required"])
	tempToken := loginResp["temp_token"].(string)
	require.NotEmpty(t, tempToken)

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	verifyBody := map[string]string{
		"code":       code,
		"temp_token": tempToken,
	}
	jsonBody, _ := json.Marshal(verifyBody)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/mfa/verify", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "MFA verified", resp["message"])
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

func TestMobileMFAVerify_WrongCode(t *testing.T) {
	router, _, _, db := setupMobileAuthRouter(t)
	user := testutil.CreateTestUser(t, db, "mobilemfaverify2", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, err)

	loginRec := mobileLogin(t, router, "mobilemfaverify2", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	tempToken := loginResp["temp_token"].(string)
	require.NotEmpty(t, tempToken)

	verifyBody := map[string]string{
		"code":       "000000",
		"temp_token": tempToken,
	}
	jsonBody, _ := json.Marshal(verifyBody)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/mfa/verify", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "MFA verification failed", resp["error"])
}

func TestMobileMFAVerify_NoTempToken(t *testing.T) {
	router, _, _, _ := setupMobileAuthRouter(t)

	verifyBody := map[string]string{"code": "123456"}
	jsonBody, _ := json.Marshal(verifyBody)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/mfa/verify", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMobileMFAVerify_InvalidTempToken(t *testing.T) {
	router, _, _, _ := setupMobileAuthRouter(t)

	verifyBody := map[string]string{
		"code":       "123456",
		"temp_token": "invalid-token",
	}
	jsonBody, _ := json.Marshal(verifyBody)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/mfa/verify", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMobileMFAVerify_Lockout(t *testing.T) {
	router, _, authInstance, db := setupMobileAuthRouter(t)
	user := testutil.CreateTestUser(t, db, "mobilemfalock", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, err)

	// Generate pre-auth token directly to match web test pattern
	preAuthToken, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, true, "")
	require.NoError(t, err)

	wrongCode := "000000"

	// The library tracks failed MFA attempts per <username, ip>. Using the same
	// IP, the 5th failed attempt (MaxAttempts=5) triggers the per-IP lockout.
	// The mobile MFA rate limiter has burst=5, so all 5 requests pass the limiter.
	const sameIP = "10.0.0.1:1000"
	var lastRec *httptest.ResponseRecorder
	for i := 0; i < 5; i++ {
		verifyBody := map[string]string{
			"code":       wrongCode,
			"temp_token": preAuthToken,
		}
		jsonBody, _ := json.Marshal(verifyBody)
		req := httptest.NewRequest(http.MethodPost, "/api/mobile/mfa/verify", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = sameIP
		lastRec = httptest.NewRecorder()
		router.ServeHTTP(lastRec, req)
	}

	// The final attempt hit the per-IP MFA lockout.
	assert.Equal(t, http.StatusTooManyRequests, lastRec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(lastRec.Body.Bytes(), &resp))
	assert.Equal(t, "too many failed attempts, try again later", resp["error"])
}

func TestMobileMFAVerify_ExpiredPending(t *testing.T) {
	router, cfg, authInstance, db := setupMobileAuthRouter(t)
	user := testutil.CreateTestUser(t, db, "mobilemfaexp", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, err)

	origMfaMaxAge := cfg.Tokens.MfaPending
	cfg.Tokens.MfaPending = -1 * time.Hour
	preAuthToken, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, true, "")
	require.NoError(t, err)
	cfg.Tokens.MfaPending = origMfaMaxAge

	verifyBody := map[string]string{
		"code":       "123456",
		"temp_token": preAuthToken,
	}
	jsonBody, _ := json.Marshal(verifyBody)
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/mfa/verify", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------- Mobile Logout Tests ----------

func TestMobileLogout_Success(t *testing.T) {
	router, _, _, db := setupMobileAuthRouterWithProtected(t)
	testutil.CreateTestUser(t, db, "mobilelogout", "pass123", "user")

	loginRec := mobileLogin(t, router, "mobilelogout", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	accessToken := loginResp["access_token"].(string)
	require.NotEmpty(t, accessToken)

	// Verify access token works
	statusReq := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	statusReq.Header.Set("Authorization", "Bearer "+accessToken)
	statusRec := httptest.NewRecorder()
	router.ServeHTTP(statusRec, statusReq)
	require.Equal(t, http.StatusOK, statusRec.Code)

	// Logout
	req := httptest.NewRequest(http.MethodPost, "/api/mobile/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Logged out", resp["message"])

	// Verify token is now rejected
	statusReq2 := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	statusReq2.Header.Set("Authorization", "Bearer "+accessToken)
	statusRec2 := httptest.NewRecorder()
	router.ServeHTTP(statusRec2, statusReq2)
	assert.Equal(t, http.StatusUnauthorized, statusRec2.Code)
}

func TestMobileLogout_NoAuth(t *testing.T) {
	router, _, _, _ := setupMobileAuthRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/mobile/logout", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// MobileLogout does not fail even without auth - returns success
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Logged out", resp["message"])
}

func TestMobileLogout_InvalidToken(t *testing.T) {
	router, _, _, _ := setupMobileAuthRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/mobile/logout", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// MobileLogout is lenient — silently succeeds
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ---------- Mobile Auth with Bearer Token (End-to-End) ----------

func TestMobileAuth_BearerTokenAccess(t *testing.T) {
	router, _, _, db := setupMobileAuthRouterWithProtected(t)
	testutil.CreateTestUser(t, db, "mobilebearer", "pass123", "user")

	loginRec := mobileLogin(t, router, "mobilebearer", "pass123")
	require.Equal(t, http.StatusOK, loginRec.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	accessToken := loginResp["access_token"].(string)

	// Access protected route with Bearer token
	rec := makeMobileBearerRequest(t, router, http.MethodGet, "/api/user/status", accessToken, nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	var statusResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &statusResp))
	assert.Equal(t, "authenticated", statusResp["message"])
}

func TestMobileAuth_BearerTokenWithDeviceID(t *testing.T) {
	router, _, _, db := setupMobileAuthRouterWithProtected(t)
	testutil.CreateTestUser(t, db, "mobilebearerdev", "pass123", "user")

	loginRec := mobileLoginWithDevice(t, router, "mobilebearerdev", "pass123", "tablet-456")
	require.Equal(t, http.StatusOK, loginRec.Code)
	var loginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	accessToken := loginResp["access_token"].(string)

	// Access with Bearer token + X-Device-Id
	req := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Device-Id", "tablet-456")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMobileAuth_PreAuthTokenRejectedOnProtected(t *testing.T) {
	router, _, authInstance, db := setupMobileAuthRouterWithProtected(t)
	user := testutil.CreateTestUser(t, db, "mobilepreauthrej", "pass123", "user")

	preAuthToken, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, true, "")
	require.NoError(t, err)

	rec := makeMobileBearerRequest(t, router, http.MethodGet, "/api/user/status", preAuthToken, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
