package service

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/state"
	"go-file-server/internal/storage"
	"go-file-server/internal/util"
	"go-file-server/internal/ws"

	"github.com/gin-gonic/gin"
)

func submitShareAsyncJob(opID, opType, opName string, tracker *util.ProgressTracker, includeSpeed bool, destDir string, action func(tracker *util.ProgressTracker) error) {
	var destDirPtr *string
	if destDir != "" {
		destDirPtr = &destDir
	}

	ws.Broadcast(ws.OperationMessage{
		OpId:     opID,
		OpType:   opType,
		OpName:   opName,
		OpStatus: "queued",
		DestDir:  destDirPtr,
	})

	JobQueue <- func() {
		ws.Broadcast(ws.OperationMessage{
			OpId:     opID,
			OpType:   opType,
			OpName:   opName,
			OpStatus: "starting",
			DestDir:  destDirPtr,
		})

		tracker.OnProgress = func(pt *util.ProgressTracker) {
			percentage := 0.0
			if pt.TotalBytes > 0 && opType == "copy" {
				percentage = float64(pt.CopiedBytes) / float64(pt.TotalBytes) * 100
			} else if pt.TotalFiles > 0 {
				percentage = float64(pt.CopiedFiles) / float64(pt.TotalFiles) * 100
			}

			var speedStr *string
			if includeSpeed {
				elapsed := time.Since(pt.StartTime).Seconds()
				if elapsed > 0 && pt.CopiedBytes > 0 {
					speed := float64(pt.CopiedBytes) / elapsed
					s := util.FormatBytes(int64(speed)) + "/s"
					speedStr = &s
				}
			}

			fileCountStr := fmt.Sprintf("%d/%d", pt.CopiedFiles, pt.TotalFiles)

			ws.Broadcast(ws.OperationMessage{
				OpId:         opID,
				OpType:       opType,
				OpName:       opName,
				OpStatus:     "in-progress",
				OpPercentage: &percentage,
				OpSpeed:      speedStr,
				OpFileCount:  &fileCountStr,
				DestDir:      destDirPtr,
			})
		}

		err := action(tracker)

		if err != nil {
			log.Printf("Error in share %s operation %s: %v", opType, opID, err)
			errMsg := err.Error()

			status := "error"
			if strings.Contains(errMsg, "operation canceled") {
				status = "aborted"
			}

			ws.Broadcast(ws.OperationMessage{
				OpId:     opID,
				OpType:   opType,
				OpName:   opName,
				OpStatus: status,
				Error:    &errMsg,
				DestDir:  destDirPtr,
			})
		} else {
			ws.Broadcast(ws.OperationMessage{
				OpId:     opID,
				OpType:   opType,
				OpName:   opName,
				OpStatus: "completed",
				DestDir:  destDirPtr,
			})
		}
	}
}

func ShareCopyFiles(req CopyReq, cfg *config.CloudConfig) error {
	log.Printf("[OpID: %s] ShareCopyFiles: sources=%v, destDir=%s", req.OpID, req.Sources, req.DestDir)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}

	opName := getOpName("copy", req.Sources, req.DestDir)

	sendError := func(err error) error {
		errMsg := err.Error()
		ws.Broadcast(ws.OperationMessage{
			OpId:     opID,
			OpType:   "copy",
			OpName:   opName,
			OpStatus: "error",
			Error:    &errMsg,
		})
		return err
	}

	safeSources, err := util.SanitizeRepoPaths(cfg.Server.FileRoot, req.Sources)
	if err != nil {
		return sendError(err)
	}

	safeDestDir, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.DestDir)
	if err != nil {
		return sendError(err)
	}

	isSameDir := false
	cleanDestDir := filepath.Clean(safeDestDir)
	for _, source := range safeSources {
		if filepath.Dir(source) == cleanDestDir {
			isSameDir = true
			break
		}
	}

	tracker := util.NewProgressTracker()
	state.SetProgress(opID, tracker)

	submitShareAsyncJob(opID, "copy", opName, tracker, true, req.DestDir, func(t *util.ProgressTracker) error {
		var reservedSize int64
		err := util.CopyFiles(t, safeSources, safeDestDir, opID, isSameDir, func(totalSize int64) error {
			if config.AppCloudConfig != nil {
				if err := storage.HasSufficientStorage(config.AppCloudConfig.StorageLimit, totalSize); err != nil {
					return fmt.Errorf("storage limit exceeded")
				}
			}
			storage.AddUsage(totalSize)
			reservedSize = totalSize
			return nil
		})

		if err != nil {
			storage.SubtractUsage(reservedSize)
		}

		return err
	})

	return nil
}

