package service

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"go-file-server/internal/config"

	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

// ServeVideo serves video files with HTTP range support
func ServeVideo(c *gin.Context, cfg *config.CloudConfig) {
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

	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		var start, end int64
		n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		if n == 1 {
			end = stat.Size() - 1
		}
		if end >= stat.Size() {
			end = stat.Size() - 1
		}

		chunkSize := (end - start) + 1

		c.Status(http.StatusPartialContent)
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, stat.Size()))
		c.Header("Accept-Ranges", "bytes")
		c.Header("Content-Length", fmt.Sprintf("%d", chunkSize))
		c.Header("Content-Type", util.MimeType(fullPath))

		// Stream directly from file descriptor (no full buffering)
		file.Seek(start, io.SeekStart)
		io.CopyN(c.Writer, file, chunkSize)
		return
	}

	// no Range header → stream entire file
	c.Header("Accept-Ranges", "bytes")
	c.File(fullPath)
}
