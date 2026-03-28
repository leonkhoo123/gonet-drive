package controller

import (
	"go-file-server/internal/config"
	"go-file-server/internal/service"

	"github.com/gin-gonic/gin"
)

func AudioBookRoutes(router *gin.RouterGroup, cfg *config.CloudConfig, srv *service.AudiobookService) {
	abGroup := router.Group("/audiobook")
	{
		abGroup.GET("/paths", srv.GetAudioBookPaths)
		abGroup.POST("/path", srv.AddAudioBookPath)
		abGroup.PUT("/path/:id", srv.UpdateAudioBookPath)
		abGroup.DELETE("/path/:id", srv.DeleteAudioBookPath)

		abGroup.POST("/progress", srv.ReportAudioBookProgress)
		abGroup.GET("/list", srv.ListAudioBooks(cfg))
		abGroup.GET("/filelist", srv.AudioBookFileList(cfg))
		abGroup.GET("/stream/:name", srv.StreamAudioBook(cfg))
	}
}
