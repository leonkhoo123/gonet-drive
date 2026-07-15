package controller_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"
	"go-file-server/internal/ws"

	"github.com/gin-gonic/gin"
)

func setupFileRouter(t *testing.T) (*gin.Engine, *config.CloudConfig, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot
	userService, _, _, authInstance, authCfg := testutil.SetupServices(t, db, workDir)

	controller.ResetLoginLimiterForTest()

	service.JobQueue = make(chan service.Job, 100)
	service.StartFileOperationWorker()

	go ws.Manager.Start()

	router := gin.New()
	controller.SetupPublicAuthRoutes(router, cfg, authInstance, authCfg)
	controller.SetupAuthenticatedRoutes(router, cfg, authInstance, authCfg, userService, nil, nil, nil, nil)

	return router, cfg, db
}

func waitForJobQueue(t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	service.JobQueue <- func() {
		close(done)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for job queue to drain")
	}
}

func writeTestFile(t *testing.T, root, path, content string) string {
	t.Helper()
	fullPath := filepath.Join(root, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", dir, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", fullPath, err)
	}
	return fullPath
}
