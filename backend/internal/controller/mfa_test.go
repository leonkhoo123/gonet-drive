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
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"

	gonetauth "github.com/leonkhoo123/gonet-auth"
	"github.com/leonkhoo123/gonet-auth/auth"
	"github.com/leonkhoo123/gonet-auth/ratelimit"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var loginLimiter = ratelimit.NewIPRateLimiter(controller.NewMemoryRateLimiterStore(1, 5))


func setupMFARouter(t *testing.T) (*gin.Engine, *gonetauth.AuthConfig, *auth.Auth, *service.UserService, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()
	loginLimiter = ratelimit.NewIPRateLimiter(controller.NewMemoryRateLimiterStore(1, 5))

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)

	h := authgin.NewHandlers(authInstance, authCfg)

	authRouter := router.Group("/api/user")
	authRouter.Use(authgin.JWTAuthMiddleware(authInstance, []string{"/api/user/me", "/api/user/mfa/setup", "/api/user/mfa/confirm", "/api/logout"}))
	{
		authRouter.POST("/mfa/setup", authgin.RateLimitMiddleware(loginLimiter), h.MFASetup())
		authRouter.POST("/mfa/confirm", authgin.RateLimitMiddleware(loginLimiter), h.MFAConfirm())
		authRouter.GET("/me", func(c *gin.Context) {
			username := c.GetString("username")
			c.JSON(http.StatusOK, gin.H{"username": username})
		})
		authRouter.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		authRouter.GET("/files/file-list", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"files": []string{}})
		})
	}

	return router, authCfg, authInstance, userService, db
}





// ---------- 3.4 MFA Setup Tests ----------

func TestMFASetup_Success(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfasetupuser", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-setup")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/user/mfa/setup", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Cookies.AccessToken, Value: token, Path: "/"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	assert.Equal(t, "success", envelope["status"])
	resp := envelope["data"].(map[string]interface{})
	assert.NotEmpty(t, resp["secret"])
	assert.NotEmpty(t, resp["url"])
}

func TestMFASetup_AlreadyEnabled(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfasetupuser2", "pass123", "user")
	_, dbErr := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", "JBSWY3DPEHPK3PXP", user.ID)
	require.NoError(t, dbErr)

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-setup2")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/user/mfa/setup", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Cookies.AccessToken, Value: token, Path: "/"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "mfa already enabled", resp["error"])
}

func TestMFASetup_Unauthenticated(t *testing.T) {
	router, _, _, _, _ := setupMFARouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/user/mfa/setup", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------- 3.4 MFA Enable Tests ----------

func TestMFAEnable_Success(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfaenableuser", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-enable")
	require.NoError(t, err)
	authCookie := &http.Cookie{Name: cfg.Cookies.AccessToken, Value: token, Path: "/"}

	setupReq := httptest.NewRequest(http.MethodPost, "/api/user/mfa/setup", nil)
	setupReq.AddCookie(authCookie)
	setupRec := httptest.NewRecorder()
	router.ServeHTTP(setupRec, setupReq)
	require.Equal(t, http.StatusOK, setupRec.Code)
	var setupEnvelope map[string]interface{}
	require.NoError(t, json.Unmarshal(setupRec.Body.Bytes(), &setupEnvelope))
	setupResp := setupEnvelope["data"].(map[string]interface{})
	secret := setupResp["secret"].(string)
	require.NotEmpty(t, secret)

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	enableBody := map[string]string{"code": code}
	enableJSON, _ := json.Marshal(enableBody)
	enableReq := httptest.NewRequest(http.MethodPost, "/api/user/mfa/confirm", bytes.NewReader(enableJSON))
	enableReq.Header.Set("Content-Type", "application/json")
	enableReq.AddCookie(authCookie)
	enableRec := httptest.NewRecorder()
	router.ServeHTTP(enableRec, enableReq)

	assert.Equal(t, http.StatusOK, enableRec.Code)
	var enableResp map[string]interface{}
	require.NoError(t, json.Unmarshal(enableRec.Body.Bytes(), &enableResp))
	assert.Equal(t, "success", enableResp["status"])

	enableData, _ := enableResp["data"].(map[string]interface{})
	require.NotNil(t, enableData)
	recoveryCodes, ok := enableData["recovery_codes"].([]interface{})
	assert.True(t, ok, "recovery_codes should be present")
	assert.NotEmpty(t, recoveryCodes, "recovery_codes should not be empty")

	var mfaEnabled bool
	db.QueryRow("SELECT mfa_enabled FROM users WHERE id = ?", user.ID).Scan(&mfaEnabled)
	assert.True(t, mfaEnabled, "MFA should be enabled in DB")
}

