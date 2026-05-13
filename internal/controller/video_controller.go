package controller

import (
	"go-file-server/internal/config"
	"go-file-server/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func VideoRoutes(router *gin.RouterGroup, cfg *config.CloudConfig) {
	api := router.Group("/video")

	api.GET("/play/file/*filepath", func(c *gin.Context) {
		service.ServeVideo(c, cfg)
	})

	api.GET("/thumbnail/file/*filepath", func(c *gin.Context) {
		service.ServeVideoThumbnail(c, cfg)
	})

	api.POST("/disqualified", func(c *gin.Context) {
		service.VideoDisqualified(c, cfg)
	})

	api.POST("/rename-done", func(c *gin.Context) {
		service.VideoRenameDone(c, cfg)
	})

	api.GET("/video-test", videoTestHandler)

	api.Static("/static", "./static")
}

// videoTestHandler godoc
// @Summary      Video API Health Check
// @Description  Simple health check for the video API module.
// @Tags         Media
// @Produce      plain
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {string}  string  "OK"
// @Router       /api/user/video/video-test [get]
func videoTestHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}
