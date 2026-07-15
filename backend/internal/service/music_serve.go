package service

import (
	"net/http"
	"os"

	"go-file-server/internal/config"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

// ServeMusic serves music/audio files.
// @Summary      Serve Music
// @Description  Stream a music/audio file.
// @Tags         Media
// @Produce      audio/*
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true  "Relative file path"
// @Success      200  {file}  binary
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/music/play/file/{filepath} [get]
func ServeMusic(c *gin.Context, cfg *config.CloudConfig) {
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

	c.File(fullPath)
}