func TestMFAEnable_WrongCode(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfaenableuser2", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-enable2")
	require.NoError(t, err)
	authCookie := &http.Cookie{Name: cfg.Cookies.AccessToken, Value: token, Path: "/"}

	setupReq := httptest.NewRequest(http.MethodPost, "/api/user/mfa/setup", nil)
	setupReq.AddCookie(authCookie)
	setupRec := httptest.NewRecorder()
	router.ServeHTTP(setupRec, setupReq)
	require.Equal(t, http.StatusOK, setupRec.Code)

	enableBody := map[string]string{"code": "000000"}
	enableJSON, _ := json.Marshal(enableBody)
	enableReq := httptest.NewRequest(http.MethodPost, "/api/user/mfa/confirm", bytes.NewReader(enableJSON))
	enableReq.Header.Set("Content-Type", "application/json")
	enableReq.AddCookie(authCookie)
	enableRec := httptest.NewRecorder()
	router.ServeHTTP(enableRec, enableReq)

	assert.Equal(t, http.StatusBadRequest, enableRec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(enableRec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid code", resp["error"])
}

func TestMFAEnable_NoSetupFirst(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfaenableuser3", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-enable3")
	require.NoError(t, err)

	enableBody := map[string]string{"code": "123456"}
	enableJSON, _ := json.Marshal(enableBody)
	req := httptest.NewRequest(http.MethodPost, "/api/user/mfa/confirm", bytes.NewReader(enableJSON))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: cfg.Cookies.AccessToken, Value: token, Path: "/"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "initiate mfa setup first", resp["error"])
}

// ---------- 3.4 MFA Verify Tests ----------

func TestMFAVerify_Success(t *testing.T) {
	router, _, _, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfaverifyuser", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, dbErr := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, dbErr)

	// Login to get mfa_pending cookie
	body := map[string]string{"username": "mfaverifyuser", "password": "pass123"}
	jsonBody, _ := json.Marshal(body)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	var mfaPendingCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "mfa_pending" {
			mfaPendingCookie = c
			break
		}
	}
	require.NotNil(t, mfaPendingCookie, "mfa_pending cookie should be set")

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	verifyBody := map[string]string{"code": code}
	verifyJSON, _ := json.Marshal(verifyBody)
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/mfa/verify", bytes.NewReader(verifyJSON))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyReq.AddCookie(mfaPendingCookie)
	verifyRec := httptest.NewRecorder()
	router.ServeHTTP(verifyRec, verifyReq)

	assert.Equal(t, http.StatusOK, verifyRec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(verifyRec.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])
	data, _ := resp["data"].(map[string]interface{})
	require.NotNil(t, data)
	assert.Equal(t, "logged_in", data["auth_status"])

	hasAccess := false
	hasRefresh := false
	for _, c := range verifyRec.Result().Cookies() {
		if c.Name == "access_token" {
			hasAccess = true
		}
		if c.Name == "refresh_token" {
			hasRefresh = true
		}
	}
	assert.True(t, hasAccess, "access_token cookie should be set")
	assert.True(t, hasRefresh, "refresh_token cookie should be set")
}

func TestMFAVerify_WrongCode(t *testing.T) {
	router, _, _, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfaverifyuser2", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, err)

	body := map[string]string{"username": "mfaverifyuser2", "password": "pass123"}
	jsonBody, _ := json.Marshal(body)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	var mfaPendingCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "mfa_pending" {
			mfaPendingCookie = c
			break
		}
	}
	require.NotNil(t, mfaPendingCookie)

	verifyBody := map[string]string{"code": "000000"}
	verifyJSON, _ := json.Marshal(verifyBody)
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/mfa/verify", bytes.NewReader(verifyJSON))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyReq.AddCookie(mfaPendingCookie)
	verifyRec := httptest.NewRecorder()
	router.ServeHTTP(verifyRec, verifyReq)

	assert.Equal(t, http.StatusBadRequest, verifyRec.Code)
}

