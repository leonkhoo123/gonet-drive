package controller

import (
	"go-file-server/internal/config"
	"go-file-server/internal/middleware"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/ws"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ShareFileRoutes handles accessing share files
func ShareFileRoutes(router *gin.Engine, shareRepo repository.SharingRepository) {
	// Share File Access (requires shareJwt token)
	fileAccess := router.Group("/api/share/file")
	fileAccess.Use(middleware.ShareAuthMiddleware(config.AppConfig, shareRepo))
	{
		fileAccess.GET("/ws", ws.WsHandler)
		fileAccess.GET("/list", ShareFileList)
		fileAccess.GET("/download", ShareFileDownload)
		fileAccess.POST("/properties", ShareFileProperties)

		fileAccess.GET("/music/play/file/*filepath", ShareFileServe)
		fileAccess.GET("/video/play/file/*filepath", ShareFileServe)
		fileAccess.GET("/video/thumbnail/file/*filepath", ShareFileServe)
		fileAccess.GET("/photo/play/file/*filepath", ShareFileServe)
		fileAccess.GET("/photo/thumbnail/file/*filepath", ShareFileServe)
		fileAccess.GET("/document/read/file/*filepath", ShareFileServe)

		// Modify Authority required
		modify := fileAccess.Group("")
		modify.Use(middleware.ShareModifyAuthorityMiddleware())
		{
			modify.POST("/upload-chunk", ShareFileUpload)
			modify.POST("/delete", ShareFileDeleteSoft)
			modify.POST("/delete-permanent", ShareFileDelete)
			modify.POST("/create-folder", ShareCreateFolder)
			modify.POST("/rename", ShareFileRename)
			modify.POST("/copy", ShareFileCopy)
			modify.POST("/move", ShareFileMove)
		}
	}
}

// ensureWithinAuthorizedPath checks if the requested subpath stays within the root of the authorized path
// It returns the fully resolved path if safe, or an error/empty string if not.
func ensureWithinAuthorizedPath(authorizedPath string, requestedSubPath string) (string, bool) {
	authorizedPath = filepath.Clean(authorizedPath)
	requestedSubPath = filepath.Clean(requestedSubPath)

	var targetPath string
	if requestedSubPath == "" || requestedSubPath == "." || requestedSubPath == "/" {
		targetPath = authorizedPath
	} else {
		// Join them and clean again
		targetPath = filepath.Clean(filepath.Join(authorizedPath, requestedSubPath))
	}

	if authorizedPath == "." {
		// If authorizedPath is root (.), any path that doesn't go up (..) is valid
		if strings.HasPrefix(targetPath, "..") {
			return "", false
		}
		return targetPath, true
	}

	// Ensure targetPath still starts with the authorizedPath with a trailing slash to prevent partial matches
	prefix := authorizedPath
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	if targetPath != authorizedPath && !strings.HasPrefix(targetPath, prefix) {
		return "", false
	}

	return targetPath, true
}

// ShareFileList lists files within a shared path.
// @Summary      List Share Files
// @Description  List files and directories within the authorized share path.
// @Tags         Share Files
// @Produce      json
// @Security     ShareAuth
// @Param        path       query     string  false  "Directory path"
// @Param        sort       query     string  false  "Sort by: name, size, modified"
// @Param        order      query     string  false  "Sort order: asc, desc"
// @Param        showHidden query     string  false  "Show hidden files: true, false"
// @Param        X-Share-Id header    string  false  "Share ID"
// @Param        share_id   query     string  false  "Share ID (alternative)"
// @Success      200        {object}  map[string]interface{}
// @Failure      403        {object}  map[string]interface{}
// @Router       /api/share/file/list [get]
func ShareFileList(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")
	reqPath := c.Query("path")

	targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), reqPath)
	if !safe {
		c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
		return
	}

	c.Set("override_path", targetPath)
	c.Set("share_root", authorizedPath)

	// We call the dedicated share file list service
	service.ShareFileList(c, config.AppConfig)
}

// ShareFileDownload downloads files from a shared path.
// @Summary      Download Share Files
// @Description  Download one or more files/directories from a shared path. Single files are served directly; multiple items are packaged as ZIP.
// @Tags         Share Files
// @Produce      octet-stream
// @Produce      application/zip
// @Security     ShareAuth
// @Param        source     query     []string  true  "Source file paths"
// @Param        X-Share-Id header    string     false  "Share ID"
// @Param        share_id   query     string     false  "Share ID (alternative)"
// @Success      200        {file}    binary
// @Failure      403        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Router       /api/share/file/download [get]
func ShareFileDownload(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")
	sources := c.QueryArray("source")

	var safeSources []string
	for _, src := range sources {
		targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), src)
		if !safe {
			c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
			return
		}
		safeSources = append(safeSources, targetPath)
	}

	c.Set("override_sources", safeSources)

	service.ShareDownloadFiles(c, config.AppConfig)
}

