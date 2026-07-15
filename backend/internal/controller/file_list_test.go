package controller_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileList_EmptyDir(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/file-list?path=/", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
	assert.Equal(t, "/", resp["path"])
	assert.Equal(t, float64(0), resp["count"])
	assert.Equal(t, float64(0), resp["file_count"])
	assert.Equal(t, float64(0), resp["folder_count"])
	items := resp["items"]
	assert.Empty(t, items, "items should be empty")
}

func TestFileList_WithFiles(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "alpha.txt", "hello")
	writeTestFile(t, cfg.Server.FileRoot, "beta.txt", "world")
	if err := os.MkdirAll(filepath.Join(cfg.Server.FileRoot, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/file-list?path=/", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})

	items := resp["items"].([]interface{})
	assert.Len(t, items, 3)
	assert.Equal(t, float64(2), resp["file_count"])
	assert.Equal(t, float64(1), resp["folder_count"])
	assert.Equal(t, float64(3), resp["count"])

	names := make(map[string]bool)
	for _, item := range items {
		m := item.(map[string]interface{})
		names[m["name"].(string)] = true
	}
	assert.True(t, names["alpha.txt"])
	assert.True(t, names["beta.txt"])
	assert.True(t, names["subdir"])
}

func TestFileList_PathTraversal(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/file-list?path=../", nil, accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestFileList_Unauthenticated(t *testing.T) {
	router, _, _ := setupFileRouter(t)

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/file-list?path=/", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	testutil.AssertAuthError(t, rec, http.StatusUnauthorized)
}
