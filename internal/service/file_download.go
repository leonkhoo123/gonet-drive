package service

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-file-server/internal/config"

	"github.com/gin-gonic/gin"
)

func DownloadFiles(c *gin.Context, cfg *config.CloudConfig) {
	sources := c.QueryArray("source")
	if len(sources) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no sources provided"})
		return
	}

	baseDir := cfg.Server.FileRoot

	// Validate all paths first
	var validPaths []string
	var totalValidFiles int
	for _, src := range sources {
		fullPath := filepath.Join(baseDir, filepath.Clean(src))
		// Ensure the path is within baseDir
		if !strings.HasPrefix(fullPath, filepath.Clean(baseDir)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
			return
		}

		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip non-existent files
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error accessing file"})
			return
		}

		validPaths = append(validPaths, fullPath)
		if info.IsDir() {
			totalValidFiles += 2 // Just an indicator that it's more than a single file
		} else {
			totalValidFiles++
		}
	}

	if len(validPaths) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no valid files found"})
		return
	}

	// Case 1: Single file, download directly
	if len(validPaths) == 1 {
		info, err := os.Stat(validPaths[0])
		if err == nil && !info.IsDir() {
			c.Writer.Header().Set("X-Accel-Buffering", "no")
			c.Writer.Header().Set("Content-Type", "application/octet-stream")
			c.FileAttachment(validPaths[0], filepath.Base(validPaths[0]))
			return
		}
	}

	// Case 2: Multiple files or a single directory, stream as a zip archive
	serverName := "My_Server"
	if config.AppCloudConfig != nil && config.AppCloudConfig.TitleName != "" {
		serverName = config.AppCloudConfig.TitleName
	}
	serverName = strings.ReplaceAll(serverName, " ", "_")
	var cleanServerName strings.Builder
	for _, r := range serverName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			cleanServerName.WriteRune(r)
		}
	}
	if cleanServerName.Len() == 0 {
		cleanServerName.WriteString("Server")
	}

	now := time.Now()
	timestamp := now.Unix()
	dateStr := now.Format("2006_01_02")
	zipFileName := fmt.Sprintf("%s_%d_%s_download.zip", dateStr, timestamp, cleanServerName.String())

	c.Writer.Header().Set("Content-Type", "application/zip")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipFileName))
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Write directly to the HTTP response writer
	zipWriter := zip.NewWriter(c.Writer)
	// We want no compression to save CPU power
	// To do this in Go zip, we need to register a custom compressor or use CreateHeader

	defer zipWriter.Close()

	for _, fullPath := range validPaths {
		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Calculate the relative path to store in the zip
			// For example, if downloading /base/folder, inside zip it should be folder/file
			relPath, err := filepath.Rel(filepath.Dir(fullPath), path)
			if err != nil {
				return err
			}

			// Ensure forward slashes in zip
			relPath = filepath.ToSlash(relPath)

			if info.IsDir() {
				// Directories in zip need a trailing slash
				relPath += "/"
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			header.Name = relPath
			// Set method to Store (no compression)
			header.Method = zip.Store

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return err
			}

			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				_, err = io.Copy(writer, file)
				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			// If we fail midway, we might have already written some headers,
			// but we can at least log it. We can't change the HTTP status code anymore.
			fmt.Printf("Error zipping %s: %v\n", fullPath, err)
			return
		}
	}
}
