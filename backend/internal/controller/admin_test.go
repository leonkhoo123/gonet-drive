package controller_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/testutil"

	"github.com/leonkhoo123/gonet-auth/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAdminRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *auth.Auth, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)
	controller.SetupAuthenticatedRoutes(router, cfg, authInstance, authCfg, userService, nil, nil, nil, nil)

	return router, cfg, authInstance, db
}

func mustUnmarshal(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func TestAdminMiddleware_NoToken(t *testing.T) {
	router, _, _, _ := setupAdminRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/admin/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	testutil.AssertAuthError(t, rec, http.StatusUnauthorized)
}

func TestAdminMiddleware_RegularUser(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "regularuser", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "regularuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/users", nil, accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "admin access required", resp["error"])
}

func TestAdminMiddleware_Admin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "regularadmin", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "regularadmin", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/users", nil, accessCookie)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetUsers_Admin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	testutil.CreateTestUser(t, db, "user1", "pass1", "user")
	testutil.CreateTestUser(t, db, "user2", "pass2", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/users", nil, accessCookie)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Users []map[string]interface{} `json:"users"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp.Status)
	users := resp.Data.Users
	assert.GreaterOrEqual(t, len(users), 3, "should list at least the created users")

	usernames := make(map[string]bool)
	for _, u := range users {
		usernames[u["username"].(string)] = true
	}
	assert.True(t, usernames["adminuser"])
	assert.True(t, usernames["user1"])
	assert.True(t, usernames["user2"])
}

func TestGetUsers_NonAdmin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "regularuser", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "regularuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/users", nil, accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "admin access required", resp["error"])
}

func TestCreateUser_Admin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	body := map[string]interface{}{
		"username": "newuser",
		"password": "newpass1234",
		"role":     "user",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "success", resp["status"])
	data, _ := resp["data"].(map[string]interface{})
	require.NotNil(t, data)
	user, _ := data["user"].(map[string]interface{})
	require.NotNil(t, user)
	userID := user["id"].(string)
	assert.NotEmpty(t, userID)

	var dbUsername string
	err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&dbUsername)
	require.NoError(t, err)
	assert.Equal(t, "newuser", dbUsername)
}

func TestCreateUser_MissingFields(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", map[string]string{}, accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "invalid request", resp["error"])
}

func TestCreateUser_WeakPassword(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	body := map[string]interface{}{
		"username": "weakuser",
		"password": "short",
		"role":     "user",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "password too weak (min 8 chars)", resp["error"])
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	body := map[string]interface{}{
		"username": "dupuser",
		"password": "pass12345",
		"role":     "user",
	}
	rec1 := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)
	require.Equal(t, http.StatusCreated, rec1.Code)

	rec2 := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)

	assert.Equal(t, http.StatusConflict, rec2.Code)
	resp := mustUnmarshal(t, rec2.Body.Bytes())
	assert.Equal(t, "username already taken", resp["error"])
}

func TestDeleteUser_Admin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	regularUser := testutil.CreateTestUser(t, db, "todelete", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	req := httptest.NewRequest(http.MethodDelete, "/api/user/admin/users/"+regularUser.ID, nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "success", resp["status"])

	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", regularUser.ID).Scan(&exists)
	require.NoError(t, err)
	assert.Equal(t, 0, exists, "user should be deleted from database")
}

func TestDeleteUser_CannotDeleteSelf(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	adminUser := testutil.CreateTestUser(t, db, "selfadmin", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "selfadmin", "pass123")

	req := httptest.NewRequest(http.MethodDelete, "/api/user/admin/users/"+adminUser.ID, nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "cannot delete your own account", resp["error"])

	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", adminUser.ID).Scan(&exists)
	require.NoError(t, err)
	assert.Equal(t, 1, exists, "user should still exist after self-delete attempt")
}

func TestDeleteUser_AdminCanDeleteOtherAdmin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminone", "pass123", "admin")
	otherAdmin := testutil.CreateTestUser(t, db, "admintwo", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminone", "pass123")

	req := httptest.NewRequest(http.MethodDelete, "/api/user/admin/users/"+otherAdmin.ID, nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "success", resp["status"])

	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", otherAdmin.ID).Scan(&exists)
	require.NoError(t, err)
	assert.Equal(t, 0, exists, "the other admin should be deleted")
}

func TestRevokeSessions_Admin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminuser", "pass123", "admin")
	targetUser := testutil.CreateTestUser(t, db, "targetuser", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminuser", "pass123")

	req := httptest.NewRequest(http.MethodPost, "/api/user/admin/users/"+targetUser.ID+"/revoke-all", nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "success", resp["status"])
}

func TestRevokeSessions_CannotRevokeAdmin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "adminone", "pass123", "admin")
	otherAdmin := testutil.CreateTestUser(t, db, "admintwo", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "adminone", "pass123")

	req := httptest.NewRequest(http.MethodPost, "/api/user/admin/users/"+otherAdmin.ID+"/revoke-all", nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "cannot bulk revoke admin sessions; revoke one by one", resp["error"])
}