func TestMFAVerify_NoPendingCookie(t *testing.T) {
	router, _, _, _, _ := setupMFARouter(t)

	verifyBody := map[string]string{"code": "123456"}
	verifyJSON, _ := json.Marshal(verifyBody)
	req := httptest.NewRequest(http.MethodPost, "/api/mfa/verify", bytes.NewReader(verifyJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "missing pre-auth token", resp["error"])
}

func TestMFAVerify_ExpiredPending(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfaverifyuser3", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, err)

	origMfaMaxAge := cfg.Tokens.MfaPending
	cfg.Tokens.MfaPending = -1 * time.Hour
	preAuthToken, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, true, "")
	require.NoError(t, err)
	cfg.Tokens.MfaPending = origMfaMaxAge

	verifyBody := map[string]string{"code": "123456"}
	verifyJSON, _ := json.Marshal(verifyBody)
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/mfa/verify", bytes.NewReader(verifyJSON))
	verifyReq.Header.Set("Content-Type", "application/json")
	verifyReq.AddCookie(&http.Cookie{Name: "mfa_pending", Value: preAuthToken, Path: "/"})
	verifyRec := httptest.NewRecorder()
	router.ServeHTTP(verifyRec, verifyReq)

	assert.Equal(t, http.StatusUnauthorized, verifyRec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(verifyRec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid or expired pre-auth token", resp["error"])
}

func TestMFAVerify_ReplayAttack(t *testing.T) {
	router, _, _, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfaverifyuser4", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, err)

	body := map[string]string{"username": "mfaverifyuser4", "password": "pass123"}
	jsonBody, _ := json.Marshal(body)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	var mfaPendingCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "mfa_pending" {
			mfaPendingCookie = c
			break
		}
	}
	require.NotNil(t, mfaPendingCookie)

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	verifyBody := map[string]string{"code": code}
	verifyJSONBytes, _ := json.Marshal(verifyBody)

	verifyReq1 := httptest.NewRequest(http.MethodPost, "/api/mfa/verify", bytes.NewReader(verifyJSONBytes))
	verifyReq1.Header.Set("Content-Type", "application/json")
	verifyReq1.AddCookie(mfaPendingCookie)
	verifyRec1 := httptest.NewRecorder()
	router.ServeHTTP(verifyRec1, verifyReq1)
	require.Equal(t, http.StatusOK, verifyRec1.Code, "first verify should succeed")

	verifyReq2 := httptest.NewRequest(http.MethodPost, "/api/mfa/verify", bytes.NewReader(verifyJSONBytes))
	verifyReq2.Header.Set("Content-Type", "application/json")
	verifyReq2.AddCookie(mfaPendingCookie)
	verifyRec2 := httptest.NewRecorder()
	router.ServeHTTP(verifyRec2, verifyReq2)

	assert.Equal(t, http.StatusBadRequest, verifyRec2.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(verifyRec2.Body.Bytes(), &resp))
	assert.Equal(t, "code already used", resp["error"])
}

func TestMFAVerify_RateLimit(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfaverifyuser5", "pass123", "user")
	secret := "JBSWY3DPEHPK3PXP"
	_, err := db.Exec("UPDATE users SET mfa_enabled = 1, mfa_secret = ? WHERE id = ?", secret, user.ID)
	require.NoError(t, err)

	// Temporarily raise MFA max attempts to prevent MFA lockout from interfering with IP rate limit test.
	origMaxAttempts := cfg.MFA.MaxAttempts
	cfg.MFA.MaxAttempts = 100

	preAuthToken, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, true, "")
	require.NoError(t, err)
	pendingCookie := &http.Cookie{Name: "mfa_pending", Value: preAuthToken, Path: "/"}

	verifyBody := map[string]string{"code": "wrongcode"}
	verifyJSONBytes, _ := json.Marshal(verifyBody)

	for i := 0; i < 6; i++ {
		verifyReq := httptest.NewRequest(http.MethodPost, "/api/mfa/verify", bytes.NewReader(verifyJSONBytes))
		verifyReq.Header.Set("Content-Type", "application/json")
		verifyReq.AddCookie(pendingCookie)
		verifyReq.RemoteAddr = "10.0.1.1:1234"
		verifyRec := httptest.NewRecorder()
		router.ServeHTTP(verifyRec, verifyReq)

		if i < 5 {
			require.NotEqual(t, http.StatusTooManyRequests, verifyRec.Code, "request %d should not be rate-limited", i+1)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, verifyRec.Code, "request %d should be rate-limited", i+1)
		}
	}

	cfg.MFA.MaxAttempts = origMaxAttempts
}

