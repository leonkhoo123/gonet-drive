package controller_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/middleware"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"
	"go-file-server/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var startWorkerOnce sync.Once
var startWSOnce sync.Once

type shareFileFixture struct {
	Router   *gin.Engine
	WorkDir  string
	Share    *model.SharingInfo
	ShareJWT string
}

func setupShareFileRouter(t *testing.T, authority string) *shareFileFixture {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot

	// Start the background job worker (idempotent via sync.Once)
	startWorkerOnce.Do(service.StartFileOperationWorker)
	// Start the WebSocket broadcast manager (idempotent via sync.Once)
	startWSOnce.Do(func() { go ws.Manager.Start() })

	shareRepo := repository.NewSQLiteSharingRepo(db)
	_, _, _, authInstance, _ := testutil.SetupServices(t, db, workDir)

	// Create test directory structure inside workdir
	sharedDir := filepath.Join(workDir, "shared_subdir")
	require.NoError(t, os.MkdirAll(sharedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "readme.txt"), []byte("hello share world\n"), 0o644))

	nestedDir := filepath.Join(sharedDir, "nested")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "data.bin"), []byte("binary data"), 0o644))

	// Create directory outside the shared path for traversal tests
	outsideDir := filepath.Join(workDir, "outside")
	require.NoError(t, os.MkdirAll(outsideDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0o644))

	// Create share in DB
	shareID := "share-file-test-" + authority
	share := &model.SharingInfo{
		ID:          shareID,
		Path:        "shared_subdir",
		PinHash:     "$2a$10$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Blocked:     false,
		Authority:   authority,
		Username:    "fileowner",
		Description: "share file test",
		CreatedAt:   time.Now(),
	}
	require.NoError(t, shareRepo.Create(share))

	// Generate share JWT signed with the library-managed secret.
	tokenStr, err := middleware.GenerateShareJWT(authInstance.JWT, share.ID, share.Path, share.Authority, cfg.Auth.ShareJwtMaxAge)
	require.NoError(t, err)

	router := gin.New()
	controller.SetupShareFileRoutes(router, shareRepo, authInstance)

	return &shareFileFixture{
		Router:   router,
		WorkDir:  workDir,
		Share:    share,
		ShareJWT: tokenStr,
	}
}

// makeShareFileReq makes a JSON request with share JWT cookie and share_id query param.
func makeShareFileReq(t *testing.T, fix *shareFileFixture, method, urlPath string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader *bytes.Buffer
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewBuffer(jsonBytes)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, urlPath, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	cookieName := "shareJwt_" + fix.Share.ID
	req.AddCookie(&http.Cookie{Name: cookieName, Value: fix.ShareJWT, Path: "/"})

	// Add share_id query param
	q := req.URL.Query()
	q.Set("share_id", fix.Share.ID)
	req.URL.RawQuery = q.Encode()

	rec := httptest.NewRecorder()
	fix.Router.ServeHTTP(rec, req)
	return rec
}

// makeShareFileReqNoToken makes a request without share JWT.
func makeShareFileReqNoToken(t *testing.T, fix *shareFileFixture, method, urlPath string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, urlPath, nil)
	q := req.URL.Query()
	q.Set("share_id", fix.Share.ID)
	req.URL.RawQuery = q.Encode()
	rec := httptest.NewRecorder()
	fix.Router.ServeHTTP(rec, req)
	return rec
}

// makeShareFileMultipartReq makes a multipart upload request with share JWT cookie and share_id query param.
func makeShareFileMultipartReq(t *testing.T, fix *shareFileFixture, method, urlPath, destination string, fileData []byte) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("identifier", "test-upload")
	_ = w.WriteField("destination", destination)
	_ = w.WriteField("filename", "testfile.txt")
	_ = w.WriteField("chunkNumber", "1")
	_ = w.WriteField("chunkSize", "1024")
	_ = w.WriteField("totalChunks", "1")
	_ = w.WriteField("fileSize", "4")
	part, err := w.CreateFormFile("chunk", "testfile.txt")
	require.NoError(t, err)
	_, err = part.Write(fileData)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(method, urlPath, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	cookieName := "shareJwt_" + fix.Share.ID
	req.AddCookie(&http.Cookie{Name: cookieName, Value: fix.ShareJWT, Path: "/"})
	q := req.URL.Query()
	q.Set("share_id", fix.Share.ID)
	req.URL.RawQuery = q.Encode()

	rec := httptest.NewRecorder()
	fix.Router.ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// Share File List
// ---------------------------------------------------------------------------

func TestShareFileList_Valid(t *testing.T) {
	fix := setupShareFileRouter(t, "view")

	rec := makeShareFileReq(t, fix, http.MethodGet, "/api/share/file/list", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "/", data["path"])
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(items), 1)
}

