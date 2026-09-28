package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/state"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"
)

// errVideoMetadataFailed is the generic message surfaced in the operation queue.
// The detailed cause (ffmpeg stderr, IO errors) stays in the server log.
var errVideoMetadataFailed = errors.New("Saving events failed")

// minEventDuration drops degenerate spans created by a stray tap.
const minEventDuration = 0.05

// VideoMetadataOpType identifies the async "save edited highlights" job.
const VideoMetadataOpType = "video-metadata"

// VideoMetadataCommitReq is the payload for saving edited event highlights.
// Events are absolute [start, end] second pairs; an empty list clears the
// highlights.
type VideoMetadataCommitReq struct {
	Path   string      `json:"path"`
	Events [][]float64 `json:"events"`
	OpID   string      `json:"opId"`
}

// VideoMetadataCommit validates an edited highlight list and queues an async job
// that overwrites both the AI sidecar and (for MP4-family files) the embedded
// container tags. Like copy/move/rename-done it returns an operation id and
// streams progress over the WebSocket.
// @Summary      Update Video Event Metadata
// @Description  Save edited event spans, overwriting the sidecar and embedded MP4 tags.
// @Tags         Media
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        body  body      service.VideoMetadataCommitReq  true  "Request with path, events, opId"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Router       /api/user/video/metadata/commit [post]
func VideoMetadataCommit(c *gin.Context, cfg *config.CloudConfig) {
	var req VideoMetadataCommitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Err(c, http.StatusBadRequest, "invalid request")
		return
	}

	username := c.GetString(authgin.KeyUsername)
	requestID := c.GetString("request_id")

	opID, err := StartVideoMetadataCommit(req, cfg, requestID, username)
	if err != nil {
		logger.L.Error("video metadata commit failed to start", "err", err, "path", req.Path)
		httpx.Err(c, http.StatusBadRequest, err.Error())
		return
	}

	httpx.OK(c, http.StatusOK, gin.H{
		"message": "Event update started",
		"opId":    opID,
	})
}

// StartVideoMetadataCommit validates the request and enqueues the commit job. It
// does not move the file: unlike rename-done, highlight editing rewrites the
// video in place at its existing path.
func StartVideoMetadataCommit(req VideoMetadataCommitReq, cfg *config.CloudConfig, requestID, username string) (string, error) {
	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, req.Path)
	if err != nil {
		return "", fmt.Errorf("access forbidden: %w", err)
	}

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("video not found")
	}

	// Clamp to the real duration so a stale client cannot persist out-of-range
	// spans; an unknown duration (0) leaves the raw values clamped to >= 0.
	duration := 0.0
	if meta, probeErr := probeVideoMeta(context.Background(), fullPath); probeErr == nil {
		duration = meta.Duration
	}
	events := normalizeVideoEvents(req.Events, duration)

	opID := req.OpID
	if opID == "" {
		opID = util.GenerateOpID()
	}
	state.SetOperationOwner(opID, username)

	opName := "Saving events " + util.TruncateString(filepath.Base(req.Path), 20)
	virtualParent := parentVirtualPath(req.Path)

	tracker := util.NewProgressTracker()
	tracker.TotalBytes = info.Size()
	tracker.TotalFiles = 1
	state.SetProgress(opID, tracker)

	submitAsyncJob(opID, VideoMetadataOpType, opName, tracker, true, virtualParent, requestID, username, func(t *util.ProgressTracker) error {
		if err := processVideoMetadataCommit(fullPath, events, t); err != nil {
			logger.L.Error("video metadata commit failed", "opId", opID, "path", fullPath, "err", err)
			return errVideoMetadataFailed
		}
		return nil
	})

	logger.L.Debug("video metadata commit queued", "opId", opID, "path", req.Path, "events", len(events))
	return opID, nil
}

