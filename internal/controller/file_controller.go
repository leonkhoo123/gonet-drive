package controller

import (
	"net/http"

	"go-file-server/internal/config"
	"go-file-server/internal/service"
	"go-file-server/internal/storage"

	"github.com/gin-gonic/gin"
)

func FilesRoutes(router *gin.RouterGroup, cfg *config.CloudConfig) {
	api := router.Group("/files")
	// file api health check
	api.GET("/file-test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	api.GET("/file-list", func(c *gin.Context) {
		service.FileList(c, cfg)
	})

	api.GET("/storage-usage", func(c *gin.Context) {
		var limit int64 = 0
		if config.AppCloudConfig != nil {
			limit = config.AppCloudConfig.StorageLimit
		}
		c.JSON(http.StatusOK, gin.H{
			"used":  storage.GetUsage(),
			"limit": limit,
		})
	})

	api.POST("/delete-rotate-temp", func(c *gin.Context) {
		service.DeleteTempRotate(c, cfg)
	})

	api.POST("/copy", func(c *gin.Context) {
		var req service.CopyReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := service.CopyFiles(req, cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Copy operation started"})
	})

	api.POST("/move", func(c *gin.Context) {
		var req service.MoveReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := service.MoveFiles(req, cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Move operation started"})
	})

	api.POST("/cancel", func(c *gin.Context) {
		var req service.CancelReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := service.CancelOperation(req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Operation cancelled"})
	})

	api.POST("/delete", func(c *gin.Context) {
		var req service.DeleteReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := service.DeleteFiles(req, cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Delete operation started"})
	})

	api.POST("/delete-permanent", func(c *gin.Context) {
		var req service.DeleteReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := service.DeletePermanentFiles(req, cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Permanent delete operation started"})
	})

	api.POST("/rename", func(c *gin.Context) {
		var req service.RenameReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := service.RenameFile(req, cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "File renamed successfully"})
	})

	api.POST("/create-folder", func(c *gin.Context) {
		var req service.CreateFolderReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := service.CreateFolder(req, cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Folder created successfully"})
	})

	api.POST("/properties", func(c *gin.Context) {
		var req service.PropertiesReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		res, err := service.GetFileProperties(req, cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, res)
	})

	api.POST("/check-duplicates", func(c *gin.Context) {
		service.CheckDuplicates(c, cfg)
	})

	api.POST("/check-upload-duplicates", func(c *gin.Context) {
		service.CheckUploadDuplicates(c, cfg)
	})

	api.GET("/download", func(c *gin.Context) {
		service.DownloadFiles(c, cfg)
	})

	api.POST("/upload-chunk", func(c *gin.Context) {
		service.UploadChunk(c, cfg)
	})

	api.Static("/static", "./static") // optional for thumbnails, etc.
}
