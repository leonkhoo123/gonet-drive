package service

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go-file-server/internal/config"
	"go-file-server/internal/storage"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

// UploadChunk handles chunked file uploads from the frontend.
// codeql[go/path-injection] False positive: identifier is sanitized with IsSafePathComponent, and destination/filename with SanitizeRepoPath/SanitizeFilename
func UploadChunk(c *gin.Context, cfg *config.CloudConfig) {
	// Quick check on Content-Length to prevent massive abuse before parsing
	if config.AppCloudConfig != nil {
		// allow some overhead for multipart form data, e.g., 1MB
		maxAllowed := config.AppCloudConfig.UploadChunkSize + 1024*1024
		if c.Request.ContentLength > maxAllowed {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
			return
		}
		// Wrap the request body with http.MaxBytesReader to enforce the limit during parsing
		// This prevents malicious clients from omitting Content-Length and sending huge bodies,
		// which would otherwise cause disk exhaustion during c.FormFile parsing or OOM.
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAllowed)
	}

	identifier := c.PostForm("identifier")
	status := c.PostForm("status") // "start", "uploading", "end", "cancel"

	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing identifier"})
		return
	}

	if !util.IsSafePathComponent(identifier) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid identifier"})
		return
	}

	// Create temp directory for this upload inside .cloud_reserve/upload_temp
	tempDir := filepath.Join(cfg.Server.FileRoot, ".cloud_reserve", "upload_temp", identifier)

	// If the frontend is aborting the upload
	if status == "cancel" {
		size := storage.GetPathSize(tempDir)
		_ = os.RemoveAll(tempDir)
		storage.SubtractUsage(size)
		c.JSON(http.StatusOK, gin.H{"message": "Upload cancelled", "status": "cancelled"})
		return
	}

	filename := c.PostForm("filename")
	destination := c.PostForm("destination")
	chunkNumberStr := c.PostForm("chunkNumber")
	totalChunksStr := c.PostForm("totalChunks")
	checksum := c.PostForm("checksum")

	if filename == "" || chunkNumberStr == "" || totalChunksStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	chunkNumber, err := strconv.Atoi(chunkNumberStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chunkNumber"})
		return
	}

	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid totalChunks"})
		return
	}

	// Validate destination path against FileRoot to prevent path traversal
	destPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, destination)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path: " + err.Error()})
		return
	}

	// Make sure the destination directory exists
	if err := os.MkdirAll(destPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create destination directory"})
		return
	}

	// Validate filename
	cleanFilename, err := util.SanitizeFilename(filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}

	// Handle the uploaded chunk file
	fileHeader, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing chunk file"})
		return
	}

	// Check if this chunk will exceed the global storage limit
	if config.AppCloudConfig != nil {
		if err := storage.HasSufficientStorage(config.AppCloudConfig.StorageLimit, fileHeader.Size); err != nil {
			c.JSON(http.StatusInsufficientStorage, gin.H{"error": "Storage limit exceeded"})
			return
		}
	}

	// Double check the exact part size
	if config.AppCloudConfig != nil {
		if fileHeader.Size > config.AppCloudConfig.UploadChunkSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Chunk size exceeds the allowed limit"})
			return
		}
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open chunk file"})
		return
	}
	defer file.Close()

	// If this is the start of a new upload for this file, clear existing temp directory
	if status == "start" && chunkNumber == 1 {
		size := storage.GetPathSize(tempDir)
		_ = os.RemoveAll(tempDir)
		storage.SubtractUsage(size)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}

	// Save chunk safely by streaming to a .tmp file first, then renaming
	chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkNumber))
	tmpChunkPath := chunkPath + ".tmp"

	tmpOutFile, err := os.Create(tmpChunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file for chunk"})
		return
	}

	// Stream file to disk to avoid loading whole chunk in memory
	if _, err := io.Copy(tmpOutFile, file); err != nil {
		tmpOutFile.Close()
		os.Remove(tmpChunkPath)
		if strings.Contains(err.Error(), "no space left on device") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cloud storage quota exceeded or disk is full"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save chunk data"})
		return
	}
	tmpOutFile.Close()

	// Verify checksum if provided by frontend
	if checksum != "" {
		if !verifyChecksumFile(tmpChunkPath, checksum) {
			os.Remove(tmpChunkPath)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Checksum mismatch"})
			return
		}
	}

	if err := os.Rename(tmpChunkPath, chunkPath); err != nil {
		// Clean up the temp file if rename fails
		os.Remove(tmpChunkPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rename chunk file"})
		return
	}

	// Record chunk usage
	storage.AddUsage(fileHeader.Size)

	// If this is the last chunk, combine them
	if status == "end" || chunkNumber == totalChunks {
		// Verify all chunks exist
		for i := 1; i <= totalChunks; i++ {
			if _, err := os.Stat(filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))); os.IsNotExist(err) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing chunk %d", i)})
				return
			}
		}

		// Prepare the final path
		finalDest := filepath.Join(destPath, cleanFilename)
		finalDest = util.GetUniqueDestPath(finalDest)

		outFile, err := os.Create(finalDest)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create final file"})
			return
		}

		// Append all chunks to the final file
		var finalSize int64
		for i := 1; i <= totalChunks; i++ {
			cp := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
			chunkFile, err := os.Open(cp)
			if err != nil {
				outFile.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open chunk during merge"})
				return
			}
			n, err := io.Copy(outFile, chunkFile)
			if err != nil {
				chunkFile.Close()
				outFile.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write final file"})
				return
			}
			finalSize += n
			chunkFile.Close()
		}
		outFile.Close()

		storage.AddUsage(finalSize)

		// Cleanup temp dir
		tempSize := storage.GetPathSize(tempDir)
		_ = os.RemoveAll(tempDir)
		storage.SubtractUsage(tempSize)

		c.JSON(http.StatusOK, gin.H{
			"message": "Upload complete",
			"status":  "done",
			"path":    filepath.ToSlash(filepath.Clean(destination + "/" + cleanFilename)),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chunk uploaded", "status": "uploading"})
}

// verifyChecksumFile attempts to match the provided checksum using common algorithms by streaming the file
func verifyChecksumFile(filePath string, expected string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	// Try CRC32 (common for lightweight rolling checksums)
	if len(expected) == 8 {
		h := crc32.NewIEEE()
		if _, err := io.Copy(h, f); err == nil {
			return fmt.Sprintf("%08x", h.Sum(nil)) == expected
		}
		return false
	}
	// Try MD5
	if len(expected) == 32 {
		h := md5.New()
		if _, err := io.Copy(h, f); err == nil {
			return hex.EncodeToString(h.Sum(nil)) == expected
		}
		return false
	}
	// Try SHA256
	if len(expected) == 64 {
		h := sha256.New()
		if _, err := io.Copy(h, f); err == nil {
			return hex.EncodeToString(h.Sum(nil)) == expected
		}
		return false
	}

	// Fallback to testing all if length doesn't directly imply algorithm
	hCrc := crc32.NewIEEE()
	hMd5 := md5.New()
	hSha256 := sha256.New()
	w := io.MultiWriter(hCrc, hMd5, hSha256)
	if _, err := io.Copy(w, f); err != nil {
		return false
	}

	if fmt.Sprintf("%08x", hCrc.Sum(nil)) == expected {
		return true
	}
	if hex.EncodeToString(hMd5.Sum(nil)) == expected {
		return true
	}
	if hex.EncodeToString(hSha256.Sum(nil)) == expected {
		return true
	}

	return false
}
