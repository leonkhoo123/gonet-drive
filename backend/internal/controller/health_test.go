package controller_test

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go-file-server/internal/config"
	"go-file-server/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupHealthRouter(t *testing.T) *gin.Engine {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig

	router := gin.New()

	router.GET("/api/health", func(c *gin.Context) {
		cloudConfig := config.AppCloudConfig
		if cloudConfig == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cloud config not available"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":            "OK",
			"service_name":      cloudConfig.ServiceName,
			"upload_chunk_size": cloudConfig.UploadChunkSize,
			"video_mode":        cfg.Server.VideoMode,
		})
	})

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}
		c.String(http.StatusOK, "fallback page")
	})

	db.Close() // unused after router setup
	return router
}

func setupHealthRouterWithSPA(t *testing.T) *gin.Engine {
	t.Helper()
	db := testutil.SetupTestDB(t)

	// Create a temp dir with index.html for SPA fallback testing
	spaDir := t.TempDir()
	indexContent := "<html><head><title>Test SPA</title></head><body>Hello</body></html>"
	err := os.WriteFile(filepath.Join(spaDir, "index.html"), []byte(indexContent), 0644)
	require.NoError(t, err)

	distFS := os.DirFS(spaDir)
	fileServer := http.FileServer(http.FS(distFS))

	router := gin.New()

	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}
		trimmed := path
		if len(trimmed) > 0 && trimmed[0] == '/' {
			trimmed = trimmed[1:]
		}
		if _, err := fs.Stat(distFS, trimmed); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	db.Close()
	return router
}

func TestHealthEndpoint_NoAuth(t *testing.T) {
	router := setupHealthRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "OK", resp["status"])
	assert.Equal(t, "GoNet Drive Test", resp["service_name"])
}

func TestHealthEndpoint_NoRoute(t *testing.T) {
	router := setupHealthRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/xxx", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "API route not found", resp["error"])
}

func TestSPAFallback(t *testing.T) {
	router := setupHealthRouterWithSPA(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "<html>")
	assert.Contains(t, body, "Test SPA")
}

func TestSPAFallback_SubPage(t *testing.T) {
	router := setupHealthRouterWithSPA(t)

	req := httptest.NewRequest(http.MethodGet, "/some-page", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "<html>")
	assert.Contains(t, body, "Test SPA")
}
