package controller

import (
	"go-file-server/internal/config"
	"go-file-server/internal/service"

	"github.com/gin-gonic/gin"
)

func PhotoRoutes(router *gin.RouterGroup, cfg *config.CloudConfig) {
	api := router.Group("/photo")

	// serve actual photo
	api.GET("/play/file/*filepath", func(c *gin.Context) {
		service.ServePhoto(c, cfg)
	})
}
