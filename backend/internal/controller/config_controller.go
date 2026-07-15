package controller

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"

	"github.com/gin-gonic/gin"
)

func PublicConfigRoutes(router *gin.RouterGroup) {
	configGroup := router.Group("/config")
	{
		configGroup.GET("/logo", getLogo)
		configGroup.GET("/manifest", getManifest)
	}
}

// getManifest returns the PWA manifest JSON.
// @Summary      Get PWA Manifest
// @Description  Return the Progressive Web App manifest JSON.
// @Tags         Config
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/config/manifest [get]
func getManifest(c *gin.Context) {
	cloudConfig := config.AppCloudConfig
	serviceName := "GoNet Drive"
	if cloudConfig != nil && cloudConfig.ServiceName != "" {
		serviceName = cloudConfig.ServiceName
	}

	c.Header("Content-Type", "application/manifest+json")
	httpx.OK(c, http.StatusOK, gin.H{
		"name":             serviceName,
		"short_name":       serviceName,
		"description":      serviceName,
		"start_url":        "/",
		"display":          "standalone",
		"theme_color":      "#ffffff",
		"background_color": "#ffffff",
		"icons": []gin.H{
			{"src": "/api/config/logo", "sizes": "192x192", "type": "image/png", "purpose": "any"},
			{"src": "/api/config/logo", "sizes": "512x512", "type": "image/png", "purpose": "any"},
		},
	})
}

// getLogo returns the service logo image.
// @Summary      Get Logo
// @Description  Return the service logo image file.
// @Tags         Config
// @Produce      image/png
// @Success      200  {file}  binary
// @Router       /api/config/logo [get]
func getLogo(c *gin.Context) {
	config.EnsureDefaultLogo()

	c.Header("Cache-Control", "no-cache")
	c.File(config.GetLogoPath())
}

// UpdateLogo uploads a new logo image (admin only).
// @Summary      Update Logo
// @Description  Upload a new PNG logo file. Requires admin role.
// @Tags         Admin
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        logo  formData  file  true  "PNG logo file"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /api/user/admin/config/logo [put]
func UpdateLogo(c *gin.Context) {
	file, err := c.FormFile("logo")
	if err != nil {
		httpx.Err(c, http.StatusBadRequest, "Logo file is required")
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".png" {
		httpx.Err(c, http.StatusBadRequest, "Only PNG files are allowed")
		return
	}

	// Validate file size (e.g., max 5MB)
	if file.Size > 5*1024*1024 {
		httpx.Err(c, http.StatusBadRequest, "File size exceeds 5MB limit")
		return
	}

	iconDir := filepath.Dir(config.GetLogoPath())
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		logger.L.Error("failed to create logo directory", "err", err, "path", iconDir)
		httpx.Err(c, http.StatusInternalServerError, "failed to save logo")
		return
	}

	if err := c.SaveUploadedFile(file, config.GetLogoPath()); err != nil {
		logger.L.Error("failed to save logo file", "err", err)
		httpx.Err(c, http.StatusInternalServerError, "failed to save logo")
		return
	}

	httpx.Msg(c, http.StatusOK, "Logo updated successfully")
}

func ConfigRoutes(router *gin.RouterGroup, repo repository.CloudConfigRepository) {
	configGroup := router.Group("/config")
	{
		configGroup.GET("", listConfigs(repo))
		configGroup.PUT("/:id", updateConfig(repo))
	}
}

// listConfigs lists all non-deleted cloud configs.
// @Summary      List Configs
// @Description  Get all cloud configuration entries.
// @Tags         Config
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {array}   model.CloudConfig
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/user/config [get]
func listConfigs(repo repository.CloudConfigRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		configs, err := repo.ListAllNotDeleted()
		if err != nil {
			logger.L.Error("failed to query configs", "err", err)
			httpx.Err(c, http.StatusInternalServerError, "failed to query configs")
			return
		}

		if configs == nil {
			configs = []model.CloudConfig{} // This will require import "go-file-server/internal/model" or we can map it
		}

		httpx.OK(c, http.StatusOK, gin.H{"configs": configs})
	}
}

type UpdateConfigRequest struct {
	ConfigValue *string `json:"config_value"`
	IsEnabled   *bool   `json:"is_enabled"`
	IsDeleted   *bool   `json:"is_deleted"`
}

// updateConfig updates a cloud config entry by ID.
// @Summary      Update Config
// @Description  Update a cloud configuration entry (value, enabled status, or soft-delete).
// @Tags         Config
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        id    path      int                  true  "Config ID"
// @Param        body  body      UpdateConfigRequest  true  "Update request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Router       /api/user/config/{id} [put]
func updateConfig(repo repository.CloudConfigRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			httpx.Err(c, http.StatusBadRequest, "invalid config id")
			return
		}

		var req UpdateConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Err(c, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.ConfigValue == nil && req.IsEnabled == nil && req.IsDeleted == nil {
			httpx.Err(c, http.StatusBadRequest, "no fields to update")
			return
		}

		rowsAffected, err := repo.Update(id, req.ConfigValue, req.IsEnabled, req.IsDeleted)
		if err != nil {
			logger.L.Error("failed to update config", "err", err, "id", id)
			httpx.Err(c, http.StatusInternalServerError, "failed to update config")
			return
		}

		if rowsAffected == 0 {
			httpx.Err(c, http.StatusNotFound, "config not found or already deleted")
			return
		}

		// Reload the cache after update
		if err := config.RefreshCloudConfigCache(); err != nil {
			logger.L.Error("failed to reload config cache after update", "err", err, "id", id)
			httpx.Err(c, http.StatusInternalServerError, "config updated, but failed to reload")
			return
		}

		httpx.Msg(c, http.StatusOK, "config updated successfully")
	}
}
