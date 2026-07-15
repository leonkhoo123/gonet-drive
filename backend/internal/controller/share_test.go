package controller_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/model"
	"go-file-server/internal/testutil"

	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupShareRouter(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot

	_, sharingService, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)

	authRouter := router.Group("/api/user")
	if cfg.Auth.AppJwt != "OFF" {
		authRouter.Use(authgin.JWTAuthMiddleware(authInstance, []string{"/api/user/me", "/api/user/mfa/setup", "/api/user/mfa/confirm", "/api/logout"}))
	}
	controller.ShareRoutes(authRouter, sharingService)

	return router, db
}

// ---------------------------------------------------------------------------
// Create Share
// ---------------------------------------------------------------------------

func TestCreateShare_Success(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "creator", "pass123", "user")

	// create a test dir so IsDir detection works
	workDir := config.AppConfig.Server.FileRoot
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, "share_me"), 0o755))

	accessCookie := testutil.LoginAndGetCookie(t, router, "creator", "pass123")

	body := map[string]interface{}{
		"path":             "share_me",
		"description":      "my shared folder",
		"expires_in_hours": 24,
		"authority":        "view",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := testutil.DecodeData(t, rec)
	assert.Equal(t, "Share link created successfully", resp["message"])
	assert.NotEmpty(t, resp["pin"])

	share, ok := resp["share"].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, share["id"])
	assert.Equal(t, "share_me", share["path"])
	assert.Equal(t, "view", share["authority"])
	assert.Equal(t, "my shared folder", share["description"])
	assert.Equal(t, "creator", share["username"])
	assert.False(t, share["blocked"].(bool))
}

func TestCreateShare_WithExpiry(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "creator", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "creator", "pass123")

	body := map[string]interface{}{
		"path":             "somefile.txt",
		"description":      "expires in 2h",
		"expires_in_hours": 2,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := testutil.DecodeData(t, rec)
	share, ok := resp["share"].(map[string]interface{})
	require.True(t, ok)

	expiresAtStr, ok := share["expires_at"].(string)
	require.True(t, ok)
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtStr)
	require.NoError(t, err)

	now := time.Now()
	expected := now.Add(2 * time.Hour)
	assert.WithinDuration(t, expected, expiresAt, 5*time.Second)
}

func TestCreateShare_NeverExpires(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "creator", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "creator", "pass123")

	body := map[string]interface{}{
		"path":             "folder",
		"description":      "never expires",
		"expires_in_hours": -1,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := testutil.DecodeData(t, rec)
	share, ok := resp["share"].(map[string]interface{})
	require.True(t, ok)

	expiresAtStr := share["expires_at"].(string)
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtStr)
	require.NoError(t, err)
	assert.True(t, expiresAt.Equal(model.NeverExpires), "expected NeverExpires sentinel")
}