// ---------- 3.4 MFA Mandatory Tests ----------

func TestMFAMandatory_NotSetup(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfamandatory", "pass123", "user")
	_, err := db.Exec("UPDATE users SET mfa_mandatory = 1 WHERE id = ?", user.ID)
	require.NoError(t, err)

	token, tokenErr := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-mfamand")
	require.NoError(t, tokenErr)
	authCookie := &http.Cookie{Name: cfg.Cookies.AccessToken, Value: token, Path: "/"}

	fileReq := httptest.NewRequest(http.MethodGet, "/api/user/files/file-list", nil)
	fileReq.AddCookie(authCookie)
	fileRec := httptest.NewRecorder()
	router.ServeHTTP(fileRec, fileReq)
	assert.Equal(t, http.StatusForbidden, fileRec.Code)
	var fileResp map[string]interface{}
	require.NoError(t, json.Unmarshal(fileRec.Body.Bytes(), &fileResp))
	assert.Equal(t, "mfa_setup_required", fileResp["error"])

	meReq := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	meReq.AddCookie(authCookie)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	assert.Equal(t, http.StatusOK, meRec.Code)

	setupReq := httptest.NewRequest(http.MethodPost, "/api/user/mfa/setup", nil)
	setupReq.AddCookie(authCookie)
	setupRec := httptest.NewRecorder()
	router.ServeHTTP(setupRec, setupReq)
	assert.Equal(t, http.StatusOK, setupRec.Code)
}

func TestMFAMandatory_AfterSetup(t *testing.T) {
	router, cfg, authInstance, _, db := setupMFARouter(t)
	user := testutil.CreateTestUser(t, db, "mfamandatory2", "pass123", "user")
	_, err := db.Exec("UPDATE users SET mfa_mandatory = 1 WHERE id = ?", user.ID)
	require.NoError(t, err)

	token, tokenErr := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-mfamand2")
	require.NoError(t, tokenErr)
	authCookie := &http.Cookie{Name: cfg.Cookies.AccessToken, Value: token, Path: "/"}

	setupReq := httptest.NewRequest(http.MethodPost, "/api/user/mfa/setup", nil)
	setupReq.AddCookie(authCookie)
	setupRec := httptest.NewRecorder()
	router.ServeHTTP(setupRec, setupReq)
	require.Equal(t, http.StatusOK, setupRec.Code)
	var setupEnvelope map[string]interface{}
	require.NoError(t, json.Unmarshal(setupRec.Body.Bytes(), &setupEnvelope))
	setupResp := setupEnvelope["data"].(map[string]interface{})
	secret := setupResp["secret"].(string)

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)
	enableBody := map[string]string{"code": code}
	enableJSON, _ := json.Marshal(enableBody)
	enableReq := httptest.NewRequest(http.MethodPost, "/api/user/mfa/confirm", bytes.NewReader(enableJSON))
	enableReq.Header.Set("Content-Type", "application/json")
	enableReq.AddCookie(authCookie)
	enableRec := httptest.NewRecorder()
	router.ServeHTTP(enableRec, enableReq)
	require.Equal(t, http.StatusOK, enableRec.Code)

	authInstance.ClearUserRoleCache(user.Username)

	fileReq := httptest.NewRequest(http.MethodGet, "/api/user/files/file-list", nil)
	fileReq.AddCookie(authCookie)
	fileRec := httptest.NewRecorder()
	router.ServeHTTP(fileRec, fileReq)
	assert.Equal(t, http.StatusOK, fileRec.Code, "after MFA setup, all routes should be accessible")
}


