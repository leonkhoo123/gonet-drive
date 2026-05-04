package controller

import (
	"go-file-server/internal/config"
	"go-file-server/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func VideoRoutes(router *gin.RouterGroup, cfg *config.CloudConfig) {
	api := router.Group("/video")

	// serve actual video
	api.GET("/play/file/*filepath", func(c *gin.Context) {
		service.ServeVideo(c, cfg)
	})

	// serve video thumbnail
	api.GET("/thumbnail/file/*filepath", func(c *gin.Context) {
		service.ServeVideoThumbnail(c, cfg)
	})

	api.POST("/disqualified", func(c *gin.Context) {
		service.VideoDisqualified(c, cfg)
	})

	api.POST("/rename-done", func(c *gin.Context) {
		service.VideoRenameDone(c, cfg)
	})

	// api.GET("/play/compress/health", func(c *gin.Context) {

	// 	// Prepare request to transcoder server
	// 	req, err := http.NewRequest("GET", "http://localhost:8080/health/", nil)
	// 	if err != nil {
	// 		c.String(http.StatusInternalServerError, "Failed to build request: %v", err)
	// 		return
	// 	}
	// 	client := &http.Client{
	// 		Timeout: 0, // stream: no timeout
	// 	}
	// 	resp, err := client.Do(req)
	// 	c.Status(resp.StatusCode)

	// })

	// api.GET("/play/compress/file/*filepath", func(c *gin.Context) {
	// 	service.ServeCompressVid(c, cfg)
	// })

	// video api health check
	api.GET("/video-test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	api.Static("/static", "./static") // optional for thumbnails, etc.
}