// processVideoMetadataCommit performs the actual work on the file-operation
// worker. It captures the pre-edit spans once, then overwrites the embedded MP4
// tag (in place, atomically) and the sidecar so both stores agree.
func processVideoMetadataCommit(fullPath string, events [][]float64, tracker *util.ProgressTracker) error {
	base := filepath.Base(fullPath)

	// Capture the detected (pre-edit) spans before the tag is overwritten: keep
	// an existing events_original, otherwise fall back to whatever is currently
	// stored (embedded first, then sidecar).
	original, haveOriginal := readSidecarEventsOriginal(fullPath)
	embedded, hadEmbedded := readEmbeddedVideoEvents(fullPath)
	current, haveCurrent := embedded, hadEmbedded
	if !haveCurrent {
		if sc, ok := readSidecarVideoEvents(fullPath); ok {
			current, haveCurrent = sc, true
		}
	}
	if !haveOriginal {
		if haveCurrent {
			original = current
		} else {
			original = events
		}
	}

	// Embed only when the container already carried an embedded tag, so a
	// sidecar-based file is never remuxed and stays sidecar-only. When a tag
	// exists it is always rewritten (even to empty, clearing deleted events).
	willEmbed := util.IsMP4FamilyExt(filepath.Ext(fullPath)) && hadEmbedded
	if willEmbed {
		payload, err := buildEmbedPayloadFromEvents(base, events)
		if err != nil {
			return err
		}
		metadata := map[string]string{"description": payload, "comment": payload}
		tmpDir := filepath.Join(filepath.Dir(fullPath), util.MetadataDirName, "temp")
		if err := util.RemuxVideoAtomic(context.Background(), fullPath, fullPath, tmpDir, 0, metadata, tracker.Update); err != nil {
			return err
		}
	}

	sidecar, err := marshalEventsSidecar(base, events, original)
	if err != nil {
		return err
	}
	if err := util.WriteFileAtomic(util.SidecarPath(fullPath), sidecar, 0644); err != nil {
		return fmt.Errorf("failed to write metadata sidecar: %w", err)
	}

	tracker.FileCompleted()
	tracker.FinalLog()
	return nil
}

// normalizeVideoEvents clamps spans to [0, duration], orders them, drops
// degenerate/duplicate entries and sorts by start.
func normalizeVideoEvents(events [][]float64, duration float64) [][]float64 {
	out := make([][]float64, 0, len(events))
	for _, e := range events {
		if len(e) < 2 {
			continue
		}
		start, end := e[0], e[1]
		if math.IsNaN(start) || math.IsInf(start, 0) || math.IsNaN(end) || math.IsInf(end, 0) {
			continue
		}
		if start > end {
			start, end = end, start
		}
		start = math.Max(start, 0)
		end = math.Max(end, 0)
		if duration > 0 {
			start = math.Min(start, duration)
			end = math.Min(end, duration)
		}
		if end-start < minEventDuration {
			continue
		}
		out = append(out, []float64{start, end})
	}

	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })

	deduped := out[:0]
	for i, e := range out {
		if i > 0 && e[0] == out[i-1][0] && e[1] == out[i-1][1] {
			continue
		}
		deduped = append(deduped, e)
	}
	return deduped
}

// buildEmbedPayloadFromEvents renders the portable superset payload embedded in
// the container: the `events` pairs the player parses plus the `scenes` array
// desktop/doc consumers expect.
func buildEmbedPayloadFromEvents(newName string, events [][]float64) (string, error) {
	if events == nil {
		events = [][]float64{}
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
		return "", err
	}
	return string(payload), nil
}

type eventScene struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// marshalEventsSidecar renders the in-place `.vid_metadata` payload, preserving
// the detected spans as events_original for a future "revert to detected".
func marshalEventsSidecar(video string, events, original [][]float64) ([]byte, error) {
	if events == nil {
		events = [][]float64{}
	}
	if original == nil {
		original = [][]float64{}
	}
	scenes := make([]eventScene, 0, len(events))
	for _, e := range events {
		scenes = append(scenes, eventScene{Start: e[0], End: e[1]})
	}
	return json.MarshalIndent(map[string]interface{}{
		"video":           video,
		"events":          events,
		"scenes":          scenes,
		"events_original": original,
		"updated_at":      time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
}

// readSidecarEventsOriginal returns the previously captured detected spans, if
// this sidecar has been edited before.
func readSidecarEventsOriginal(fullPath string) ([][]float64, bool) {
	return readSidecarEventsOriginalAt(util.SidecarPath(fullPath))
}

// readSidecarEventsOriginalAt reads the preserved `events_original` list from an
// explicit sidecar path. Used when the caller already holds the sidecar location
// (e.g. rename-done captures it before moving the source out of the browse tree).
func readSidecarEventsOriginalAt(sidecarPath string) ([][]float64, bool) {
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		return nil, false
	}
	var obj struct {
		EventsOriginal [][]float64 `json:"events_original"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, false
	}
	if len(obj.EventsOriginal) == 0 {
		return nil, false
	}
	return obj.EventsOriginal, true
}