// ShareFileProperties gets file/directory properties in a shared path.
// @Summary      Share File Properties
// @Description  Get metadata for files/directories in a shared path.
// @Tags         Share Files
// @Accept       json
// @Produce      json
// @Security     ShareAuth
// @Param        X-Share-Id header                   string  false  "Share ID"
// @Param        share_id   query                    string  false  "Share ID (alternative)"
// @Param        body       body      service.PropertiesReq  true  "Properties request"
// @Success      200        {object}  service.PropertiesResponse
// @Failure      403        {object}  map[string]interface{}
// @Router       /api/share/file/properties [post]
func ShareFileProperties(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")

	// Set context values typically expected by FilePropertiesEndpoint
	var req service.PropertiesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate all requested source paths
	var safeSources []string
	for _, p := range req.Sources {
		targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), p)
		if !safe {
			c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
			return
		}
		safeSources = append(safeSources, targetPath)
	}
	req.Sources = safeSources

	res, err := service.GetFileProperties(req, config.AppConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format the location to be relative to the share root to avoid leaking internal paths
	if res.Location != "Multiple Locations" {
		authPath := filepath.ToSlash(filepath.Clean("/" + authorizedPath.(string)))
		loc := filepath.ToSlash(filepath.Clean("/" + res.Location))

		if loc == authPath {
			res.Location = "/"
		} else if strings.HasPrefix(loc, authPath+"/") {
			res.Location = strings.TrimPrefix(loc, authPath)
			if res.Location == "" {
				res.Location = "/"
			}
		} else {
			res.Location = "/"
		}
	}

	c.JSON(http.StatusOK, res)
}

// ShareFileUpload uploads a file chunk to a shared path (requires modify authority).
// @Summary      Upload to Share
// @Description  Upload a file chunk to a shared path. Requires modify authority.
// @Tags         Share Files
// @Accept       multipart/form-data
// @Produce      json
// @Security     ShareAuth
// @Param        identifier   formData  string  true   "Upload identifier"
// @Param        status       formData  string  true   "Chunk status: start, uploading, end, cancel"
// @Param        filename     formData  string  true   "Target filename"
// @Param        destination  formData  string  true   "Destination directory"
// @Param        chunkNumber  formData  int     true   "Chunk number (1-indexed)"
// @Param        totalChunks  formData  int     true   "Total number of chunks"
// @Param        checksum     formData  string  false  "Chunk checksum"
// @Param        chunk        formData  file    true   "The chunk file data"
// @Param        X-Share-Id   header    string  false  "Share ID"
// @Param        share_id     query     string  false  "Share ID (alternative)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Router       /api/share/file/upload-chunk [post]
func ShareFileUpload(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")

	// Apply MaxBytesReader early since we need to parse the form to read the destination
	if config.AppCloudConfig != nil {
		maxAllowed := config.AppCloudConfig.UploadChunkSize + 1024*1024
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAllowed)
	}

	reqPath := c.PostForm("destination")

	targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), reqPath)
	if !safe {
		c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
		return
	}

	// Set the sanitized path back in the context/request so the underlying service uses it
	c.Set("override_destination", targetPath)

	service.ShareUploadChunk(c, config.AppConfig)
}

// ShareFileDelete permanently deletes files (requires modify authority).
// @Summary      Permanent Delete Share Files
// @Tags         Share Files
// @Accept       json
// @Produce      json
// @Security     ShareAuth
// @Param        body  body      service.DeleteReq  true  "Delete request"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/share/file/delete-permanent [post]
func ShareFileDelete(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")

	var req service.DeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var safePaths []string
	for _, p := range req.Sources {
		targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), p)
		if !safe {
			c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path: " + p})
			return
		}
		safePaths = append(safePaths, targetPath)
	}
	req.Sources = safePaths

	// Perform actual delete using existing service logic
	// We'll use DeletePermanentFiles because normal DeleteFiles moves to a .cloud_delete recycle bin
	// which is likely not what we want for share guests, or we can use regular DeleteFiles
	err := service.ShareDeletePermanentFiles(req, config.AppConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to permanently delete files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Files permanently deleted successfully"})
}

// ShareCreateFolder creates a folder in a shared path (requires modify authority).
// @Summary      Create Folder in Share
// @Tags         Share Files
// @Accept       json
// @Produce      json
// @Security     ShareAuth
// @Param        body  body      service.CreateFolderReq  true  "Create folder request"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/share/file/create-folder [post]
func ShareCreateFolder(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")

	var req service.CreateFolderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), req.Dir)
	if !safe {
		c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
		return
	}

	req.Dir = targetPath

	err := service.ShareCreateFolder(req, config.AppConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder created successfully"})
}

