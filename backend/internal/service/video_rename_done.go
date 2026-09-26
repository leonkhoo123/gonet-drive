package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/state"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"
)

// errVideoProcessingFailed is the generic message shown in the operation queue.
// The detailed cause (e.g. ffmpeg stderr) stays in the server log.
var errVideoProcessingFailed = errors.New("Video processing failed")

// VideoRenameDoneReq is the payload for the rename-and-save action.
type VideoRenameDoneReq struct {
	Path        string `json:"path"`
	NewName     string `json:"newName"`
	RotateAngle int    `json:"rotateAngle"`
	OpID        string `json:"opId"`
}

// VideoDoneOpType identifies the async "rename to done + embed metadata" job.
const VideoDoneOpType = "video-done"

// VideoProcessingDirName is the staging folder inside the hidden
// `.cloud_reserve` directory that the source video is atomically moved into
// before processing. Keeping it out of the browse tree means the staging area
// never shows up under `done/` and never has to be deleted — deleting it would
// race with other in-flight rename-done jobs sharing the folder. On failure the
// original is left here for recovery.
const VideoProcessingDirName = "temp"

// VideoRenameDone moves a video into a staging folder and queues an async job
// that rotates/embeds metadata into it before it lands in the `done` folder.
// This mirrors copy/move: it returns immediately with an operation id and
// streams progress over the WebSocket.
// @Summary      Rename Video (Done)
// @Description  Stage a video in .cloud_reserve/temp, then rotate + embed metadata into the done folder asynchronously.
// @Tags         Media
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.VideoRenameDoneReq  true  "Request with path, newName, rotateAngle, opId"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Router       /api/user/video/rename-done [post]
func VideoRenameDone(c *gin.Context, cfg *config.CloudConfig) {
	var req VideoRenameDoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Err(c, http.StatusBadRequest, "invalid request")
		return
	}

	username := c.GetString(authgin.KeyUsername)
	requestID := c.GetString("request_id")

	opID, err := StartVideoRenameDone(req, cfg, requestID, username)
	if err != nil {
		logger.L.Error("video rename-done failed to start", "err", err, "path", req.Path)
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	httpx.OK(c, http.StatusOK, gin.H{
		"message": "Video processing started",
		"opId":    opID,
	})
}

// StartVideoRenameDone validates the request, atomically moves the video out of
// its browse folder into .cloud_reserve/temp, and enqueues the async processing
// job. It returns the operation id the client can use to track progress.
func StartVideoRenameDone(req VideoRenameDoneReq, cfg *config.CloudConfig, requestID, username string) (string, error) {
	srcPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.Path)
	if err != nil {
		return "", fmt.Errorf("access forbidden: %w", err)
	}

	newName, err := util.SanitizeFilename(req.NewName)
	if err != nil {
		return "", fmt.Errorf("invalid filename: %w", err)
	}

	parentDir := filepath.Dir(srcPath)
	doneDir := filepath.Join(parentDir, "done")
	// Stage the file in the hidden reserve directory rather than in done/tmp:
	// the staging area stays out of the browse tree, never needs deleting, and
	// cannot be disturbed by other concurrent rename-done jobs.
	procDir := filepath.Join(cfg.Server.FileRoot, util.CloudReserveDirName, VideoProcessingDirName)
	if err := os.MkdirAll(procDir, 0777); err != nil {
		return "", fmt.Errorf("failed to create processing folder: %w", err)
	}

	procPath, err := util.ResolveDuplicatePath(procDir, filepath.Base(srcPath))
	if err != nil {
		return "", fmt.Errorf("invalid processing path: %w", err)
	}

	// Move the source out of the browse folder FIRST (atomic rename, same
	// filesystem) so it disappears immediately and cannot be reopened while the
	// potentially slow remux runs. On failure it stays in .cloud_reserve/temp
	// for recovery.
	if err := os.Rename(srcPath, procPath); err != nil {
		return "", fmt.Errorf("failed to move file to processing folder: %w", err)
	}

	sidecarPath := util.SidecarPath(srcPath)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}
	state.SetOperationOwner(opID, username)

	opName := "Processing " + util.TruncateString(filepath.Base(req.Path), 20)
	virtualParent := parentVirtualPath(req.Path)

	tracker := util.NewProgressTracker()
	state.SetProgress(opID, tracker)

	rotateAngle := req.RotateAngle
	submitAsyncJob(opID, VideoDoneOpType, opName, tracker, true, virtualParent, requestID, username, func(t *util.ProgressTracker) error {
		// The HTTP request context is already gone by the time the worker runs,
		// so the remux owns its own cancellable context.
		if err := processVideoRenameDone(procPath, doneDir, sidecarPath, newName, rotateAngle, t); err != nil {
			// Keep the detailed cause (ffmpeg stderr, IO errors) in the log and
			// only surface a short, generic message in the operation queue.
			logger.L.Error("video processing failed", "opId", opID, "path", procPath, "newName", newName, "err", err)
			return errVideoProcessingFailed
		}
		return nil
	})

	logger.L.Debug("video rename-done queued", "opId", opID, "path", req.Path, "newName", newName, "angle", rotateAngle)
	return opID, nil
}

