package controller_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileUpload_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("identifier", "test-upload-1")
	w.WriteField("status", "end")
	w.WriteField("filename", "uploaded.txt")
	w.WriteField("destination", "/")
	w.WriteField("chunkNumber", "1")
	w.WriteField("totalChunks", "1")
	fw, _ := w.CreateFormFile("chunk", "uploaded.txt")
	fw.Write([]byte("hello upload world"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/user/files/upload-chunk", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "done", resp["status"])

	_, err := os.Stat(filepath.Join(cfg.Server.FileRoot, "uploaded.txt"))
	assert.NoError(t, err, "uploaded file should exist")

	content, err := os.ReadFile(filepath.Join(cfg.Server.FileRoot, "uploaded.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "hello upload world", string(content))
}

func TestFileUpload_EmptyFile(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("identifier", "test-upload-empty")
	w.WriteField("status", "end")
	w.WriteField("filename", "empty.txt")
	w.WriteField("destination", "/")
	w.WriteField("chunkNumber", "1")
	w.WriteField("totalChunks", "1")
	fw, _ := w.CreateFormFile("chunk", "empty.txt")
	fw.Write([]byte{})
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/user/files/upload-chunk", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "done", resp["status"])

	_, err := os.Stat(filepath.Join(cfg.Server.FileRoot, "empty.txt"))
	assert.NoError(t, err, "empty file should exist")
}

func TestFileUpload_PathTraversal(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("identifier", "test-traversal")
	w.WriteField("status", "end")
	w.WriteField("filename", "test.txt")
	w.WriteField("destination", "../evil")
	w.WriteField("chunkNumber", "1")
	w.WriteField("totalChunks", "1")
	fw, _ := w.CreateFormFile("chunk", "test.txt")
	fw.Write([]byte("bad"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/user/files/upload-chunk", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"].(string), "Invalid destination")
}

func TestFileUpload_PathTraversal_Filename(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("identifier", "test-filename-traversal")
	w.WriteField("status", "end")
	w.WriteField("filename", "../bad.txt")
	w.WriteField("destination", "/")
	w.WriteField("chunkNumber", "1")
	w.WriteField("totalChunks", "1")
	fw, _ := w.CreateFormFile("chunk", "../bad.txt")
	fw.Write([]byte("bad"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/user/files/upload-chunk", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(accessCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	filePath := filepath.Join(cfg.Server.FileRoot, "bad.txt")
	_, err := os.Stat(filePath)
	assert.NoError(t, err, "file should be saved with sanitized name 'bad.txt'")

	_, err = os.Stat(filepath.Join(cfg.Server.FileRoot, "../bad.txt"))
	assert.True(t, os.IsNotExist(err), "unsanitized path should not exist")
}

func TestFileUpload_Unauthenticated(t *testing.T) {
	router, _, _ := setupFileRouter(t)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("identifier", "test-noauth")
	w.WriteField("status", "end")
	w.WriteField("filename", "test.txt")
	w.WriteField("destination", "/")
	w.WriteField("chunkNumber", "1")
	w.WriteField("totalChunks", "1")
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/user/files/upload-chunk", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	testutil.AssertAuthError(t, rec, http.StatusUnauthorized)
}

func TestFileDownload_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	content := "download content here"
	writeTestFile(t, cfg.Server.FileRoot, "download.txt", content)

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/download?source=/download.txt", nil, accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, content, rec.Body.String())
}

func TestFileDownload_NotFound(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/download?source=/nonexistent.txt", nil, accessCookie)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestFileDownload_PathTraversal(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/files/download?source=../etc/passwd", nil, accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