func TestCreateShare_InvalidExpiry(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "creator", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "creator", "pass123")

	body := map[string]interface{}{
		"path":             "folder",
		"description":      "bad expiry",
		"expires_in_hours": 0,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateShare_ModifyAuthority(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "creator", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "creator", "pass123")

	body := map[string]interface{}{
		"path":             "editable",
		"description":      "modify share",
		"expires_in_hours": 24,
		"authority":        "modify",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := testutil.DecodeData(t, rec)
	share, ok := resp["share"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "modify", share["authority"])
}

func TestCreateShare_ViewAuthorityDefault(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "creator", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "creator", "pass123")

	body := map[string]interface{}{
		"path":             "viewonly",
		"description":      "default view",
		"expires_in_hours": 24,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := testutil.DecodeData(t, rec)
	share, ok := resp["share"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "view", share["authority"])
}

func TestCreateShare_Unauthenticated(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "creator", "pass123", "user")

	body := map[string]interface{}{
		"path":             "folder",
		"description":      "no token",
		"expires_in_hours": 24,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------------------------------------------------------------------------
// List Shares
// ---------------------------------------------------------------------------

func TestListShares_Success(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "lister", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "lister", "pass123")

	// create a share via API first
	body := map[string]interface{}{
		"path":             "listed",
		"description":      "my share",
		"expires_in_hours": 24,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	createResp := testutil.DecodeData(t, rec)
	createdShare := createResp["share"].(map[string]interface{})
	createdID := createdShare["id"].(string)

	// list shares
	rec2 := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/share/get-shares", nil, accessCookie)
	require.Equal(t, http.StatusOK, rec2.Code)

	listResp := testutil.DecodeData(t, rec2)
	shares, ok := listResp["shares"].([]interface{})
	require.True(t, ok)
	assert.Len(t, shares, 1)
	first := shares[0].(map[string]interface{})
	assert.Equal(t, createdID, first["id"])
}

func TestListShares_EmptyList(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "noshareguy", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "noshareguy", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/share/get-shares", nil, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := testutil.DecodeData(t, rec)
	shares, ok := resp["shares"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, shares)
}

func TestListShares_OnlyOwnShares(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "alice", "pass123", "user")
	testutil.CreateTestUser(t, db, "bob", "pass456", "user")

	aliceCookie := testutil.LoginAndGetCookie(t, router, "alice", "pass123")
	bobCookie := testutil.LoginAndGetCookie(t, router, "bob", "pass456")

	// alice creates a share
	body := map[string]interface{}{
		"path":             "alice_stuff",
		"description":      "alice share",
		"expires_in_hours": 24,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, aliceCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	// bob creates a share
	body2 := map[string]interface{}{
		"path":             "bob_stuff",
		"description":      "bob share",
		"expires_in_hours": 24,
	}
	recBobCreate := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body2, bobCookie)
	require.Equal(t, http.StatusOK, recBobCreate.Code)

	// bob lists — should only see his own
	recBob := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/share/get-shares", nil, bobCookie)
	require.Equal(t, http.StatusOK, recBob.Code)

	listResp := testutil.DecodeData(t, recBob)
	shares := listResp["shares"].([]interface{})
	assert.Len(t, shares, 1)
	first := shares[0].(map[string]interface{})
	assert.Equal(t, "bob", first["username"])
}

// ---------------------------------------------------------------------------
// Toggle Block
// ---------------------------------------------------------------------------

func TestToggleBlock_Success(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "blocker", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "blocker", "pass123")

	// create share
	body := map[string]interface{}{
		"path":             "toggledir",
		"description":      "toggle test",
		"expires_in_hours": 24,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)
	createResp := testutil.DecodeData(t, rec)
	share := createResp["share"].(map[string]interface{})
	shareID := share["id"].(string)

	// toggle block (block it)
	rec2 := testutil.MakeAuthRequest(t, router, http.MethodPut, "/api/user/share/"+shareID+"/toggle-block", nil, accessCookie)
	require.Equal(t, http.StatusOK, rec2.Code)
	toggleResp := testutil.DecodeData(t, rec2)
	assert.Equal(t, "Status updated", toggleResp["message"])
	assert.True(t, toggleResp["blocked"].(bool))

	// toggle again (unblock)
	rec3 := testutil.MakeAuthRequest(t, router, http.MethodPut, "/api/user/share/"+shareID+"/toggle-block", nil, accessCookie)
	require.Equal(t, http.StatusOK, rec3.Code)
	toggleResp2 := testutil.DecodeData(t, rec3)
	assert.False(t, toggleResp2["blocked"].(bool))
}

func TestToggleBlock_NotFound(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "blocker", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "blocker", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodPut, "/api/user/share/nonexistent/toggle-block", nil, accessCookie)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "not found")
}

// ---------------------------------------------------------------------------
// Delete Share
// ---------------------------------------------------------------------------

func TestDeleteShare_Success(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "deleter", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "deleter", "pass123")

	// create share
	body := map[string]interface{}{
		"path":             "deleteme",
		"description":      "to delete",
		"expires_in_hours": 24,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, accessCookie)
	require.Equal(t, http.StatusOK, rec.Code)
	createResp := testutil.DecodeData(t, rec)
	share := createResp["share"].(map[string]interface{})
	shareID := share["id"].(string)

	// delete
	rec2 := testutil.MakeAuthRequest(t, router, http.MethodDelete, "/api/user/share/"+shareID, nil, accessCookie)
	require.Equal(t, http.StatusOK, rec2.Code)
	deleteResp := testutil.DecodeData(t, rec2)
	assert.Equal(t, "Share link deleted", deleteResp["message"])

	// verify it's really gone
	rec3 := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/share/get-shares", nil, accessCookie)
	require.Equal(t, http.StatusOK, rec3.Code)
	listResp := testutil.DecodeData(t, rec3)
	assert.Empty(t, listResp["shares"].([]interface{}))
}

func TestDeleteShare_NotFound(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "deleter", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "deleter", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodDelete, "/api/user/share/nonexistent-id", nil, accessCookie)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "not found")
}

func TestDeleteShare_NotOwn(t *testing.T) {
	router, db := setupShareRouter(t)
	testutil.CreateTestUser(t, db, "alice_del", "pass123", "user")
	testutil.CreateTestUser(t, db, "bob_del", "pass456", "user")

	aliceCookie := testutil.LoginAndGetCookie(t, router, "alice_del", "pass123")
	bobCookie := testutil.LoginAndGetCookie(t, router, "bob_del", "pass456")

	// alice creates a share
	body := map[string]interface{}{
		"path":             "alice_private",
		"description":      "alice only",
		"expires_in_hours": 24,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/share/create", body, aliceCookie)
	require.Equal(t, http.StatusOK, rec.Code)
	createResp := testutil.DecodeData(t, rec)
	share := createResp["share"].(map[string]interface{})
	aliceShareID := share["id"].(string)

	// bob tries to delete alice's share
	rec2 := testutil.MakeAuthRequest(t, router, http.MethodDelete, "/api/user/share/"+aliceShareID, nil, bobCookie)
	assert.Equal(t, http.StatusNotFound, rec2.Code)

	var deleteResp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &deleteResp))
	assert.Contains(t, deleteResp["error"], "not found")
}