func ShareMoveFiles(req MoveReq, cfg *config.CloudConfig) error {
	log.Printf("[OpID: %s] ShareMoveFiles: sources=%v, destDir=%s", req.OpID, req.Sources, req.DestDir)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}

	opName := getOpName("move", req.Sources, req.DestDir)

	sendError := func(err error) error {
		errMsg := err.Error()
		ws.Broadcast(ws.OperationMessage{
			OpId:     opID,
			OpType:   "move",
			OpName:   opName,
			OpStatus: "error",
			Error:    &errMsg,
		})
		return err
	}

	safeSources, err := util.SanitizeRepoPaths(cfg.Server.FileRoot, req.Sources)
	if err != nil {
		return sendError(err)
	}

	safeDestDir, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.DestDir)
	if err != nil {
		return sendError(err)
	}

	cleanDestDir := filepath.Clean(safeDestDir)
	for _, source := range safeSources {
		if filepath.Dir(source) == cleanDestDir {
			return sendError(fmt.Errorf("cannot move items to the same directory"))
		}
	}

	tracker := util.NewProgressTracker()
	state.SetProgress(opID, tracker)

	submitShareAsyncJob(opID, "move", opName, tracker, false, req.DestDir, func(t *util.ProgressTracker) error {
		return util.MoveFiles(t, safeSources, safeDestDir, opID)
	})

	return nil
}

func ShareDeleteFiles(req DeleteReq, cfg *config.CloudConfig) error {
	log.Printf("[OpID: %s] ShareDeleteFiles: sources=%v", req.OpID, req.Sources)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}

	opName := getOpName("delete", req.Sources, "")

	sendError := func(err error) error {
		errMsg := err.Error()
		ws.Broadcast(ws.OperationMessage{
			OpId:     opID,
			OpType:   "delete",
			OpName:   opName,
			OpStatus: "error",
			Error:    &errMsg,
		})
		return err
	}

	safeSources, err := util.SanitizeRepoPaths(cfg.Server.FileRoot, req.Sources)
	if err != nil {
		return sendError(err)
	}

	var validSources []string
	for _, source := range safeSources {
		baseName := filepath.Base(source)
		if baseName == ".cloud_reserve" || baseName == ".cloud_delete" {
			log.Printf("Skipping deletion of protected folder in share context: %s", source)
			continue
		}
		validSources = append(validSources, source)
	}

	if len(validSources) == 0 {
		return sendError(fmt.Errorf("no valid files to delete"))
	}

	tracker := util.NewProgressTracker()
	state.SetProgress(opID, tracker)

	submitShareAsyncJob(opID, "delete", opName, tracker, false, "", func(t *util.ProgressTracker) error {
		recycleBinDir := filepath.Join(cfg.Server.FileRoot, ".cloud_delete")
		if _, err := os.Stat(recycleBinDir); os.IsNotExist(err) {
			os.MkdirAll(recycleBinDir, 0755)
		}
		return util.MoveFiles(t, validSources, recycleBinDir, opID)
	})

	return nil
}

