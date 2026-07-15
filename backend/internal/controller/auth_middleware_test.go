package controller_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/testutil"

	gonetauth "github.com/leonkhoo123/gonet-auth"
	"github.com/leonkhoo123/gonet-auth/auth"
	"github.com/leonkhoo123/gonet-auth/jwt"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

	"github.com/gin-gonic/gin"
	golangJwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiddlewareRouter(t *testing.T) (*gin.Engine, *gonetauth.AuthConfig, *auth.Auth, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	_, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	router := gin.New()

	h := authgin.NewHandlers(authInstance, authCfg)

	authRouter := router.Group("/api/user")
	authRouter.Use(authgin.JWTAuthMiddleware(authInstance, []string{"/api/user/me", "/api/user/mfa/setup", "/api/user/mfa/confirm", "/api/logout"}))
	{
		authRouter.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "authenticated"})
		})
		authRouter.GET("/me", func(c *gin.Context) {
			username := c.GetString("username")
			c.JSON(http.StatusOK, gin.H{"username": username})
		})
		authRouter.POST("/mfa/setup", h.MFASetup())
		authRouter.POST("/mfa/confirm", h.MFAConfirm())
		authRouter.GET("/files/file-list", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"files": []string{}})
		})
	}

	return router, authCfg, authInstance, db
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
	router, cfg, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "expireduser", "pass123", "user")

	origMaxAge := cfg.Tokens.AccessToken
	cfg.Tokens.AccessToken = -1 * time.Hour
	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-expired")
	require.NoError(t, err)
	cfg.Tokens.AccessToken = origMaxAge

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
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

	badCfg := *cfg
	badCfg.JWT.SetSecret("a-different-secret-key")
	badJwtSvc := jwt.NewService(&badCfg)
	token, err := badJwtSvc.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-sig")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
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
	router, cfg, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "preauthuser", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, true, "")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
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
	router, cfg, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "revokeduser", "pass123", "user")

	result, err := authInstance.Login(context.Background(), user.Username, "pass123", "", "test-agent", "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, gonetauth.LoginAuthenticated, result.Code)

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
		Value: result.AccessToken,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)
	require.Equal(t, http.StatusOK, rec.Code)

	_, err = authInstance.Logout(context.Background(), result.RefreshToken, "127.0.0.1")
	require.NoError(t, err)

	rec2 := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "revoked")
}

func TestJWTAuth_TokenVersionMismatch(t *testing.T) {
	router, cfg, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "tvuser", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-tv")
	require.NoError(t, err)

	_, dbErr := db.Exec("UPDATE users SET token_version = token_version + 1 WHERE id = ?", user.ID)
	require.NoError(t, dbErr)

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
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
	router, cfg, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "validuser", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-valid")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
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
	router, _, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "beareruser", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-bearer")
	require.NoError(t, err)

	rec := makeBearerRequest(t, router, token)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestJWTAuth_MFAMandatoryNotSetup(t *testing.T) {
	router, cfg, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "mfaforced", "pass123", "user")

	_, err := db.Exec("UPDATE users SET mfa_mandatory = 1 WHERE id = ?", user.ID)
	require.NoError(t, err)

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-mfaforced")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
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
	router, cfg, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "mfaforced2", "pass123", "user")

	_, err := db.Exec("UPDATE users SET mfa_mandatory = 1 WHERE id = ?", user.ID)
	require.NoError(t, err)

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, user.Role, false, "family-mfaforced2")
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
		Value: token,
		Path:  "/",
	}

	reqMe := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	reqMe.AddCookie(cookie)
	recMe := httptest.NewRecorder()
	router.ServeHTTP(recMe, reqMe)
	assert.Equal(t, http.StatusOK, recMe.Code)

	reqSetup := httptest.NewRequest(http.MethodPost, "/api/user/mfa/setup", nil)
	reqSetup.AddCookie(cookie)
	recSetup := httptest.NewRecorder()
	router.ServeHTTP(recSetup, reqSetup)
	assert.Equal(t, http.StatusOK, recSetup.Code)
}

// Test for token with golangJwt.RegisteredClaims (no AccessTokenClaims custom fields)
func TestJWTAuth_InvalidClaimType(t *testing.T) {
	router, cfg, _, _ := setupMiddlewareRouter(t)

	token := golangJwt.NewWithClaims(golangJwt.SigningMethodHS256, golangJwt.RegisteredClaims{
		Subject:   "testuser",
		ExpiresAt: golangJwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	tokenStr, err := token.SignedString([]byte(cfg.JWT.GetSecret()))
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  cfg.Cookies.AccessToken,
		Value: tokenStr,
		Path:  "/",
	}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "invalid")
}

