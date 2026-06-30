package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"
	"go-file-server/internal/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func setupSecurityRouter(t *testing.T) (*gin.Engine, *config.CloudConfig) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	service.JobQueue = make(chan service.Job, 100)
	service.StartFileOperationWorker()
	go ws.Manager.Start()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)
	controller.SetupAuthenticatedRoutes(router, cfg, authInstance, authCfg, userService, nil, nil, nil, nil)

	return router, cfg
}

func setupSecurityRouterWithCORS(t *testing.T) *gin.Engine {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	_, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://example.com"},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "X-Share-Id"},
	}))
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)

	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	return router
}

// ---------- 7.3 SQL Injection ----------

func TestSQLInjection_Username(t *testing.T) {
	router, _ := setupSecurityRouter(t)

	payloads := []string{
		"' OR '1'='1",
		"' OR 1=1 --",
		"admin'--",
		"' UNION SELECT NULL--",
		"'; DROP TABLE users; --",
	}

	for _, payload := range payloads {
		body := map[string]string{
			"username": payload,
			"password": "anything",
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"SQL injection payload %q should not authenticate", payload)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "invalid credentials", resp["error"],
			"SQL injection payload %q should return 'invalid credentials'", payload)
	}
}

func TestSQLInjection_Password(t *testing.T) {
	router, _ := setupSecurityRouter(t)

	payloads := []string{
		"' OR '1'='1",
		"' OR 1=1 --",
		"anything' OR 'x'='x",
	}

	for _, payload := range payloads {
		body := map[string]string{
			"username": "admin",
			"password": payload,
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"SQL injection in password %q should not authenticate", payload)
	}
}

// ---------- 7.3 XSS ----------

func TestXSS_Path(t *testing.T) {
	router, cfg := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "xssuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "xssuser", "pass123")

	// Use special characters that constitute XSS vectors but are valid filenames
	// Note: < and > are not valid on all filesystems (e.g. NTFS), so we test with
	// other special chars and verify the JSON response is properly escaped.
	xssName := "xss_&_quot_test"
	xssPath := filepath.Join(cfg.Server.FileRoot, xssName)
	err := os.WriteFile(xssPath, []byte("xss test content"), 0644)
	require.NoError(t, err)

	xssDirName := "dir_with_'single'_quotes"
	xssDirPath := filepath.Join(cfg.Server.FileRoot, xssDirName)
	err = os.MkdirAll(xssDirPath, 0755)
	require.NoError(t, err)

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/file-list?path=/", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify the response is valid JSON (Go's json.Marshal HTML-escapes <, >, & by default)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp),
		"response should be valid JSON even with special-char path names")

	// Verify Content-Type is application/json (not text/html)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json",
		"response Content-Type should be application/json")

	// Verify that items are returned with their original names in JSON
	items, ok := resp["items"].([]interface{})
	require.True(t, ok, "items should be present in file list response")

	foundNames := make(map[string]bool)
	for _, item := range items {
		m := item.(map[string]interface{})
		name := m["name"].(string)
		foundNames[name] = true
	}
	assert.True(t, foundNames[xssName], "XSS-named file should appear in file list")
	assert.True(t, foundNames[xssDirName], "XSS-named dir should appear in file list")
}

// ---------- 7.3 CSRF ----------

