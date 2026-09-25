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
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
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
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	resp := envelope["data"].(map[string]interface{})
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

// uploadChunkReq posts a single multipart chunk to the authenticated upload
// endpoint. fields carries identifier/status/filename/destination/chunkNumber/
// totalChunks; chunkData is the chunk body.
func uploadChunkReq(t *testing.T, handler http.Handler, cookie *http.Cookie, fields map[string]string, chunkData []byte) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for key, value := range fields {
		require.NoError(t, w.WriteField(key, value))
	}
	part, err := w.CreateFormFile("chunk", "chunk.bin")
	require.NoError(t, err)
	_, err = part.Write(chunkData)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/user/files/upload-chunk", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestFileUpload_FinalChunkReplayIsIdempotent covers the case where the merge
// succeeded but the client never received the response: replaying the final
// chunk must return the same path instead of failing with "Missing chunk".
func TestFileUpload_FinalChunkReplayIsIdempotent(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	fields := map[string]string{
		"identifier":  "replay-single-chunk",
		"status":      "end",
		"filename":    "replay.txt",
		"destination": "/",
		"chunkNumber": "1",
		"totalChunks": "1",
	}

	first := uploadChunkReq(t, router, accessCookie, fields, []byte("payload"))
	require.Equal(t, http.StatusOK, first.Code)
	firstResp := testutil.DecodeData(t, first)
	assert.Equal(t, "done", firstResp["status"])

	// Replay the exact same request as if the response had been lost.
	second := uploadChunkReq(t, router, accessCookie, fields, []byte("payload"))
	require.Equal(t, http.StatusOK, second.Code)
	secondResp := testutil.DecodeData(t, second)
	assert.Equal(t, "done", secondResp["status"])
	assert.Equal(t, firstResp["path"], secondResp["path"])

	matches, err := filepath.Glob(filepath.Join(cfg.Server.FileRoot, "replay*.txt"))
	require.NoError(t, err)
	assert.Len(t, matches, 1, "replay must not create a duplicate file")

	content, err := os.ReadFile(filepath.Join(cfg.Server.FileRoot, "replay.txt"))
	require.NoError(t, err)
	assert.Equal(t, "payload", string(content))
}

// TestFileUpload_MultiChunkReplayAfterMerge replays only the final chunk of a
// two-chunk upload after the merge already happened.
func TestFileUpload_MultiChunkReplayAfterMerge(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	base := map[string]string{
		"identifier":  "replay-multi-chunk",
		"filename":    "multi.bin",
		"destination": "/",
		"totalChunks": "2",
	}

	start := map[string]string{}
	for k, v := range base {
		start[k] = v
	}
	start["status"] = "start"
	start["chunkNumber"] = "1"
	require.Equal(t, http.StatusOK, uploadChunkReq(t, router, accessCookie, start, []byte("hello ")).Code)

	end := map[string]string{}
	for k, v := range base {
		end[k] = v
	}
	end["status"] = "end"
	end["chunkNumber"] = "2"
	first := uploadChunkReq(t, router, accessCookie, end, []byte("world"))
	require.Equal(t, http.StatusOK, first.Code)
	firstResp := testutil.DecodeData(t, first)
	assert.Equal(t, "done", firstResp["status"])

	// Replay the final chunk after a lost response.
	second := uploadChunkReq(t, router, accessCookie, end, []byte("world"))
	require.Equal(t, http.StatusOK, second.Code)
	secondResp := testutil.DecodeData(t, second)
	assert.Equal(t, "done", secondResp["status"])
	assert.Equal(t, firstResp["path"], secondResp["path"])

	matches, err := filepath.Glob(filepath.Join(cfg.Server.FileRoot, "multi*.bin"))
	require.NoError(t, err)
	assert.Len(t, matches, 1, "replay must not create a duplicate file")

	content, err := os.ReadFile(filepath.Join(cfg.Server.FileRoot, "multi.bin"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(content))
}

// TestFileUpload_StartResetsCompletedUpload verifies that a fresh upload reusing
// an identifier clears the completion marker instead of short-circuiting.
func TestFileUpload_StartResetsCompletedUpload(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	completed := map[string]string{
		"identifier":  "reused-identifier",
		"status":      "end",
		"filename":    "first.txt",
		"destination": "/",
		"chunkNumber": "1",
		"totalChunks": "1",
	}
	require.Equal(t, http.StatusOK, uploadChunkReq(t, router, accessCookie, completed, []byte("first")).Code)

	start := map[string]string{
		"identifier":  "reused-identifier",
		"status":      "start",
		"filename":    "second.bin",
		"destination": "/",
		"chunkNumber": "1",
		"totalChunks": "2",
	}
	rec := uploadChunkReq(t, router, accessCookie, start, []byte("second"))
	require.Equal(t, http.StatusOK, rec.Code)
	resp := testutil.DecodeData(t, rec)
	assert.Equal(t, "uploading", resp["status"], "start must not be treated as a completed replay")
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