func ShareDeletePermanentFiles(req DeleteReq, cfg *config.CloudConfig) error {
	log.Printf("[OpID: %s] ShareDeletePermanentFiles: sources=%v", req.OpID, req.Sources)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}

	opName := getOpName("delete_permanent", req.Sources, "")

	sendError := func(err error) error {
		errMsg := err.Error()
		ws.Broadcast(ws.OperationMessage{
			OpId:     opID,
			OpType:   "delete_permanent",
			OpName:   opName,
			OpStatus: "error",
			Error:    &errMsg,
		})
		return err
	}

	safeSources, err := util.SanitizeRepoPaths(cfg.Server.FileRoot, req.Sources)
	if err != nil {
		return sendError(err)
	}

	var validSources []string
	for _, source := range safeSources {
		baseName := filepath.Base(source)
		if baseName == ".cloud_reserve" || baseName == ".cloud_delete" {
			log.Printf("Skipping permanent deletion of protected folder in share context: %s", source)
			continue
		}
		validSources = append(validSources, source)
	}

	if len(validSources) == 0 {
		return sendError(fmt.Errorf("no valid files to delete"))
	}

	tracker := util.NewProgressTracker()
	state.SetProgress(opID, tracker)

	submitShareAsyncJob(opID, "delete_permanent", opName, tracker, false, "", func(t *util.ProgressTracker) error {
		return util.DeleteFilesPermanent(t, validSources, opID, func(deletedSize int64) {
			storage.SubtractUsage(deletedSize)
		})
	})

	return nil
}

func ShareRenameFile(req RenameReq, cfg *config.CloudConfig) error {
	log.Printf("[OpID: %s] ShareRenameFile: source=%s, newName=%s", req.OpID, req.Source, req.NewName)

	safeSource, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.Source)
	if err != nil {
		return err
	}

	if filepath.Base(safeSource) == ".cloud_delete" || filepath.Base(safeSource) == ".cloud_reserve" {
		return fmt.Errorf("cannot rename protected system folders")
	}

	safeNewName, err := util.SanitizeFilename(req.NewName)
	if err != nil {
		return err
	}

	if err := util.RenameFile(safeSource, safeNewName); err != nil {
		return err
	}

	return nil
}

// codeql[go/path-injection] False positive: inputs are sanitized by SanitizeRepoPath and SanitizeFilename
func ShareCreateFolder(req CreateFolderReq, cfg *config.CloudConfig) error {
	log.Printf("[OpID: %s] ShareCreateFolder: dir=%s, folderName=%s", req.OpID, req.Dir, req.FolderName)

	safeDir, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.Dir)
	if err != nil {
		return err
	}

	safeFolderName, err := util.SanitizeFilename(req.FolderName)
	if err != nil {
		return err
	}

	newFolderPath := filepath.Join(safeDir, safeFolderName)

	if _, err := os.Stat(newFolderPath); !os.IsNotExist(err) {
		return fmt.Errorf("folder already exists")
	}

	if err := os.MkdirAll(newFolderPath, 0755); err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}

	return nil
}