// ShareFileServe serves media files from a shared path (video, photo, music, document).
// @Summary      Serve Share Media
// @Description  Serve media files (video, photo, music, document) from a shared path.
// @Tags         Share Files
// @Produce      octet-stream
// @Security     ShareAuth
// @Param        filepath  path  string  true  "Relative file path"
// @Success      200  {file}  binary
// @Failure      403  {object}  map[string]interface{}
// @Router       /api/share/file/{mediaType}/play/file/{filepath} [get]
func ShareFileServe(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")
	relPath := c.Param("filepath")

	targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), relPath)
	if !safe {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
		return
	}

	// Overwrite the parameter so the underlying service uses targetPath as the relative path
	for i, p := range c.Params {
		if p.Key == "filepath" {
			c.Params[i].Value = targetPath
			break
		}
	}

	// Check which service to call based on the URL
	if strings.HasPrefix(c.Request.URL.Path, "/api/share/file/music/") {
		service.ServeMusic(c, config.AppConfig)
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/share/file/video/thumbnail/") {
		service.ServeVideoThumbnail(c, config.AppConfig)
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/share/file/video/") {
		service.ServeVideo(c, config.AppConfig)
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/share/file/photo/thumbnail/") {
		service.ServePhotoThumbnail(c, config.AppConfig)
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/share/file/photo/") {
		service.ServePhoto(c, config.AppConfig)
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/share/file/document/") {
		service.ServeDocument(c, config.AppConfig)
	} else {
		c.AbortWithStatus(http.StatusNotFound)
	}
}

// ShareFileDelete soft-deletes files in a shared path (requires modify authority).
// @Summary      Soft Delete Share Files
// @Tags         Share Files
// @Accept       json
// @Produce      json
// @Security     ShareAuth
// @Param        body  body      service.DeleteReq  true  "Delete request"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/share/file/delete [post]
func ShareFileDeleteSoft(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")

	var req service.DeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var safePaths []string
	for _, p := range req.Sources {
		targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), p)
		if !safe {
			c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path: " + p})
			return
		}
		safePaths = append(safePaths, targetPath)
	}
	req.Sources = safePaths

	err := service.ShareDeleteFiles(req, config.AppConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Files deleted successfully"})
}

// ShareFileRename renames a file in a shared path (requires modify authority).
// @Summary      Rename File in Share
// @Tags         Share Files
// @Accept       json
// @Produce      json
// @Security     ShareAuth
// @Param        body  body      service.RenameReq  true  "Rename request"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/share/file/rename [post]
func ShareFileRename(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")

	var req service.RenameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), req.Source)
	if !safe {
		c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
		return
	}
	req.Source = targetPath

	if err := service.ShareRenameFile(req, config.AppConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File renamed successfully"})
}

// ShareFileCopy copies files within a shared path (requires modify authority).
// @Summary      Copy Files in Share
// @Tags         Share Files
// @Accept       json
// @Produce      json
// @Security     ShareAuth
// @Param        body  body      service.CopyReq  true  "Copy request"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/share/file/copy [post]
func ShareFileCopy(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")

	var req service.CopyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var safeSources []string
	for _, p := range req.Sources {
		targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), p)
		if !safe {
			c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
			return
		}
		safeSources = append(safeSources, targetPath)
	}
	req.Sources = safeSources

	destPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), req.DestDir)
	if !safe {
		c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
		return
	}
	req.DestDir = destPath

	if err := service.ShareCopyFiles(req, config.AppConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Files copied successfully"})
}

// ShareFileMove moves files within a shared path (requires modify authority).
// @Summary      Move Files in Share
// @Tags         Share Files
// @Accept       json
// @Produce      json
// @Security     ShareAuth
// @Param        body  body      service.MoveReq  true  "Move request"
// @Success      200   {object}  map[string]interface{}
// @Router       /api/share/file/move [post]
func ShareFileMove(c *gin.Context) {
	authorizedPath, _ := c.Get("authorized_path")

	var req service.MoveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var safeSources []string
	for _, p := range req.Sources {
		targetPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), p)
		if !safe {
			c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
			return
		}
		safeSources = append(safeSources, targetPath)
	}
	req.Sources = safeSources

	destPath, safe := ensureWithinAuthorizedPath(authorizedPath.(string), req.DestDir)
	if !safe {
		c.JSON(http.StatusForbidden, gin.H{"error": "Path traversal detected or unauthorized path"})
		return
	}
	req.DestDir = destPath

	if err := service.ShareMoveFiles(req, config.AppConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Files moved successfully"})
}
