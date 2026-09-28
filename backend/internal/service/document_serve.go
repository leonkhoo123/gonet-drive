package service

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go-file-server/internal/config"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

// ServeDocument serves text and document files.
// @Summary      Serve Document
// @Description  Serve a text document file with appropriate MIME type.
// @Tags         Media
// @Produce      text/plain
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true  "Relative file path"
// @Success      200  {file}  binary
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/document/read/file/{filepath} [get]
func ServeDocument(c *gin.Context, cfg *config.CloudConfig) {
	relPath := c.Param("filepath")

	// Resolve symlinks and confirm the real target is still inside the repo
	// root, so a symlink inside WORK_DIR cannot expose files outside it.
	fullPath, err := util.SanitizeRepoPathResolved(cfg.Server.FileRoot, relPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Never let the browser sniff and execute user-controlled content. Text and
	// markup (including .html/.xml/.js) are served as inert text/plain so they
	// cannot run scripts in the application origin. PDFs keep their real type;
	// the frontend parses those client-side.
	contentType := "text/plain; charset=utf-8"
	if strings.EqualFold(filepath.Ext(fullPath), ".pdf") {
		contentType = "application/pdf"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline")
	c.Header("X-Content-Type-Options", "nosniff")
	// Never let the browser serve a stale copy: these files are read back into
	// the editor after being changed, so a heuristic disk cache shows old text.
	c.Header("Cache-Control", "no-store")

	c.File(fullPath)
}
