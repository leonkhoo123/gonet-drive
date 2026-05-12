package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-file-server/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func accessCookieTestConfig() *config.CloudConfig {
	return &config.CloudConfig{
		Server: config.ServerConfig{
			AppEnv:         "local",
			FileRoot:       "/tmp/test",
			ListenAddr:     ":0",
			AllowedOrigins: []string{"*"},
		},
		Auth: config.AuthConfig{
			AppJwt:             "ON",
			JwtSecret:          "test-secret-key-for-testing-only",
			AdminUser:          "admin",
			AdminPass:          "admin123",
			CookieAccessToken:  "access_token",
			CookieRefreshToken: "refresh_token",
			CookieMfaPending:   "mfa_pending",
			CookieShareJwt:     "shareJwt",
			AccessTokenMaxAge:  15 * time.Minute,
			RefreshTokenMaxAge: 7 * 24 * time.Hour,
			MfaPendingMaxAge:   5 * time.Minute,
			ShareJwtMaxAge:     7 * 24 * time.Hour,
		},
		Defaults: config.AppDefaults{
			ServiceName:     "GoNet Drive Test",
			UploadChunkSize: "5",
			StorageLimit:    "20480",
		},
	}
}

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, rec
}

func extractSetCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestSetAccessToken_CookieSet(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetAccessToken(c, cfg, "access-token-value")

	cookie := extractSetCookie(rec, "access_token")
	require.NotNil(t, cookie)
	assert.Equal(t, "access-token-value", cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
}

func TestClearAccessToken_CookieCleared(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	ClearAccessToken(c, cfg)

	cookie := extractSetCookie(rec, "access_token")
	require.NotNil(t, cookie)
	assert.Equal(t, -1, cookie.MaxAge)
}

func TestGetAccessToken_Exists(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetAccessToken(c, cfg, "my-token")
	cookie := extractSetCookie(rec, "access_token")
	c.Request.AddCookie(cookie)

	result, err := GetAccessToken(c, cfg)
	assert.NoError(t, err)
	assert.Equal(t, "my-token", result)
}

func TestGetAccessToken_Missing(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, _ := newTestContext()

	_, err := GetAccessToken(c, cfg)
	assert.Error(t, err)
}

func TestSetRefreshToken_CookieSet(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetRefreshToken(c, cfg, "refresh-token-value")

	cookie := extractSetCookie(rec, "refresh_token")
	require.NotNil(t, cookie)
	assert.Equal(t, "refresh-token-value", cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
}

func TestGetRefreshToken_Exists(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetRefreshToken(c, cfg, "my-refresh-token")
	cookie := extractSetCookie(rec, "refresh_token")
	c.Request.AddCookie(cookie)

	result, err := GetRefreshToken(c, cfg)
	assert.NoError(t, err)
	assert.Equal(t, "my-refresh-token", result)
}

func TestGetRefreshToken_Missing(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, _ := newTestContext()

	_, err := GetRefreshToken(c, cfg)
	assert.Error(t, err)
}

func TestClearRefreshToken_CookieCleared(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	ClearRefreshToken(c, cfg)

	cookie := extractSetCookie(rec, "refresh_token")
	require.NotNil(t, cookie)
	assert.Equal(t, -1, cookie.MaxAge)
}

func TestSetMfaPendingToken_CookieSet(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetMfaPendingToken(c, cfg, "mfa-token-value")

	cookie := extractSetCookie(rec, "mfa_pending")
	require.NotNil(t, cookie)
	assert.Equal(t, "mfa-token-value", cookie.Value)
	assert.True(t, cookie.HttpOnly)
}

func TestSetShareJwt_CookieSet(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetShareJwt(c, cfg, "share-jwt-token", 3600, "share123")

	cookieName := "shareJwt_share123"
	cookie := extractSetCookie(rec, cookieName)
	require.NotNil(t, cookie, "expected cookie named %q", cookieName)
	assert.Equal(t, "share-jwt-token", cookie.Value)
	assert.True(t, cookie.HttpOnly)
}

func TestShareCookie_SameSiteLax(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetShareJwt(c, cfg, "token", 3600, "share1")

	cookie := extractSetCookie(rec, "shareJwt_share1")
	require.NotNil(t, cookie)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}

func TestAuthCookie_SameSiteStrict(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetAccessToken(c, cfg, "token")
	cookie := extractSetCookie(rec, "access_token")
	require.NotNil(t, cookie)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)

	rec2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(rec2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	SetRefreshToken(c2, cfg, "token")
	cookie2 := extractSetCookie(rec2, "refresh_token")
	require.NotNil(t, cookie2)
	assert.Equal(t, http.SameSiteStrictMode, cookie2.SameSite)
}

func TestSecureMode_Local(t *testing.T) {
	cfg := accessCookieTestConfig()
	c, rec := newTestContext()

	SetAccessToken(c, cfg, "token")
	cookie := extractSetCookie(rec, "access_token")
	require.NotNil(t, cookie)
	assert.False(t, cookie.Secure)
}

func TestSecureMode_Prod(t *testing.T) {
	cfg := accessCookieTestConfig()
	cfg.Server.AppEnv = "prod"
	c, rec := newTestContext()

	SetAccessToken(c, cfg, "token")
	cookie := extractSetCookie(rec, "access_token")
	require.NotNil(t, cookie)
	assert.True(t, cookie.Secure)
}