func ShareFileList(c *gin.Context, cfg *config.CloudConfig) {
	relPath := c.GetString("override_path")
	if relPath == "" {
		relPath = c.Query("path")
	}
	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, relPath)
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	cleanPath, _ := filepath.Rel(cfg.Server.FileRoot, fullPath)
	if cleanPath == "." {
		cleanPath = "/"
	} else {
		cleanPath = "/" + filepath.ToSlash(cleanPath)
	}

	shareRoot := c.GetString("share_root")

	// Calculate displayPath relative to shareRoot
	displayPath := cleanPath
	if shareRoot != "" {
		sRoot := filepath.ToSlash(filepath.Clean("/" + shareRoot))
		if sRoot == "/" {
			displayPath = cleanPath
		} else {
			if cleanPath == sRoot {
				displayPath = "/"
			} else if strings.HasPrefix(cleanPath, sRoot+"/") {
				displayPath = strings.TrimPrefix(cleanPath, sRoot)
				if displayPath == "" {
					displayPath = "/"
				}
			} else {
				displayPath = "/"
			}
		}
	}

	sortBy := c.DefaultQuery("sort", "name")
	order := c.DefaultQuery("order", "asc")
	showHiddenStr := c.DefaultQuery("showHidden", "false")
	showHidden := showHiddenStr == "true"

	entries, err := os.ReadDir(fullPath)
	isSingleFile := false
	if err != nil {
		info, statErr := os.Stat(fullPath)
		if statErr == nil && !info.IsDir() {
			entries = []os.DirEntry{fs.FileInfoToDirEntry(info)}
			isSingleFile = true
		} else {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	}

	var result []gin.H
	var fileCount, folderCount int

	for _, e := range entries {
		// Do not show system files in shares
		if e.Name() == ".cloud_reserve" || e.Name() == ".cloud_delete" {
			continue
		}
		if !showHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}

		if e.IsDir() {
			folderCount++
		} else {
			fileCount++
		}

		info, _ := e.Info()
		url := ""
		mediaType := ""

		itemPath := ""
		if isSingleFile {
			itemPath = displayPath
		} else {
			if displayPath == "/" {
				itemPath = "/" + e.Name()
			} else {
				itemPath = displayPath + "/" + e.Name()
			}
		}

		baseURL := cfg.Server.Hostname
		if baseURL == "" {
			baseURL = util.GetBaseURL(c.Request)
		}
		prefix := "/api/share/file"
		shareID := c.GetString("share_id")

		if isVideoFile(info.Name()) {
			url = fmt.Sprintf("%s%s/video/play/file%s?share_id=%s", baseURL, prefix, itemPath, shareID)
			mediaType = "video"
		} else if isPhotoFile(info.Name()) {
			url = fmt.Sprintf("%s%s/photo/play/file%s?share_id=%s", baseURL, prefix, itemPath, shareID)
			mediaType = "photo"
		} else if isMusicFile(info.Name()) {
			url = fmt.Sprintf("%s%s/music/play/file%s?share_id=%s", baseURL, prefix, itemPath, shareID)
			mediaType = "music"
		} else if isPdfFile(info.Name()) {
			url = fmt.Sprintf("%s%s/document/read/file%s?share_id=%s", baseURL, prefix, itemPath, shareID)
			mediaType = "pdf"
		} else if isTextFile(info.Name()) {
			url = fmt.Sprintf("%s%s/document/read/file%s?share_id=%s", baseURL, prefix, itemPath, shareID)
			mediaType = "text_documents"
		}

		result = append(result, gin.H{
			"name":       e.Name(),
			"type":       map[bool]string{true: "dir", false: "file"}[e.IsDir()],
			"size":       info.Size(),
			"modified":   info.ModTime(),
			"url":        url,
			"media_type": mediaType,
			"path":       itemPath,
		})
	}

	sortMethod := ""
	ordering := "asc"
	// --- Sorting ---
	sort.Slice(result, func(i, j int) bool {
		isDirI := result[i]["type"].(string) == "dir"
		isDirJ := result[j]["type"].(string) == "dir"

		if isDirI != isDirJ {
			return isDirI
		}

		var less bool
		switch sortBy {
		case "size":
			if !isDirI && !isDirJ {
				less = result[i]["size"].(int64) < result[j]["size"].(int64)
			} else {
				less = strings.ToLower(result[i]["name"].(string)) < strings.ToLower(result[j]["name"].(string))
			}
			sortMethod = "size"
		case "modified":
			less = result[i]["modified"].(time.Time).Before(result[j]["modified"].(time.Time))
			sortMethod = "modified"
		default: // name
			less = strings.ToLower(result[i]["name"].(string)) < strings.ToLower(result[j]["name"].(string))
			sortMethod = "name"
		}

		if order == "desc" {
			ordering = "desc"
			return !less
		}
		return less
	})

	var limit int64 = 0
	if config.AppCloudConfig != nil {
		limit = config.AppCloudConfig.StorageLimit
	}

	response := gin.H{
		"path":           displayPath,
		"sort":           sortMethod,
		"order":          ordering,
		"items":          result,
		"file_count":     fileCount,
		"folder_count":   folderCount,
		"count":          len(result),
		"is_single_file": isSingleFile,
		"storage": gin.H{
			"used":  storage.GetUsage(),
			"limit": limit,
		},
	}

	if shareRoot != "" {
		shareRootClean := filepath.ToSlash(filepath.Clean("/" + shareRoot))
		if shareRootClean == "/" {
			response["share_root"] = "/"
		} else {
			response["share_root"] = "/" + filepath.Base(shareRootClean)
		}
	}

	c.JSON(http.StatusOK, response)
}

