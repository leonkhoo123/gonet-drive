package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/logger"
	"go-file-server/internal/state"
	"go-file-server/internal/storage"
	"go-file-server/internal/util"
	"go-file-server/internal/ws"
)

type CopyReq struct {
	Sources []string `json:"sources" binding:"required"`
	DestDir string   `json:"destDir" binding:"required"`
	OpID    string   `json:"opId"` // Optional: client provided operation ID
}

type MoveReq struct {
	Sources []string `json:"sources" binding:"required"`
	DestDir string   `json:"destDir" binding:"required"`
	OpID    string   `json:"opId"`
}

type DeleteReq struct {
	Sources []string `json:"sources" binding:"required"`
	OpID    string   `json:"opId"`
}

type RenameReq struct {
	Source  string `json:"source" binding:"required"`
	NewName string `json:"newName" binding:"required"`
	OpID    string `json:"opId"`
}

type CreateFolderReq struct {
	Dir        string `json:"dir"`
	FolderName string `json:"folderName" binding:"required"`
	OpID       string `json:"opId"`
}

type CancelReq struct {
	OpID   string `json:"opId" binding:"required"`
	Cancel *bool  `json:"cancel"` // optional explicit cancel flag
}

func getOpName(opType string, sources []string, destDir string) string {
	if len(sources) == 0 {
		return opType
	}
	firstFile := filepath.Base(sources[0])
	firstFile = util.TruncateString(firstFile, 20)

	actionDesc := "Processing"
	switch opType {
	case "copy":
		actionDesc = "Copying"
	case "move":
		actionDesc = "Moving"
	case "delete":
		actionDesc = "Deleting"
	case "delete_permanent":
		actionDesc = "Permanently Deleting"
	}

	destName := ""
	if destDir != "" {
		destName = util.TruncateString(filepath.Base(destDir), 20)
	}

	if len(sources) == 1 {
		if destDir != "" {
			return fmt.Sprintf("%s %s to %s", actionDesc, firstFile, destName)
		}
		return fmt.Sprintf("%s %s", actionDesc, firstFile)
	}

	if destDir != "" {
		return fmt.Sprintf("%s %s and %d more items to %s", actionDesc, firstFile, len(sources)-1, destName)
	}
	return fmt.Sprintf("%s %s and %d more items", actionDesc, firstFile, len(sources)-1)
}

func submitAsyncJob(opID, opType, opName string, tracker *util.ProgressTracker, includeSpeed bool, destDir string, requestID string, username string, action func(tracker *util.ProgressTracker) error) {
	var destDirPtr *string
	if destDir != "" {
		destDirPtr = &destDir
	}

	var requestIDPtr *string
	if requestID != "" {
		requestIDPtr = &requestID
	}

	ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
		OpId:      opID,
		OpType:    opType,
		OpName:    opName,
		OpStatus:  "queued",
		DestDir:   destDirPtr,
		RequestId: requestIDPtr,
	})

	logger.L.With("request_id", requestID).Info("job queued", "opId", opID, "opType", opType)

	JobQueue <- func() {
		jobLog := logger.L.With("request_id", requestID, "opId", opID)

		ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
			OpId:      opID,
			OpType:    opType,
			OpName:    opName,
			OpStatus:  "starting",
			DestDir:   destDirPtr,
			RequestId: requestIDPtr,
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

			ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
				OpId:         opID,
				OpType:       opType,
				OpName:       opName,
				OpStatus:     "in-progress",
				OpPercentage: &percentage,
				OpSpeed:      speedStr,
				OpFileCount:  &fileCountStr,
				DestDir:      destDirPtr,
				RequestId:    requestIDPtr,
			})
		}

		err := action(tracker)

		if err != nil {
			jobLog.Error("file operation failed", "opType", opType, "err", err)
			errMsg := err.Error()

			status := "error"
			if strings.Contains(errMsg, "operation canceled") {
				status = "aborted"
			}

			ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
				OpId:      opID,
				OpType:    opType,
				OpName:    opName,
				OpStatus:  status,
				Error:     &errMsg,
				DestDir:   destDirPtr,
				RequestId: requestIDPtr,
			})
		} else {
			jobLog.Debug("file operation completed", "opType", opType, "files", tracker.CopiedFiles, "bytes", tracker.CopiedBytes)
			ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
				OpId:      opID,
				OpType:    opType,
				OpName:    opName,
				OpStatus:  "completed",
				DestDir:   destDirPtr,
				RequestId: requestIDPtr,
			})
		}
	}
}

