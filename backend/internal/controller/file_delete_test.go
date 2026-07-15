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

func TestDeleteSoft_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "todel.txt", "delete me")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/delete",
		map[string]interface{}{
			"sources": []string{"/todel.txt"},
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	waitForJobQueue(t)

	_, err := os.Stat(filepath.Join(cfg.Server.FileRoot, "todel.txt"))
	assert.True(t, os.IsNotExist(err), "file should be gone from original location")

	_, err = os.Stat(filepath.Join(cfg.Server.FileRoot, ".cloud_delete", "todel.txt"))
	assert.NoError(t, err, "file should be in .cloud_delete")
}

func TestDeleteSoft_ProtectedDir(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	reservePath := filepath.Join(cfg.Server.FileRoot, ".cloud_reserve")
	if err := os.MkdirAll(reservePath, 0755); err != nil {
		t.Fatalf("failed to create .cloud_reserve: %v", err)
	}

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/delete",
		map[string]interface{}{
			"sources": []string{"/.cloud_reserve"},
		},
		accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "failed to start delete operation")
}

func TestDeletePermanent_Success(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "permdel.txt", "delete me permanently")

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/delete-permanent",
		map[string]interface{}{
			"sources": []string{"/permdel.txt"},
		},
		accessCookie)

	assert.Equal(t, http.StatusOK, rec.Code)
	waitForJobQueue(t)

	_, err := os.Stat(filepath.Join(cfg.Server.FileRoot, "permdel.txt"))
	assert.True(t, os.IsNotExist(err), "file should be permanently gone")
}

func TestDeletePermanent_ProtectedDir(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	reservePath := filepath.Join(cfg.Server.FileRoot, ".cloud_reserve")
	if err := os.MkdirAll(reservePath, 0755); err != nil {
		t.Fatalf("failed to create .cloud_reserve: %v", err)
	}

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost,
		"/api/user/files/delete-permanent",
		map[string]interface{}{
			"sources": []string{"/.cloud_reserve", "/.cloud_delete"},
		},
		accessCookie)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "failed to start permanent delete operation")
}
