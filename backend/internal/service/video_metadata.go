package service

import (
	"encoding/json"
	"net/http"
	"os"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

// videoEventsResponse is the normalized payload returned to the player.
// `events` is always a list of [start, end] second pairs regardless of whether
// the data came from the container tag or the sidecar JSON.
type videoEventsResponse struct {
	Events [][]float64 `json:"events"`
	Source string      `json:"source"`
}

// GetVideoMetadata returns the detected event spans for a video.
// Resolution order: embedded MP4 tag -> sibling sidecar JSON -> 404.
// @Summary      Video Event Metadata
// @Description  Return detected event spans embedded in the container, falling back to the sidecar JSON.
// @Tags         Media
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true  "Relative video path"
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/video/metadata/file/{filepath} [get]
func GetVideoMetadata(c *gin.Context, cfg *config.CloudConfig) {
	relPath := c.Param("filepath")
	fullPath, err := util.SanitizeRepoPath(cfg.Server.FileRoot, relPath)
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	stat, err := os.Stat(fullPath)
	if err != nil || stat.IsDir() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if events, ok := readEmbeddedVideoEvents(fullPath); ok {
		httpx.OK(c, http.StatusOK, videoEventsResponse{Events: events, Source: "embedded"})
		return
	}

	if events, ok := readSidecarVideoEvents(fullPath); ok {
		httpx.OK(c, http.StatusOK, videoEventsResponse{Events: events, Source: "sidecar"})
		return
	}

	httpx.Err(c, http.StatusNotFound, "no event metadata found")
}

// readEmbeddedVideoEvents reads the description/comment MP4 tags and parses the
// first one that contains a usable event list.
func readEmbeddedVideoEvents(fullPath string) ([][]float64, bool) {
	tags, err := util.ReadMP4Tags(fullPath)
	if err != nil {
		logger.L.Debug("mp4 tag read failed", "path", fullPath, "err", err)
		return nil, false
	}

	for _, key := range []string{"description", "comment"} {
		payload, ok := tags[key]
		if !ok || payload == "" {
			continue
		}
		if events, ok := parseEventsJSON([]byte(payload)); ok {
			return events, true
		}
	}
	return nil, false
}

// readSidecarVideoEvents reads `<dir>/.vid_metadata/<filename>_timestamps.json`.
func readSidecarVideoEvents(fullPath string) ([][]float64, bool) {
	sidecar := util.SidecarPath(fullPath)
	data, err := os.ReadFile(sidecar)
	if err != nil {
		return nil, false
	}
	return parseEventsJSON(data)
}

// parseEventsJSON tolerantly extracts [start, end] pairs from any of the shapes
// the pipeline or third-party tools may produce:
//
//	{"events": [[s,e], ...]}                      (detector sidecar / our tag)
//	{"e": [[s,e], ...]}                           (compact sidecar)
//	{"scenes": [{"start":s,"end":e}, ...]}        (portable viewer shape)
//	[[s,e], ...]                                  (bare array)
//
// Entries may be [s,e] arrays or {start,end} objects in any of the containers.
func parseEventsJSON(data []byte) ([][]float64, bool) {
	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, false
	}

	if arr, ok := root.([]interface{}); ok {
		return normalizeEventList(arr)
	}

	obj, ok := root.(map[string]interface{})
	if !ok {
		return nil, false
	}

	for _, key := range []string{"events", "e", "scenes"} {
		if arr, ok := obj[key].([]interface{}); ok {
			if events, ok := normalizeEventList(arr); ok {
				return events, true
			}
		}
	}

	// A single {start,end} object.
	if events, ok := normalizeEventEntry(obj); ok {
		return [][]float64{events}, true
	}

	return nil, false
}

// normalizeEventList converts a heterogeneous list of [s,e] / {start,end}
// entries into a sorted, clamped pair list.
func normalizeEventList(arr []interface{}) ([][]float64, bool) {
	events := make([][]float64, 0, len(arr))
	for _, item := range arr {
		pair, ok := eventPair(item)
		if !ok {
			continue
		}
		events = append(events, pair)
	}
	if len(events) == 0 {
		return nil, false
	}
	sortEventPairs(events)
	return events, true
}

func normalizeEventEntry(obj map[string]interface{}) ([]float64, bool) {
	start, okStart := numberValue(obj["start"])
	end, okEnd := numberValue(obj["end"])
	if !okStart || !okEnd {
		return nil, false
	}
	return orderedPair(start, end), true
}

func eventPair(item interface{}) ([]float64, bool) {
	switch v := item.(type) {
	case []interface{}:
		if len(v) < 2 {
			return nil, false
		}
		start, okStart := numberValue(v[0])
		end, okEnd := numberValue(v[1])
		if !okStart || !okEnd {
			return nil, false
		}
		return orderedPair(start, end), true
	case map[string]interface{}:
		return normalizeEventEntry(v)
	default:
		return nil, false
	}
}

func numberValue(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func orderedPair(a, b float64) []float64 {
	if a > b {
		a, b = b, a
	}
	return []float64{a, b}
}

func sortEventPairs(events [][]float64) {
	// Small lists: insertion sort keeps it dependency-free and stable.
	for i := 1; i < len(events); i++ {
		for j := i; j > 0 && events[j][0] < events[j-1][0]; j-- {
			events[j], events[j-1] = events[j-1], events[j]
		}
	}
}