func ShareDownloadFiles(c *gin.Context, cfg *config.CloudConfig) {
	sources := c.GetStringSlice("override_sources")
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
	serverName := "Share_Download"
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
	defer zipWriter.Close()

	for _, fullPath := range validPaths {
		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(filepath.Dir(fullPath), path)
			if err != nil {
				return err
			}
			relPath = filepath.ToSlash(relPath)
			if info.IsDir() {
				relPath += "/"
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = relPath
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
			fmt.Printf("Error zipping %s: %v\n", fullPath, err)
			return
		}
	}
}

// ShareUploadChunk handles chunked file uploads from the frontend for shared files.
// codeql[go/path-injection] False positive: identifier is sanitized with IsSafePathComponent, and destination/filename with SanitizeRepoPath/SanitizeFilename
func ShareUploadChunk(c *gin.Context, cfg *config.CloudConfig) {
	if config.AppCloudConfig != nil {
		maxAllowed := config.AppCloudConfig.UploadChunkSize + 1024*1024
		if c.Request.ContentLength > maxAllowed {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
			return
		}
	}

	identifier := c.PostForm("identifier")
	status := c.PostForm("status")

	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing identifier"})
		return
	}

	if !util.IsSafePathComponent(identifier) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid identifier"})
		return
	}

	tempDir := filepath.Join(cfg.Server.FileRoot, ".cloud_reserve", "upload_temp", identifier)

	if status == "cancel" {
		size := storage.GetPathSize(tempDir)
		_ = os.RemoveAll(tempDir)
		storage.SubtractUsage(size)
		c.JSON(http.StatusOK, gin.H{"message": "Upload cancelled", "status": "cancelled"})
		return
	}

	filename := c.PostForm("filename")
	destination := c.GetString("override_destination")
	if destination == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing destination"})
		return
	}

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

	destPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, destination)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path: " + err.Error()})
		return
	}

	if err := os.MkdirAll(destPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create destination directory"})
		return
	}

	cleanFilename, err := util.SanitizeFilename(filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}

	fileHeader, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing chunk file"})
		return
	}

	if config.AppCloudConfig != nil {
		if err := storage.HasSufficientStorage(config.AppCloudConfig.StorageLimit, fileHeader.Size); err != nil {
			c.JSON(http.StatusInsufficientStorage, gin.H{"error": "Storage limit exceeded"})
			return
		}
	}

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

	if status == "start" && chunkNumber == 1 {
		size := storage.GetPathSize(tempDir)
		_ = os.RemoveAll(tempDir)
		storage.SubtractUsage(size)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}

	chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkNumber))
	tmpChunkPath := chunkPath + ".tmp"

	tmpOutFile, err := os.Create(tmpChunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file for chunk"})
		return
	}

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

	// Verify checksum here using the shared verifyChecksumFile which we will also duplicate or use if exported.
	// verifyChecksumFile is unexported in file_upload.go, but we are in the same package!
	if checksum != "" {
		if !verifyChecksumFile(tmpChunkPath, checksum) {
			os.Remove(tmpChunkPath)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Checksum mismatch"})
			return
		}
	}

	if err := os.Rename(tmpChunkPath, chunkPath); err != nil {
		os.Remove(tmpChunkPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rename chunk file"})
		return
	}

	storage.AddUsage(fileHeader.Size)

	if status == "end" || chunkNumber == totalChunks {
		for i := 1; i <= totalChunks; i++ {
			if _, err := os.Stat(filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))); os.IsNotExist(err) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing chunk %d", i)})
				return
			}
		}

		finalDest := filepath.Join(destPath, cleanFilename)
		finalDest = util.GetUniqueDestPath(finalDest)

		outFile, err := os.Create(finalDest)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create final file"})
			return
		}

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
