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

func TestCopyFiles_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "src/file.txt", "copy content")
	if err := os.MkdirAll(filepath.Join(cfg.Server.FileRoot, "dest"), 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/copy",
		map[string]interface{}{
			"sources": []string{"/src/file.txt"},
			"destDir": "/dest",
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	waitForJobQueue(t)

	_, err := os.Stat(filepath.Join(cfg.Server.FileRoot, "src/file.txt"))
	assert.NoError(t, err, "source file should still exist after copy")

	_, err = os.Stat(filepath.Join(cfg.Server.FileRoot, "dest/file.txt"))
	assert.NoError(t, err, "copied file should exist at destination")
}

func TestCopyFiles_SameDir(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "same/file.txt", "copy content")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/copy",
		map[string]interface{}{
			"sources": []string{"/same/file.txt"},
			"destDir": "/same",
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	waitForJobQueue(t)

	entries, err := os.ReadDir(filepath.Join(cfg.Server.FileRoot, "same"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2, "should have original + copy")
}

func TestCopyFiles_PathTraversal(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/copy",
		map[string]interface{}{
			"sources": []string{"../"},
			"destDir": "/",
		},
		accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMoveFiles_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "src/file.txt", "move content")
	if err := os.MkdirAll(filepath.Join(cfg.Server.FileRoot, "dest"), 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/move",
		map[string]interface{}{
			"sources": []string{"/src/file.txt"},
			"destDir": "/dest",
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	waitForJobQueue(t)

	_, err := os.Stat(filepath.Join(cfg.Server.FileRoot, "src/file.txt"))
	assert.True(t, os.IsNotExist(err), "source file should be gone after move")

	_, err = os.Stat(filepath.Join(cfg.Server.FileRoot, "dest/file.txt"))
	assert.NoError(t, err, "moved file should exist at destination")
}

func TestMoveFiles_SameDir(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "same/file.txt", "move content")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/move",
		map[string]interface{}{
			"sources": []string{"/same/file.txt"},
			"destDir": "/same",
		},
		accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "cannot move items to the same directory")
}
