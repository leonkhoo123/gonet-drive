package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterVideoIntegrityAdminRoutes(adminRouter *gin.RouterGroup, cfg *config.CloudConfig) {
	viGroup := adminRouter.Group("/video-integrity")
	{
		viGroup.POST("/scan", startVideoIntegrityScan(cfg))
		viGroup.GET("/status", getVideoIntegrityStatus)
		viGroup.GET("/list", listVideoIntegrityEntries(cfg))
	}
}

type scanRequestBody struct {
	Action string `json:"action"`
}

func startVideoIntegrityScan(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body scanRequestBody
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			raw, _ := io.ReadAll(c.Request.Body)
			_ = json.Unmarshal(raw, &body)
		}

		if body.Action == "stop" {
			if !service.RequestScanStop() {
				httpx.Err(c, http.StatusNotFound, "no scan is currently running")
				return
			}
			logger.L.Info("video integrity scan stop requested via API")
			httpx.OK(c, http.StatusOK, gin.H{
				"opId":   "integrity-scan",
				"status": "stopping",
			})
			return
		}

		if service.IsVideoIntegrityScanRunning() {
			httpx.Err(c, http.StatusConflict, "a video integrity scan is already running")
			return
		}

		logger.L.Info("video integrity scan triggered via API")

		go func() {
			if _, err := service.ScanVideoIntegrity(cfg.Server.FileRoot); err != nil {
				logger.L.Error("video integrity scan failed", "err", err)
			}
		}()

		httpx.OK(c, http.StatusAccepted, gin.H{
			"opId":   "integrity-scan",
			"status": "started",
		})
	}
}

func getVideoIntegrityStatus(c *gin.Context) {
	count, lastScan, running, err := service.GetScanStatus()
	if err != nil {
		logger.L.Error("failed to get video integrity scan status", "err", err)
		httpx.Err(c, http.StatusInternalServerError, "failed to get scan status")
		return
	}

	resp := gin.H{
		"corrupt_count": count,
		"scan_running":  running,
	}
	if lastScan != nil {
		resp["last_scan"] = lastScan
	}
	httpx.OK(c, http.StatusOK, resp)
}

func listVideoIntegrityEntries(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		repo := service.GetVideoIntegrityRepo()
		if repo == nil {
			httpx.Err(c, http.StatusInternalServerError, "video integrity repository not initialized")
			return
		}

		entries, err := repo.All()
		if err != nil {
			logger.L.Error("failed to query video integrity entries", "err", err)
			httpx.Err(c, http.StatusInternalServerError, "failed to query video integrity entries")
			return
		}

		count, err := repo.Count()
		if err != nil {
			logger.L.Error("failed to count video integrity entries", "err", err)
			httpx.Err(c, http.StatusInternalServerError, "failed to query video integrity entries")
			return
		}

		// Enrich entries with relative_path for thumbnail generation
		// Strip full server paths from response for security
		type enrichedEntry struct {
			Hash            string `json:"hash"`
			RelativePath    string `json:"relative_path"`
			IssueType       string `json:"issue_type"`
			MimeCodecString string `json:"mime_codec_string"`
			DetectedAt      string `json:"detected_at"`
			LastCheckedAt   string `json:"last_checked_at"`
		}
		root := cfg.Server.FileRoot
		enriched := make([]enrichedEntry, len(entries))
		for i, e := range entries {
			rel := strings.TrimPrefix(e.FilePath, root)
			rel = strings.TrimPrefix(rel, "/")
			rel = strings.TrimPrefix(rel, "\\")
			enriched[i] = enrichedEntry{
				Hash:            e.Hash,
				RelativePath:    rel,
				IssueType:       e.IssueType,
				MimeCodecString: e.MimeCodecString,
				DetectedAt:      e.DetectedAt.Format(time.RFC3339),
				LastCheckedAt:   e.LastCheckedAt.Format(time.RFC3339),
			}
		}

		httpx.OK(c, http.StatusOK, gin.H{
			"total":   count,
			"entries": enriched,
		})
	}
}

func GetVideoIntegrityCorruptHashes(hashes []string) (map[string]bool, error) {
	repo := service.GetVideoIntegrityRepo()
	if repo == nil {
		return map[string]bool{}, nil
	}
	return repo.GetCorruptHashes(hashes)
}
