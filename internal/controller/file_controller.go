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

	api.GET("/file-test", fileTestHandler)

	api.GET("/file-list", func(c *gin.Context) {
		service.FileList(c, cfg)
	})

	api.GET("/storage-usage", storageUsageHandler)

	api.POST("/copy", copyFilesHandler(cfg))
	api.POST("/move", moveFilesHandler(cfg))
	api.POST("/cancel", cancelOperationHandler)
	api.POST("/delete", deleteFilesHandler(cfg))
	api.POST("/delete-permanent", deletePermanentFilesHandler(cfg))
	api.POST("/rename", renameFileHandler(cfg))
	api.POST("/create-folder", createFolderHandler(cfg))
	api.POST("/properties", filePropertiesHandler(cfg))
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

	api.Static("/static", "./static")
}

// fileTestHandler godoc
// @Summary      File API Health Check
// @Description  Simple health check for the files API module.
// @Tags         Files
// @Produce      plain
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {string}  string  "OK"
// @Router       /api/user/files/file-test [get]
func fileTestHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

// storageUsageHandler godoc
// @Summary      Storage Usage
// @Description  Get current storage usage and limit.
// @Tags         Files
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {object}  map[string]interface{}
// @Router       /api/user/files/storage-usage [get]
func storageUsageHandler(c *gin.Context) {
	var limit int64 = 0
	if config.AppCloudConfig != nil {
		limit = config.AppCloudConfig.StorageLimit
	}
	c.JSON(http.StatusOK, gin.H{
		"used":  storage.GetUsage(),
		"limit": limit,
	})
}

// copyFilesHandler godoc
// @Summary      Copy Files
// @Description  Start an asynchronous copy operation for files/directories.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.CopyReq  true  "Copy request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/copy [post]
func copyFilesHandler(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CopyReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		requestID := c.GetString("request_id")
		if err := service.CopyFiles(req, cfg, requestID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Copy operation started"})
	}
}

// moveFilesHandler godoc
// @Summary      Move Files
// @Description  Start an asynchronous move operation for files/directories.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.MoveReq  true  "Move request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/move [post]
func moveFilesHandler(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.MoveReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		requestID := c.GetString("request_id")
		if err := service.MoveFiles(req, cfg, requestID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Move operation started"})
	}
}

// cancelOperationHandler godoc
// @Summary      Cancel Operation
// @Description  Cancel a running or queued file operation by its operation ID.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.CancelReq  true  "Cancel request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/cancel [post]
func cancelOperationHandler(c *gin.Context) {
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
}

// deleteFilesHandler godoc
// @Summary      Soft Delete Files
// @Description  Move files/directories to the recycle bin (.cloud_delete).
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.DeleteReq  true  "Delete request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/delete [post]
func deleteFilesHandler(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.DeleteReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		requestID := c.GetString("request_id")
		if err := service.DeleteFiles(req, cfg, requestID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Delete operation started"})
	}
}

// deletePermanentFilesHandler godoc
// @Summary      Permanent Delete
// @Description  Permanently delete files/directories (bypass recycle bin).
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.DeleteReq  true  "Delete request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/delete-permanent [post]
func deletePermanentFilesHandler(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.DeleteReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		requestID := c.GetString("request_id")
		if err := service.DeletePermanentFiles(req, cfg, requestID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Permanent delete operation started"})
	}
}

// renameFileHandler godoc
// @Summary      Rename File
// @Description  Rename a file or directory.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.RenameReq  true  "Rename request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/rename [post]
func renameFileHandler(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// createFolderHandler godoc
// @Summary      Create Folder
// @Description  Create a new directory.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.CreateFolderReq  true  "Create folder request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/create-folder [post]
func createFolderHandler(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// filePropertiesHandler godoc
// @Summary      File Properties
// @Description  Get metadata properties for files or directories.
// @Tags         Files
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.PropertiesReq  true  "Properties request"
// @Success      200   {object}  service.PropertiesResponse
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/files/properties [post]
func filePropertiesHandler(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