// processVideoRenameDone performs the actual work on the file-operation worker.
// The source already lives in .cloud_reserve/temp/<name>; the annotated result is
// written to done/<newName>. On failure the original is left in
// .cloud_reserve/temp.
func processVideoRenameDone(procPath, doneDir, sidecarPath, newName string, rotateAngle int, tracker *util.ProgressTracker) error {
	procInfo, err := os.Stat(procPath)
	if err != nil {
		return fmt.Errorf("processing file not found: %w", err)
	}
	if procInfo.IsDir() {
		return fmt.Errorf("processing path is not a file")
	}

	// The staging folder no longer lives under done/, so make sure the
	// destination folder exists before we move/remux into it.
	if err := os.MkdirAll(doneDir, 0777); err != nil {
		return fmt.Errorf("failed to create done folder: %w", err)
	}

	destPath, err := util.ResolveDuplicatePath(doneDir, newName)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	_, sidecarStatErr := os.Stat(sidecarPath)
	sidecarExists := sidecarStatErr == nil
	payload, hasMetadata := buildEmbedPayload(sidecarPath, newName)
	willEmbed := hasMetadata && util.IsMP4FamilyExt(filepath.Ext(procPath))

	tracker.TotalBytes = procInfo.Size()
	tracker.TotalFiles = 1

	// A genuine processing failure leaves the original in .cloud_reserve/temp and
	// keeps the sidecar next to it so the pair can be recovered together.
	failProcessing := func(processingErr error) error {
		if sidecarExists {
			if relocateErr := util.RelocateSidecar(sidecarPath, procPath); relocateErr != nil {
				logger.L.Warn("failed to relocate sidecar next to processing file", "path", sidecarPath, "err", relocateErr)
			}
		}
		return processingErr
	}

	if rotateAngle != 0 || willEmbed {
		metadata := map[string]string{}
		if willEmbed {
			metadata["description"] = payload
			metadata["comment"] = payload
		}
		if err := util.RemuxVideoWithMetadata(context.Background(), procPath, destPath, rotateAngle, metadata, tracker.Update); err != nil {
			return failProcessing(err)
		}
		// Remux succeeded: drop the staging file, the annotated copy is in done/.
		if err := os.Remove(procPath); err != nil {
			logger.L.Warn("failed to remove staging file after remux", "path", procPath, "err", err)
		}
	} else {
		// Nothing to process (no rotation and nothing to embed): just move it.
		if err := os.Rename(procPath, destPath); err != nil {
			return failProcessing(fmt.Errorf("failed to move file to done: %w", err))
		}
	}

	tracker.FileCompleted()

	// The container tag is the source of truth on success. When embedding did
	// not happen (non-MP4 container), move the sidecar into done/.vid_metadata/
	// instead: its presence there is the "not embedded" signal.
	if willEmbed {
		if err := os.Remove(sidecarPath); err != nil && !os.IsNotExist(err) {
			logger.L.Warn("failed to remove metadata sidecar after embed", "path", sidecarPath, "err", err)
		}
	} else if sidecarExists {
		if err := util.RelocateSidecar(sidecarPath, destPath); err != nil {
			logger.L.Warn("failed to relocate sidecar to done folder", "path", sidecarPath, "err", err)
		}
	}

	logger.L.Info("video moved to done", "from", filepath.Base(procPath), "to", filepath.Base(destPath), "embedded", willEmbed, "rotated", rotateAngle != 0)
	return nil
}

// buildEmbedPayload reads the sidecar and renders a portable, superset payload:
// the original `events` pairs (what the player parses) plus a `scenes` array
// (what desktop/doc consumers expect) and the final file name.
func buildEmbedPayload(sidecarPath, newName string) (string, bool) {
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		return "", false
	}

	events, ok := parseEventsJSON(data)
	if !ok {
		return "", false
	}

	scenes := make([]map[string]float64, 0, len(events))
	for _, e := range events {
		scenes = append(scenes, map[string]float64{"start": e[0], "end": e[1]})
	}

	payload, err := json.Marshal(map[string]interface{}{
		"video":  newName,
		"events": events,
		"scenes": scenes,
	})
	if err != nil {
		return "", false
	}
	return string(payload), true
}

// parentVirtualPath returns the client-facing directory that contains the given
// client path, used as the `destDir` of the operation message so the browse
// view refreshes when the job completes.
func parentVirtualPath(path string) string {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." || dir == "" {
		return "/"
	}
	return dir
}