func TestJWTAuth_AlgorithmNone(t *testing.T) {
	router, cfg, _, _ := setupMiddlewareRouter(t)

	token := golangJwt.NewWithClaims(golangJwt.SigningMethodNone, golangJwt.MapClaims{
		"sub": "testuser",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(golangJwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	cookie := &http.Cookie{Name: cfg.Cookies.AccessToken, Value: tokenStr, Path: "/"}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_AlgorithmRS256(t *testing.T) {
	router, cfg, _, _ := setupMiddlewareRouter(t)

	// Create a token signed with HS256 using the correct secret, but marked as RS256 in header
	token := golangJwt.NewWithClaims(golangJwt.SigningMethodHS256, golangJwt.MapClaims{
		"sub": "testuser",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(cfg.JWT.GetSecret()))
	require.NoError(t, err)
	// Manually replace alg header from HS256 to RS256
	tokenStr = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." + tokenStr[strings.Index(tokenStr, ".")+1:]

	cookie := &http.Cookie{Name: "access_token", Value: tokenStr, Path: "/"}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_TamperedRoleClaim(t *testing.T) {
	router, cfg, authInstance, db := setupMiddlewareRouter(t)
	user := testutil.CreateTestUser(t, db, "tamperuser", "pass123", "user")

	token, err := authInstance.JWT.GenerateAccessToken(user.Username, user.TokenVersion, "admin", false, "family-tamper")
	require.NoError(t, err)

	cookie := &http.Cookie{Name: cfg.Cookies.AccessToken, Value: token, Path: "/"}
	rec := makeJWTAuthRequest(t, router, cookie)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "token invalidated due to role change", resp["error"])
}

func TestCookieSecurity_SecureFlag(t *testing.T) {
	t.Setenv("SECURE_MODE", "true")
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	cfg.Auth.SecureMode = true
	workDir := cfg.Server.FileRoot
	_, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)
	_ = authInstance

	assert.True(t, authCfg.SecureMode, "SecureMode should be true in production")

	testutil.CreateTestUser(t, db, "secureuser", "pass123", "user")

	router := gin.New()
	controller.ResetLoginLimiterForTest()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)

	body := map[string]string{"username": "secureuser", "password": "pass123"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	for _, c := range rec.Result().Cookies() {
		if c.Name == authCfg.Cookies.AccessToken || c.Name == authCfg.Cookies.RefreshToken {
			assert.True(t, c.Secure, "cookie %s should have Secure=true in production", c.Name)
			assert.Equal(t, http.SameSiteStrictMode, c.SameSite, "cookie %s should have SameSite=Strict", c.Name)
			assert.True(t, c.HttpOnly, "cookie %s should have HttpOnly=true", c.Name)
		}
	}
}

// Timing test: verify that login responds in ~same time for valid user vs nonexistent user
// to prevent timing-based user enumeration.
func TestLogin_TimingResistant(t *testing.T) {
	router, _, _, db := setupAuthRouter(t)

	testutil.CreateTestUser(t, db, "timinguser", "correctpass", "user")

	// Use a distinct IP per request so the per-IP login rate limiter never
	// short-circuits before reaching Login — otherwise we'd be timing 429
	// responses instead of the bcrypt comparison. The library performs a dummy
	// bcrypt hash for nonexistent users to keep the response constant-time.
	ip := func(i int) string { return fmt.Sprintf("10.10.%d.%d:1234", i/250, i%250+1) }

	// Warm up
	for i := 0; i < 3; i++ {
		loginJSONWithIP(t, router, "timinguser", "correctpass", ip(i))
		loginJSONWithIP(t, router, "nonexistent", "anypass", ip(100+i))
	}

	// Measure
	validStart := time.Now()
	for i := 0; i < 10; i++ {
		loginJSONWithIP(t, router, "timinguser", "wrongpass", ip(200+i))
	}
	validDuration := time.Since(validStart)

	nonexistentStart := time.Now()
	for i := 0; i < 10; i++ {
		loginJSONWithIP(t, router, "nonexistent", "anypass", ip(400+i))
	}
	nonexistentDuration := time.Since(nonexistentStart)

	// Both should be reasonably close (within ~50% of each other)
	ratio := float64(validDuration) / float64(nonexistentDuration)
	t.Logf("valid: %v, nonexistent: %v, ratio: %.2f", validDuration, nonexistentDuration, ratio)
	assert.InDelta(t, 1.0, ratio, 0.5, "bcrypt timing should be similar for valid vs nonexistent users")
}
