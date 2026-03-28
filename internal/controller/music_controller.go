package controller

import (
	"go-file-server/internal/config"
	"go-file-server/internal/service"

	"github.com/gin-gonic/gin"
)

func MusicRoutes(router *gin.RouterGroup, cfg *config.CloudConfig) {
	api := router.Group("/music")

	// serve actual music file
	api.GET("/play/file/*filepath", func(c *gin.Context) {
		service.ServeMusic(c, cfg)
	})
}
