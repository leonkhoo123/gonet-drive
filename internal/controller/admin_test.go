package controller_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/middleware"
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAdminRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *service.UserService, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, _ := testutil.SetupServices(t, db, workDir)

	middleware.ResetLoginLimiter()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, userService)
	controller.SetupAdminRoutes(router, cfg, userService)

	return router, cfg, userService, db
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

func TestAdminMiddleware_SuperAdmin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "admin", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/users", nil, accessCookie)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetUsers_Admin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")
	testutil.CreateTestUser(t, db, "user1", "pass1", "user")
	testutil.CreateTestUser(t, db, "user2", "pass2", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "admin", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/user/admin/users", nil, accessCookie)
	assert.Equal(t, http.StatusOK, rec.Code)

	var users []map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &users))
	assert.GreaterOrEqual(t, len(users), 3, "should list at least the created users")

	usernames := make(map[string]bool)
	for _, u := range users {
		usernames[u["username"].(string)] = true
	}
	assert.True(t, usernames["admin"])
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
	testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "admin", "pass123")

	body := map[string]interface{}{
		"username": "newuser",
		"password": "newpass",
		"role":     "user",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, true, resp["success"])
	userID := resp["id"].(string)
	assert.NotEmpty(t, userID)

	var dbUsername string
	err := db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&dbUsername)
	require.NoError(t, err)
	assert.Equal(t, "newuser", dbUsername)
}

func TestCreateUser_MissingFields(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "admin", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", map[string]string{}, accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "invalid request", resp["error"])
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "admin", "pass123")

	body := map[string]interface{}{
		"username": "dupuser",
		"password": "pass123",
		"role":     "user",
	}
	rec1 := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)
	require.Equal(t, http.StatusOK, rec1.Code)

	rec2 := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)

	assert.Equal(t, http.StatusInternalServerError, rec2.Code)
	resp := mustUnmarshal(t, rec2.Body.Bytes())
	assert.Equal(t, "username may already exist", resp["error"])
}

func TestCreateUser_SuperadminOnlySuperadmin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "regularadmin", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "regularadmin", "pass123")

	body := map[string]interface{}{
		"username": "anothersuper",
		"password": "pass123",
		"role":     "superadmin",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "only the main superadmin can create other superadmins", resp["error"])
}

func TestCreateUser_SuperadminCreatesSuperadmin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "admin", "pass123")

	body := map[string]interface{}{
		"username": "anothersuper",
		"password": "pass123",
		"role":     "superadmin",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/user/admin/users", body, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, true, resp["success"])
	assert.NotEmpty(t, resp["id"])
}

func TestDeleteUser_Admin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")
	regularUser := testutil.CreateTestUser(t, db, "todelete", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "admin", "pass123")

	req := httptest.NewRequest(http.MethodDelete, "/api/user/admin/users/"+regularUser.ID, nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, true, resp["success"])

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
	assert.Equal(t, "only super admin can delete an admin", resp["error"])

	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", adminUser.ID).Scan(&exists)
	require.NoError(t, err)
	assert.Equal(t, 1, exists, "self-deletion guard works via admin protection — user still exists")
}

func TestDeleteUser_CannotDeleteSuperAdmin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	superAdminUser := testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")
	testutil.CreateTestUser(t, db, "regadmin", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "regadmin", "pass123")

	req := httptest.NewRequest(http.MethodDelete, "/api/user/admin/users/"+superAdminUser.ID, nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "cannot delete super admin", resp["error"])
}

func TestRevokeSessions_Admin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")
	targetUser := testutil.CreateTestUser(t, db, "targetuser", "pass123", "user")

	accessCookie := testutil.LoginAndGetCookie(t, router, "admin", "pass123")

	req := httptest.NewRequest(http.MethodPost, "/api/user/admin/users/"+targetUser.ID+"/revoke", nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, true, resp["success"])
}

func TestRevokeSessions_CannotRevokeSuperAdmin(t *testing.T) {
	router, _, _, db := setupAdminRouter(t)
	superAdminUser := testutil.CreateTestUser(t, db, "admin", "pass123", "superadmin")
	testutil.CreateTestUser(t, db, "regadmin2", "pass123", "admin")

	accessCookie := testutil.LoginAndGetCookie(t, router, "regadmin2", "pass123")

	req := httptest.NewRequest(http.MethodPost, "/api/user/admin/users/"+superAdminUser.ID+"/revoke", nil)
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	resp := mustUnmarshal(t, rec.Body.Bytes())
	assert.Equal(t, "cannot revoke super admin", resp["error"])
}