func CancelOperation(req CancelReq, username string) error {
	logger.L.Debug("received cancel request", "opId", req.OpID)
	// We assume sending to this endpoint means intent to cancel,
	// but we respect the cancel flag if it's explicitly false.
	if req.Cancel != nil && !*req.Cancel {
		return nil
	}

	// Verify ownership: only the user who started the operation can cancel it
	owner := state.GetOperationOwner(req.OpID)
	if owner != "" && owner != username {
		return fmt.Errorf("access denied: you did not initiate this operation")
	}

	util.SetCancelOperation(req.OpID)
	return nil
}

func CopyFiles(req CopyReq, cfg *config.CloudConfig, requestID string, username string) error {
	logger.L.Debug("copy files requested", "opId", req.OpID, "sources", req.Sources, "destDir", req.DestDir)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}
	state.SetOperationOwner(opID, username)

	opName := getOpName("copy", req.Sources, req.DestDir)

	sendError := func(err error) error {
		errMsg := err.Error()
		ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
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

	submitAsyncJob(opID, "copy", opName, tracker, true, req.DestDir, requestID, username, func(t *util.ProgressTracker) error {
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

func MoveFiles(req MoveReq, cfg *config.CloudConfig, requestID string, username string) error {
	logger.L.Debug("move files requested", "opId", req.OpID, "sources", req.Sources, "destDir", req.DestDir)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}
	state.SetOperationOwner(opID, username)

	opName := getOpName("move", req.Sources, req.DestDir)

	sendError := func(err error) error {
		errMsg := err.Error()
		ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
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

	submitAsyncJob(opID, "move", opName, tracker, false, req.DestDir, requestID, username, func(t *util.ProgressTracker) error {
		return util.MoveFiles(t, safeSources, safeDestDir, opID)
	})

	return nil
}

func DeleteFiles(req DeleteReq, cfg *config.CloudConfig, requestID string, username string) error {
	logger.L.Debug("delete files requested", "opId", req.OpID, "sources", req.Sources)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}
	state.SetOperationOwner(opID, username)

	opName := getOpName("delete", req.Sources, "")

	sendError := func(err error) error {
		errMsg := err.Error()
		ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
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
			logger.L.Warn("skipping protected folder", "path", source)
			continue
		}
		validSources = append(validSources, source)
	}

	if len(validSources) == 0 {
		return sendError(fmt.Errorf("no valid files to delete"))
	}

	tracker := util.NewProgressTracker()
	state.SetProgress(opID, tracker)

	submitAsyncJob(opID, "delete", opName, tracker, false, "", requestID, username, func(t *util.ProgressTracker) error {
		recycleBinDir := filepath.Join(cfg.Server.FileRoot, ".cloud_delete")
		if _, err := os.Stat(recycleBinDir); os.IsNotExist(err) {
			os.MkdirAll(recycleBinDir, 0755)
		}
		return util.MoveFiles(t, validSources, recycleBinDir, opID)
	})

	return nil
}

func DeletePermanentFiles(req DeleteReq, cfg *config.CloudConfig, requestID string, username string) error {
	logger.L.Debug("permanent delete requested", "opId", req.OpID, "sources", req.Sources)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}
	state.SetOperationOwner(opID, username)

	opName := getOpName("delete_permanent", req.Sources, "")

	sendError := func(err error) error {
		errMsg := err.Error()
		ws.BroadcastAndCacheToUser(username, ws.OperationMessage{
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
			logger.L.Warn("skipping protected folder", "path", source)
			continue
		}
		validSources = append(validSources, source)
	}

	if len(validSources) == 0 {
		return sendError(fmt.Errorf("no valid files to delete"))
	}

	tracker := util.NewProgressTracker()
	state.SetProgress(opID, tracker)

	submitAsyncJob(opID, "delete_permanent", opName, tracker, false, "", requestID, username, func(t *util.ProgressTracker) error {
		return util.DeleteFilesPermanent(t, validSources, opID, func(deletedSize int64) {
			storage.SubtractUsage(deletedSize)
		})
	})

	return nil
}

func RenameFile(req RenameReq, cfg *config.CloudConfig) error {
	logger.L.Debug("rename file requested", "opId", req.OpID, "source", req.Source, "newName", req.NewName)

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

func CreateFolder(req CreateFolderReq, cfg *config.CloudConfig) error {
	logger.L.Debug("create folder requested", "opId", req.OpID, "dir", req.Dir, "folderName", req.FolderName)

	safeDir, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.Dir)
	if err != nil {
		return err
	}

	safeFolderName, err := util.SanitizeFilename(req.FolderName)
	if err != nil {
		return err
	}

	newFolderPath := filepath.Clean(filepath.Join(safeDir, safeFolderName))
	if strings.Contains(newFolderPath, "..") {
		return fmt.Errorf("invalid path")
	}

	if _, err := os.Stat(newFolderPath); !os.IsNotExist(err) {
		return fmt.Errorf("folder already exists")
	}

	if err := os.MkdirAll(newFolderPath, 0755); err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}

	return nil
}