func TestShareFileList_PathTraversal(t *testing.T) {
	fix := setupShareFileRouter(t, "view")

	// Request a path that tries to escape the authorized root
	rec := makeShareFileReq(t, fix, http.MethodGet, "/api/share/file/list?path=../../outside", nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Path traversal")
}

func TestShareFileList_NoToken(t *testing.T) {
	fix := setupShareFileRouter(t, "view")

	rec := makeShareFileReqNoToken(t, fix, http.MethodGet, "/api/share/file/list")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------------------------------------------------------------------------
// Share File Download
// ---------------------------------------------------------------------------

func TestShareFileDownload_Valid(t *testing.T) {
	fix := setupShareFileRouter(t, "view")

	rec := makeShareFileReq(t, fix, http.MethodGet, "/api/share/file/download?source=readme.txt", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello share world\n", rec.Body.String())
}

func TestShareFileDownload_OutsidePath(t *testing.T) {
	fix := setupShareFileRouter(t, "view")

	rec := makeShareFileReq(t, fix, http.MethodGet, "/api/share/file/download?source=../../outside/secret.txt", nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Path traversal")
}

// ---------------------------------------------------------------------------
// Share File Modify — Denied for View
// ---------------------------------------------------------------------------

func TestShareFileModify_DeniedView(t *testing.T) {
	fix := setupShareFileRouter(t, "view")

	// Attempt rename on view-only share
	renameBody := map[string]string{
		"source":  "readme.txt",
		"newName": "newname.txt",
	}
	rec := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/rename", renameBody)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Attempt delete on view-only share
	deleteBody := map[string]interface{}{
		"sources": []string{"readme.txt"},
	}
	rec2 := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/delete", deleteBody)
	assert.Equal(t, http.StatusForbidden, rec2.Code)

	// Attempt upload on view-only share
	rec3 := makeShareFileMultipartReq(t, fix, http.MethodPost, "/api/share/file/upload-chunk", "readme.txt", []byte("data"))
	assert.Equal(t, http.StatusForbidden, rec3.Code)
}

// ---------------------------------------------------------------------------
// Share File Modify — Allowed for Modify
// ---------------------------------------------------------------------------

func TestShareFileModify_AllowedModify(t *testing.T) {
	fix := setupShareFileRouter(t, "modify")

	// Test rename
	renameBody := map[string]string{
		"source":  "readme.txt",
		"newName": "renamed.txt",
	}
	rec := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/rename", renameBody)
	assert.Equal(t, http.StatusOK, rec.Code)
	resp := testutil.DecodeData(t, rec)
	assert.Equal(t, "File renamed successfully", resp["message"])

	// Verify file was renamed
	_, err := os.Stat(filepath.Join(fix.WorkDir, "shared_subdir", "renamed.txt"))
	assert.NoError(t, err)

	// Test create folder
	folderBody := map[string]string{
		"dir":        "",
		"folderName": "newfolder",
	}
	rec2 := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/create-folder", folderBody)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// Verify folder was created
	info, err := os.Stat(filepath.Join(fix.WorkDir, "shared_subdir", "newfolder"))
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Test copy
	copyBody := map[string]interface{}{
		"sources": []string{"renamed.txt"},
		"destDir": "newfolder",
	}
	rec3 := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/copy", copyBody)
	assert.Equal(t, http.StatusOK, rec3.Code)

	// Copy is async via job queue — poll for completion
	require.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(fix.WorkDir, "shared_subdir", "newfolder", "renamed.txt"))
		return err == nil
	}, 5*time.Second, 50*time.Millisecond)

	// Test move
	moveBody := map[string]interface{}{
		"sources": []string{"nested"},
		"destDir": "newfolder",
	}
	rec4 := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/move", moveBody)
	assert.Equal(t, http.StatusOK, rec4.Code)

	// Move is async via job queue — poll for completion
	require.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(fix.WorkDir, "shared_subdir", "newfolder", "nested"))
		return err == nil
	}, 5*time.Second, 50*time.Millisecond)

	// Test delete (soft delete moves to recycle)
	deleteBody := map[string]interface{}{
		"sources": []string{"renamed.txt"},
	}
	rec5 := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/delete", deleteBody)
	assert.Equal(t, http.StatusOK, rec5.Code)

	// Soft delete is async — poll until the file is moved to recycle
	require.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(fix.WorkDir, "shared_subdir", "renamed.txt"))
		return os.IsNotExist(err)
	}, 5*time.Second, 50*time.Millisecond)
}

// ---------------------------------------------------------------------------
// Share File Rename — edge cases
// ---------------------------------------------------------------------------

func TestShareFileRename_Valid(t *testing.T) {
	fix := setupShareFileRouter(t, "modify")

	body := map[string]string{
		"source":  "readme.txt",
		"newName": "new_readme.txt",
	}
	rec := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/rename", body)
	assert.Equal(t, http.StatusOK, rec.Code)

	resp := testutil.DecodeData(t, rec)
	assert.Equal(t, "File renamed successfully", resp["message"])
}

// ---------------------------------------------------------------------------
// Share File Delete
// ---------------------------------------------------------------------------

func TestShareFileDelete_Authored(t *testing.T) {
	fix := setupShareFileRouter(t, "modify")

	body := map[string]interface{}{
		"sources": []string{"readme.txt"},
	}
	rec := makeShareFileReq(t, fix, http.MethodPost, "/api/share/file/delete", body)
	assert.Equal(t, http.StatusOK, rec.Code)

	resp := testutil.DecodeData(t, rec)
	assert.Equal(t, "Files deleted successfully", resp["message"])

	// Soft delete is async — poll until the file is moved to recycle
	require.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(fix.WorkDir, "shared_subdir", "readme.txt"))
		return os.IsNotExist(err)
	}, 5*time.Second, 50*time.Millisecond)
}
