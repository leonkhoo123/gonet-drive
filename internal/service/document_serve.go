package service

import (
	"net/http"
	"os"

	"go-file-server/internal/config"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

// ServeDocument serves text and document files
func ServeDocument(c *gin.Context, cfg *config.CloudConfig) {
	relPath := c.Param("filepath")
	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, relPath)
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	file, err := os.Open(fullPath)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Read content and serve as text
	// Let gin handle Content-Type sensing or we could force text/plain
	// if we wanted to prevent browser rendering for HTML.
	// For API usage, the frontend will just fetch it, so c.File is fine.
	c.File(fullPath)
}