func TestCSRF_NoTokenCookie(t *testing.T) {
	// This application uses JWT in HttpOnly cookies, not CSRF tokens.
	// CORS with AllowCredentials restricts cross-origin cookie usage.
	// This test documents that unauthenticated POST requests are rejected.
	router, _ := setupSecurityRouter(t)

	// Test unauthenticated POST to a protected endpoint
	req := httptest.NewRequest(http.MethodPost, "/api/user/files/copy",
		bytes.NewReader([]byte(`{"sources":["/test"],"destDir":"/dest"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code,
		"POST without auth cookie should be rejected")
}

// ---------- 7.3 Path Traversal ----------

func TestPathTraversal_FileList(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "pathtravuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "pathtravuser", "pass123")

	payloads := []string{
		"../",
		"..%2F",
		"..././",
		"../../",
		"..\\",
		"/../../../etc/passwd",
		"....//....//",
	}

	for _, payload := range payloads {
		rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
			"/api/user/files/file-list?path="+payload, nil, accessCookie)

		assert.NotEqual(t, http.StatusOK, rec.Code,
			"file-list with path %q should not return 200", payload)
	}
}

func TestPathTraversal_CopySources(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "pathtravuser2", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "pathtravuser2", "pass123")

	payloads := []string{
		"../",
		"../../",
		"/../../../etc/passwd",
	}

	for _, payload := range payloads {
		rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
			"/api/user/files/copy",
			map[string]interface{}{
				"sources": []string{payload},
				"destDir": "/dest",
			},
			accessCookie)

		assert.NotEqual(t, http.StatusOK, rec.Code,
			"copy with source %q should not return 200", payload)
	}
}

func TestPathTraversal_CopyDestDir(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "pathtravuser3", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "pathtravuser3", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/copy",
		map[string]interface{}{
			"sources": []string{"/src"},
			"destDir": "../../etc",
		},
		accessCookie)

	assert.NotEqual(t, http.StatusOK, rec.Code,
		"copy with traversal destination should not return 200")
}

func TestPathTraversal_Move(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "pathtravuser4", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "pathtravuser4", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/move",
		map[string]interface{}{
			"sources": []string{"../"},
			"destDir": "/dest",
		},
		accessCookie)

	assert.NotEqual(t, http.StatusOK, rec.Code,
		"move with traversal source should not return 200")
}

func TestPathTraversal_Delete(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "pathtravuser5", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "pathtravuser5", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/delete",
		map[string]interface{}{
			"sources": []string{"../../"},
		},
		accessCookie)

	assert.NotEqual(t, http.StatusOK, rec.Code,
		"delete with traversal source should not return 200")
}

func TestPathTraversal_CreateFolder(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "pathtravuser6", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "pathtravuser6", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/create-folder",
		map[string]interface{}{
			"dir": "../",
		},
		accessCookie)

	assert.NotEqual(t, http.StatusOK, rec.Code,
		"create-folder with traversal dir should not return 200")
}

func TestPathTraversal_Rename(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "pathtravuser7", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "pathtravuser7", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/rename",
		map[string]interface{}{
			"source":  "../../",
			"newName": "test",
		},
		accessCookie)

	assert.NotEqual(t, http.StatusOK, rec.Code,
		"rename with traversal source should not return 200")
}

func TestPathTraversal_Properties(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "pathtravuser8", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "pathtravuser8", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/properties",
		map[string]interface{}{
			"sources": []string{"../../"},
		},
		accessCookie)

	assert.NotEqual(t, http.StatusOK, rec.Code,
		"properties with traversal source should not return 200")
}

// ---------- 7.3 HttpOnly Cookies ----------

func TestTokenInCookieNotAccessibleToJS(t *testing.T) {
	router, _ := setupSecurityRouter(t)
	db := config.DB
	testutil.CreateTestUser(t, db, "cookieuser", "pass123", "user")

	body := map[string]string{
		"username": "cookieuser",
		"password": "pass123",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	require.NotEmpty(t, cookies, "login should set cookies")

	cookieNames := make(map[string]bool)
	for _, c := range cookies {
		cookieNames[c.Name] = true
		assert.True(t, c.HttpOnly,
			"cookie %q must be HttpOnly to prevent JavaScript access", c.Name)
		assert.Equal(t, "/", c.Path,
			"cookie %q should have Path=/" , c.Name)
	}

	assert.True(t, cookieNames["access_token"], "access_token cookie should be set")
	assert.True(t, cookieNames["refresh_token"], "refresh_token cookie should be set")
}

// ---------- 7.3 CORS ----------

func TestCORS_HeadersPresent(t *testing.T) {
	router := setupSecurityRouterWithCORS(t)

	// Test that CORS headers are present on cross-origin GET requests.
	// Origin must match the allowed list, but Host must differ (httptest default host is "example.com").
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Host = "mydrive.local"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	assert.NotEmpty(t, allowOrigin, "Access-Control-Allow-Origin header should be present")
	assert.Equal(t, "https://example.com", allowOrigin)

	allowCredentials := rec.Header().Get("Access-Control-Allow-Credentials")
	assert.Equal(t, "true", allowCredentials,
		"Access-Control-Allow-Credentials should be true")
}

func TestCORS_PreflightOptions(t *testing.T) {
	router := setupSecurityRouterWithCORS(t)

	// OPTIONS preflight from an allowed cross-origin
	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Host = "mydrive.local"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	assert.NotEmpty(t, allowOrigin, "Access-Control-Allow-Origin should be present on preflight")
	assert.Equal(t, "https://example.com", allowOrigin)

	allowCredentials := rec.Header().Get("Access-Control-Allow-Credentials")
	assert.Equal(t, "true", allowCredentials,
		"Access-Control-Allow-Credentials should be true on preflight")

	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	assert.NotEmpty(t, allowMethods, "Access-Control-Allow-Methods should be present, got headers: %v", rec.Header())
}

func TestCORS_OriginRejected(t *testing.T) {
	router := setupSecurityRouterWithCORS(t)

	// Request from an origin NOT in the allowed list
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "https://evil-site.com")
	req.Host = "mydrive.local"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// gin-contrib/cors returns 403 Forbidden for disallowed origins
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
