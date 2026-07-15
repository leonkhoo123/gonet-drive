package controller_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileProperties_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "props.txt", "property content")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/properties",
		map[string]interface{}{
			"sources": []string{"/props.txt"},
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
	assert.Equal(t, "file", resp["type"])
	assert.NotNil(t, resp["name"])
	assert.Equal(t, float64(16), resp["totalSizeBytes"])
	assert.Equal(t, "/", resp["location"])
	assert.NotNil(t, resp["modifiedAt"])
}

func TestFileProperties_MultipleFiles(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "a/file1.txt", "content1")
	writeTestFile(t, cfg.Server.FileRoot, "b/file2.txt", "content2")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/properties",
		map[string]interface{}{
			"sources": []string{"/a/file1.txt", "/b/file2.txt"},
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
	assert.Equal(t, "multiple", resp["type"])
	assert.Nil(t, resp["name"])
	assert.Equal(t, "Multiple Locations", resp["location"])
}

func TestFileProperties_NotFound(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/properties",
		map[string]interface{}{
			"sources": []string{"/nonexistent.txt"},
		},
		accessCookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotNil(t, resp["error"])
}

func TestFileProperties_PathTraversal(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/properties",
		map[string]interface{}{
			"sources": []string{"../"},
		},
		accessCookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCheckDuplicates(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "src/dup.txt", "same content")
	writeTestFile(t, cfg.Server.FileRoot, "dest/dup.txt", "same content")
	writeTestFile(t, cfg.Server.FileRoot, "src/unique.txt", "different")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/check-duplicates",
		map[string]interface{}{
			"sources": []string{"/src/dup.txt", "/src/unique.txt"},
			"destDir": "/dest",
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
	assert.Equal(t, true, resp["hasDuplicates"])

	duplicates := resp["duplicates"].([]interface{})
	assert.Len(t, duplicates, 1)
	dup := duplicates[0].(map[string]interface{})
	source := dup["source"].(map[string]interface{})
	assert.Equal(t, "dup.txt", source["name"])
}

func TestStorageUsage_Authenticated(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/storage-usage", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
	assert.NotNil(t, resp["used"])
	assert.NotNil(t, resp["limit"])
}

func TestCancelOperation_Exists(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/cancel",
		map[string]interface{}{
			"opId": "nonexistent-op-id",
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
}
