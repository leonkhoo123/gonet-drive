package controller

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go-file-server/internal/config"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"

	"github.com/gin-gonic/gin"
)

func PublicConfigRoutes(router *gin.RouterGroup) {
	configGroup := router.Group("/config")
	{
		configGroup.GET("/logo", getLogo)
	}
}

func getLogo(c *gin.Context) {
	logoPath := filepath.Join(config.AppConfig.Server.FileRoot, ".cloud_reserve", "config", "icon", "logo.png")

	c.Header("Cache-Control", "no-cache")
	c.File(logoPath)
}

func UpdateLogo(c *gin.Context) {
	file, err := c.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Logo file is required"})
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PNG files are allowed"})
		return
	}

	// Validate file size (e.g., max 5MB)
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds 5MB limit"})
		return
	}

	iconDir := filepath.Join(config.AppConfig.Server.FileRoot, ".cloud_reserve", "config", "icon")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory", "details": err.Error()})
		return
	}

	logoPath := filepath.Join(iconDir, "logo.png")

	if err := c.SaveUploadedFile(file, logoPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save logo file", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logo updated successfully"})
}

func ConfigRoutes(router *gin.RouterGroup, repo repository.CloudConfigRepository) {
	configGroup := router.Group("/config")
	{
		configGroup.GET("", listConfigs(repo))
		configGroup.PUT("/:id", updateConfig(repo))
	}
}

func listConfigs(repo repository.CloudConfigRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		configs, err := repo.ListAllNotDeleted()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query configs", "details": err.Error()})
			return
		}

		if configs == nil {
			configs = []model.CloudConfig{} // This will require import "go-file-server/internal/model" or we can map it
		}

		c.JSON(http.StatusOK, configs)
	}
}

type UpdateConfigRequest struct {
	ConfigValue *string `json:"config_value"`
	IsEnabled   *bool   `json:"is_enabled"`
	IsDeleted   *bool   `json:"is_deleted"`
}

func updateConfig(repo repository.CloudConfigRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
			return
		}

		var req UpdateConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
			return
		}

		if req.ConfigValue == nil && req.IsEnabled == nil && req.IsDeleted == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		rowsAffected, err := repo.Update(id, req.ConfigValue, req.IsEnabled, req.IsDeleted)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config", "details": err.Error()})
			return
		}

		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found or already deleted"})
			return
		}

		// Reload the cache after update
		if err := config.RefreshCloudConfigCache(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "config updated, but failed to reload cache", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "config updated successfully"})
	}
}
