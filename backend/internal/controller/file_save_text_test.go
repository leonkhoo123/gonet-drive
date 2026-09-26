package controller_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveBaseline(t *testing.T, root, rel string) (int64, string) {
	t.Helper()
	info, err := os.Stat(filepath.Join(root, rel))
	require.NoError(t, err)
	return info.Size(), info.ModTime().Format(time.RFC3339Nano)
}

func TestSaveText_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	full := writeTestFile(t, cfg.Server.FileRoot, "note.txt", "original")
	size, modified := saveBaseline(t, cfg.Server.FileRoot, "note.txt")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/save-text",
		map[string]interface{}{
			"path":         "/note.txt",
			"content":      "updated content",
			"allowEmpty":   false,
			"baseSize":     size,
			"baseModified": modified,
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)

	onDisk, err := os.ReadFile(full)
	require.NoError(t, err)
	assert.Equal(t, "updated content", string(onDisk))

	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
	assert.Equal(t, float64(len("updated content")), resp["size"])
	assert.NotEmpty(t, resp["modified"])
}

func TestSaveText_EmptyBlocked(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	full := writeTestFile(t, cfg.Server.FileRoot, "keep.txt", "do not delete")
	size, modified := saveBaseline(t, cfg.Server.FileRoot, "keep.txt")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/save-text",
		map[string]interface{}{
			"path":         "/keep.txt",
			"content":      "",
			"baseSize":     size,
			"baseModified": modified,
		},
		accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	onDisk, err := os.ReadFile(full)
	require.NoError(t, err)
	assert.Equal(t, "do not delete", string(onDisk))
}

func TestSaveText_MissingContentField(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "keep.txt", "do not delete")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/save-text",
		map[string]interface{}{
			"path": "/keep.txt",
		},
		accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSaveText_ConflictReturnsCurrentState(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "note.txt", "original")
	size, modified := saveBaseline(t, cfg.Server.FileRoot, "note.txt")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/save-text",
		map[string]interface{}{
			"path":         "/note.txt",
			"content":      "new",
			"baseSize":     size + 1,
			"baseModified": modified,
		},
		accessCookie)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
	assert.Equal(t, float64(len("original")), resp["size"])
	assert.NotEmpty(t, resp["modified"])
}

func TestSaveText_NonTextTargetRejected(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "movie.mp4", "binary-ish")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/save-text",
		map[string]interface{}{
			"path":    "/movie.mp4",
			"content": "text",
		},
		accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSaveText_NotFound(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/save-text",
		map[string]interface{}{
			"path":    "/missing.txt",
			"content": "x",
		},
		accessCookie)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSaveText_PathTraversalForbidden(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/save-text",
		map[string]interface{}{
			"path":    "../evil.txt",
			"content": "x",
		},
		accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
