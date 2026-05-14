package controller_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiddlewareRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *service.UserService, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, _ := testutil.SetupServices(t, db, workDir)

	middleware.ResetLoginLimiter()

	router := gin.New()

	authRouter := router.Group("/api/user")
	authRouter.Use(middleware.JWTAuthMiddleware(cfg))
	{
		authRouter.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "authenticated"})
		})
		authRouter.GET("/me", func(c *gin.Context) {
			username := c.GetString("username")
			c.JSON(http.StatusOK, gin.H{"username": username})
		})
		authRouter.GET("/mfa/setup", func(c *gin.Context) { userService.SetupMFA(c, cfg) })
		authRouter.POST("/mfa/enable", userService.EnableMFA)
		authRouter.GET("/files/file-list", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"files": []string{}})
		})
	}

	return router, cfg, userService, db
}

func makeJWTAuthRequest(t *testing.T, router *gin.Engine, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func makeBearerRequest(t *testing.T, router *gin.Engine, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestJWTAuth_NoToken(t *testing.T) {
	router, _, _, _ := setupMiddlewareRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	testutil.AssertAuthError(t, rec, http.StatusUnauthorized)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "expireduser", "pass123", "user")

	origMaxAge := cfg.Auth.AccessTokenMaxAge
	cfg.Auth.AccessTokenMaxAge = -1 * time.Hour
	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, "family-expired")
	require.NoError(t, err)
	cfg.Auth.AccessTokenMaxAge = origMaxAge

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: token,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid or expired token", resp["error"])
}

func TestJWTAuth_InvalidSignature(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "siguser", "pass123", "user")

	badCfg := &config.CloudConfig{
		Server: cfg.Server,
		Auth:   cfg.Auth,
	}
	badCfg.Auth.JwtSecret = "a-different-secret-key"
	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, badCfg, false, "family-sig")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: token,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid or expired token", resp["error"])
}

func TestJWTAuth_PreAuthTokenRejected(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "preauthuser", "pass123", "user")

	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, true, "")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: token,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "pre-auth token not allowed", resp["error"])
}

func TestJWTAuth_RevokedSession(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "revokeduser", "pass123", "user")

	familyID := "family-revoked-test"
	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, familyID)
	require.NoError(t, err)

	middleware.RevokedSessionsCache.Set(familyID, true, 20*time.Minute)

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: token,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "session revoked", resp["error"])
}

func TestJWTAuth_TokenVersionMismatch(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "tvuser", "pass123", "user")

	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, "family-tv")
	require.NoError(t, err)

	_, dbErr := db.Exec("UPDATE users SET token_version = token_version + 1 WHERE id = ?", user.ID)
	require.NoError(t, dbErr)

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: token,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "token revoked", resp["error"])
}

func TestJWTAuth_ValidToken(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "validuser", "pass123", "user")

	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, "family-valid")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: token,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "authenticated", resp["message"])
}

func TestJWTAuth_BearerHeader(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "beareruser", "pass123", "user")

	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, "family-bearer")
	require.NoError(t, err)

	rec := makeBearerRequest(t, router, token)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestJWTAuth_MFAMandatoryNotSetup(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "mfaforced", "pass123", "user")

	_, err := db.Exec("UPDATE users SET mfa_mandatory = 1 WHERE id = ?", user.ID)
	require.NoError(t, err)

	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, "family-mfaforced")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: token,
		Path:  "/",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/files/file-list", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "mfa_setup_required", resp["error"])
}

func TestJWTAuth_MFAMandatoryAllowedPaths(t *testing.T) {
	router, cfg, _, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "mfaforced2", "pass123", "user")

	_, err := db.Exec("UPDATE users SET mfa_mandatory = 1 WHERE id = ?", user.ID)
	require.NoError(t, err)

	token, err := middleware.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, user.Username == cfg.Auth.AdminUser, cfg, false, "family-mfaforced2")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: token,
		Path:  "/",
	}

	reqMe := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	reqMe.AddCookie(cookie)
	recMe := httptest.NewRecorder()
	router.ServeHTTP(recMe, reqMe)
	assert.Equal(t, http.StatusOK, recMe.Code)

	reqSetup := httptest.NewRequest(http.MethodGet, "/api/user/mfa/setup", nil)
	reqSetup.AddCookie(cookie)
	recSetup := httptest.NewRecorder()
	router.ServeHTTP(recSetup, reqSetup)
	assert.Equal(t, http.StatusOK, recSetup.Code)
}

// Test for token with jwt.RegisteredClaims (no AccessTokenClaims custom fields)
func TestJWTAuth_InvalidClaimType(t *testing.T) {
	router, cfg, _, _ := setupMiddlewareRouter(t)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "testuser",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	tokenStr, err := token.SignedString([]byte(cfg.Auth.JwtSecret))
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Auth.CookieAccessToken,
		Value: tokenStr,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "user not found", resp["error"])
}
