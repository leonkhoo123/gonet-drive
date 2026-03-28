package controller

import (
	"go-file-server/internal/config"
	"go-file-server/internal/service"

	"github.com/gin-gonic/gin"
)

func DocumentRoutes(router *gin.RouterGroup, cfg *config.CloudConfig) {
	api := router.Group("/document")

	// serve document text content
	api.GET("/read/file/*filepath", func(c *gin.Context) {
		service.ServeDocument(c, cfg)
	})
}
