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

func TestCreateFolder_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/create-folder",
		map[string]interface{}{"dir": "/", "folderName": "newfolder"},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Folder created successfully", resp["message"])

	fullPath := filepath.Join(cfg.Server.FileRoot, "newfolder")
	info, err := os.Stat(fullPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCreateFolder_AlreadyExists(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	if err := os.MkdirAll(filepath.Join(cfg.Server.FileRoot, "existing"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/create-folder",
		map[string]interface{}{"dir": "/", "folderName": "existing"},
		accessCookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "already exists")
}

func TestCreateFolder_InvalidName(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/create-folder",
		map[string]interface{}{"dir": "/", "folderName": ".."},
		accessCookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCreateFolder_ProtectedName(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/create-folder",
		map[string]interface{}{"dir": "/", "folderName": ".cloud_delete"},
		accessCookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRename_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "oldname.txt", "content")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/rename",
		map[string]interface{}{"source": "/oldname.txt", "newName": "newname.txt"},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "File renamed successfully", resp["message"])

	_, err := os.Stat(filepath.Join(cfg.Server.FileRoot, "oldname.txt"))
	assert.True(t, os.IsNotExist(err), "old file should not exist")

	_, err = os.Stat(filepath.Join(cfg.Server.FileRoot, "newname.txt"))
	assert.NoError(t, err, "new file should exist")
}

func TestRename_ProtectedFile(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	reservePath := filepath.Join(cfg.Server.FileRoot, ".cloud_reserve")
	if err := os.MkdirAll(reservePath, 0755); err != nil {
		t.Fatalf("failed to create .cloud_reserve: %v", err)
	}

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/rename",
		map[string]interface{}{"source": "/.cloud_reserve", "newName": "newname"},
		accessCookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "cannot rename protected system folders")
}

func TestRename_SameName(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "file.txt", "content")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/rename",
		map[string]interface{}{"source": "/file.txt", "newName": "file.txt"},
		accessCookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
