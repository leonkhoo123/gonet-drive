package controller_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"

	"github.com/leonkhoo123/gonet-auth/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupConfigRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *service.UserService, *auth.Auth, repository.CloudConfigRepository, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, configRepo, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()

	// Public config routes
	controller.SetupPublicConfigRoutes(router)

	// Auth routes (includes ConfigRoutes, admin, etc.)
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)
	controller.SetupAuthenticatedRoutes(router, cfg, authInstance, authCfg, userService, nil, nil, configRepo, nil)

	return router, cfg, userService, authInstance, configRepo, db
}

func createFakePNG() []byte {
	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x01, 0x01, 0x00, 0x05, 0x18, 0xD8, 0x72,
		0x9F, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	return png
}

func uploadLogoRequest(t *testing.T, router *gin.Engine, path, filename string, fileData []byte, accessCookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("logo", filename)
	require.NoError(t, err)
	_, err = part.Write(fileData)
	require.NoError(t, err)
	writer.Close()

	req := httptest.NewRequest(http.MethodPut, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if accessCookie != nil {
		req.AddCookie(accessCookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// ---------- 7.2 Manifest ----------

func TestGetManifest_Valid(t *testing.T) {
	router, _, _, _, _, _ := setupConfigRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/config/manifest", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	assert.Equal(t, "GoNet Drive Test", resp["name"])
	assert.Equal(t, "standalone", resp["display"])

	icons, ok := resp["icons"].([]interface{})
	require.True(t, ok, "icons should be present")
	assert.GreaterOrEqual(t, len(icons), 1)
}

// ---------- 7.2 Logo ----------

func TestGetLogo_ReturnsImage(t *testing.T) {
	router, _, _, _, _, _ := setupConfigRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/config/logo", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "image/png")
	assert.NotEmpty(t, rec.Body.Bytes())
}

// ---------- 7.2 Update Logo ----------

func TestUpdateLogo_Success(t *testing.T) {
	router, _, _, _, _, db := setupConfigRouter(t)
	testutil.CreateTestUser(t, db, "adminlogo", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminlogo", "pass123")

	pngData := createFakePNG()
	rec := uploadLogoRequest(t, router, "/api/user/admin/config/logo", "test.png", pngData, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Logo updated successfully", resp["message"])

	// Verify the logo file was written
	logoPath := config.GetLogoPath()
	data, err := os.ReadFile(logoPath)
	require.NoError(t, err)
	assert.Equal(t, pngData, data)
}

func TestUpdateLogo_NonPNG(t *testing.T) {
	router, _, _, _, _, db := setupConfigRouter(t)
	testutil.CreateTestUser(t, db, "adminlogo2", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminlogo2", "pass123")

	rec := uploadLogoRequest(t, router, "/api/user/admin/config/logo", "test.jpg", []byte("not a png"), accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Only PNG files are allowed", resp["error"])
}

func TestUpdateLogo_NonAdmin(t *testing.T) {
	router, _, _, _, _, db := setupConfigRouter(t)
	testutil.CreateTestUser(t, db, "regularlogo", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "regularlogo", "pass123")

	pngData := createFakePNG()
	rec := uploadLogoRequest(t, router, "/api/user/admin/config/logo", "test.png", pngData, accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "admin access required", resp["error"])
}

func TestUpdateLogo_ExceedsSize(t *testing.T) {
	router, _, _, _, _, db := setupConfigRouter(t)
	testutil.CreateTestUser(t, db, "adminlogo3", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminlogo3", "pass123")

	// Create a 6MB fake file (exceeds 5MB limit)
	largePNG := make([]byte, 6*1024*1024+1)
	copy(largePNG, createFakePNG())

	rec := uploadLogoRequest(t, router, "/api/user/admin/config/logo", "large.png", largePNG, accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "File size exceeds 5MB limit", resp["error"])
}

// ---------- 7.2 List Configs ----------

func TestListConfigs_Valid(t *testing.T) {
	router, _, _, _, _, db := setupConfigRouter(t)
	testutil.CreateTestUser(t, db, "configuser", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "configuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/config", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)

	var configs []map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &configs))
	assert.GreaterOrEqual(t, len(configs), 3, "should have at least the 3 default configs")
}

// ---------- 7.2 Update Config ----------

func TestUpdateConfig_Valid(t *testing.T) {
	router, _, _, _, _, db := setupConfigRouter(t)
	testutil.CreateTestUser(t, db, "configuser2", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "configuser2", "pass123")

	// First list configs to get an ID
	listRec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/config", nil, accessCookie)
	require.Equal(t, http.StatusOK, listRec.Code)

	var configs []map[string]interface{}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &configs))
	require.NotEmpty(t, configs)

	firstID := int(configs[0]["id"].(float64))
	newValue := "Updated Service Name"

	body := map[string]interface{}{
		"config_value": newValue,
	}
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/user/config/%d", firstID), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "config updated successfully", resp["message"])
}

func TestUpdateConfig_InvalidID(t *testing.T) {
	router, _, _, _, _, db := setupConfigRouter(t)
	testutil.CreateTestUser(t, db, "configuser3", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "configuser3", "pass123")

	body := map[string]string{
		"config_value": "test",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/user/config/abc", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid config id", resp["error"])
}

func TestUpdateConfig_NoAuth(t *testing.T) {
	router, _, _, _, _, _ := setupConfigRouter(t)

	body := map[string]string{
		"config_value": "test",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/user/config/1", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateConfig_NotFound(t *testing.T) {
	router, _, _, _, _, db := setupConfigRouter(t)
	testutil.CreateTestUser(t, db, "configuser4", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "configuser4", "pass123")

	body := map[string]string{
		"config_value": "test",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/user/config/99999", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}


