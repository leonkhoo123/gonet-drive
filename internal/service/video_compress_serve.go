package service

import (
	"fmt"
	"io"
	"net/http"

	"go-file-server/internal/config"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

func ServeCompressVid(c *gin.Context, cfg *config.CloudConfig) {
	// Example:
	// client requests: /play/compress/videos/sample.mp4?start=120
	relPath := c.Param("filepath") // e.g. "/videos/sample.mp4"
	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, relPath)
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	start := c.Query("start") // optional seek time
	if start == "" {
		start = "0"
	}

	// Construct transcoder API URL (your new Go + FFmpeg server)
	targetURL := fmt.Sprintf("http://localhost:8080/video/stream?filepath=%s&start=%s",
		fullPath, start,
	)

	// Prepare request to transcoder server
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to build request: %v", err)
		return
	}

	// Forward headers if needed
	req.Header.Set("Accept", "application/x-mpegURL")

	client := &http.Client{
		Timeout: 0, // stream: no timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusBadGateway, "Transcoder server error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Copy transcoder response headers
	for k, v := range resp.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	c.Status(resp.StatusCode)

	// Stream response directly to browser
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		fmt.Printf("Stream error: %v\n", err)
	}
}
