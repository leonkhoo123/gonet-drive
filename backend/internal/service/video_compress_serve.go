package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/logger"
	"go-file-server/internal/util"

	"github.com/gin-gonic/gin"
)

// Defaults for the on-the-fly compression stream. They can be overridden with
// the VIDEO_COMPRESS_* env vars (see config.ServerConfig).
const (
	compressDefaultShortSide = 480
	compressDefaultCRF       = 20
	compressDefaultPreset    = "faster"

	compressAudioBitrate = "96k"
	// One keyframe per second at 24fps keeps fragments small enough that the
	// browser can start playing quickly; -sc_threshold 0 disables scene-cut
	// keyframes so fragment cadence stays predictable.
	compressGOPFrames  = 24
	compressReadBuffer = 64 * 1024

	compressContentType = "video/mp4"
	// Fragmented MP4: flush an empty moov immediately, then emit fragments as
	// they are produced so the response can be streamed with chunked encoding.
	compressMovFlags = "+frag_keyframe+empty_moov+default_base_moof"

	// Bounds for a client-requested compression resolution (the shorter side,
	// in px). A malformed query outside this range falls back to the default.
	qualityMinShortSide = 144
	qualityMaxShortSide = 2160
)

func compressShortSide(cfg *config.CloudConfig) int {
	if cfg.Server.VideoCompressShortSide > 0 {
		return cfg.Server.VideoCompressShortSide
	}
	return compressDefaultShortSide
}

func compressCRF(cfg *config.CloudConfig) int {
	if cfg.Server.VideoCompressCRF > 0 {
		return cfg.Server.VideoCompressCRF
	}
	return compressDefaultCRF
}

func compressPreset(cfg *config.CloudConfig) string {
	if cfg.Server.VideoCompressPreset != "" {
		return cfg.Server.VideoCompressPreset
	}
	return compressDefaultPreset
}

// compressScaleFilter builds an ffmpeg scale expression that caps the *shorter*
// side of the video at shortSide px, so both landscape and portrait sources are
// handled without probing their orientation. Sources whose shorter side is
// already <= shortSide are passed through untouched (no upscaling), and the
// output dimensions are always even to satisfy libx264.
func compressScaleFilter(shortSide int) string {
	// min(iw,ih) is the shorter side. When it exceeds shortSide, scale both
	// dimensions by shortSide/shorter and round down to an even number.
	width := fmt.Sprintf(
		"if(gt(min(iw\\,ih)\\,%d)\\,floor(iw*%d/min(iw\\,ih)/2)*2\\,iw)",
		shortSide, shortSide,
	)
	height := fmt.Sprintf(
		"if(gt(min(iw\\,ih)\\,%d)\\,floor(ih*%d/min(iw\\,ih)/2)*2\\,ih)",
		shortSide, shortSide,
	)
	return fmt.Sprintf("scale=w='%s':h='%s'", width, height)
}

// buildCompressArgs assembles the ffmpeg invocation that re-encodes a segment
// of path to a fragmented MP4 on stdout. start>0 is applied as an *input* seek
// (-ss before -i) so ffmpeg decodes only from the nearest keyframe and output
// timestamps are rebased to 0.
func buildCompressArgs(path string, start, shortSide, crf int, preset string) []string {
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
	if start > 0 {
		args = append(args, "-ss", strconv.Itoa(start))
	}
	args = append(args, "-i", path)
	args = append(args,
		"-vf", compressScaleFilter(shortSide),
		"-c:v", "libx264",
		"-preset", preset,
		"-crf", strconv.Itoa(crf),
		"-pix_fmt", "yuv420p",
		// No B-frames: fragmented MP4 + reordering produced a ~0.1s timestamp
		// gap at the start of every segment, visible as a hitch on each seek.
		// B-frames are also pointless for a low-latency, restart-on-seek stream.
		"-bf", "0",
		"-c:a", "aac",
		"-b:a", compressAudioBitrate,
		"-ac", "2",
		"-g", strconv.Itoa(compressGOPFrames),
		"-keyint_min", strconv.Itoa(compressGOPFrames),
		"-sc_threshold", "0",
		"-movflags", compressMovFlags,
		"-f", "mp4",
		"pipe:1",
	)
	return args
}

// parseQualityParam resolves the optional `quality` query parameter, which is
// the target shorter-side resolution in px (e.g. 480, 720, 1080). Missing or
// out-of-range values fall back to the configured default so a bad query can
// never ask ffmpeg for a degenerate frame size.
func parseQualityParam(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < qualityMinShortSide || value > qualityMaxShortSide {
		return fallback
	}
	return value
}

// parseStartParam parses the optional `start` query parameter (seconds, may be
// fractional) into a whole number of seconds. Invalid or negative values fall
// back to 0 with ok=false.
func parseStartParam(raw string) (seconds int, ok bool) {
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, false
	}
	return int(value), true
}

// ServeCompressedStream re-encodes a video to a low-resolution fragmented MP4
// and streams it as it is produced, so playback can start long before the whole
// file is transcoded. `start` selects the segment start with input seeking.
//
// @Summary      Stream Compressed Video
// @Description  Transcode a video to a low-resolution fragmented MP4 and stream it on the fly.
// @Tags         Media
// @Produce      video/mp4
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true   "Relative video path"
// @Param        start     query integer false  "Seek offset in seconds"
// @Param        quality   query integer false  "Target shorter side in px (e.g. 480, 720, 1080)"
// @Success      200  {file}  binary
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/video/stream/file/{filepath} [get]
func ServeCompressedStream(c *gin.Context, cfg *config.CloudConfig) {
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

	start, _ := parseStartParam(c.Query("start"))
	shortSide := parseQualityParam(c.Query("quality"), compressShortSide(cfg))
	args := buildCompressArgs(fullPath, start, shortSide, compressCRF(cfg), compressPreset(cfg))

	// The request context is cancelled when the client disconnects (seek, tab
	// close, page navigation); util.NewCommand kills the whole ffmpeg process
	// group in that case, so no transcode is left running in the background.
	cmd := util.NewCommand(c.Request.Context(), "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.L.Error("compress stream pipe failed", "file", fullPath, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		logger.L.Error("ffmpeg compress start failed", "file", fullPath, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Wait must run after stdout is drained (see os/exec docs) and also reaps
	// the process when the client disconnects mid-stream.
	defer func() {
		if err := cmd.Wait(); err != nil && c.Request.Context().Err() == nil {
			if msg := strings.TrimSpace(stderr.String()); msg != "" {
				logger.L.Debug("ffmpeg compress ended", "file", fullPath, "err", err, "stderr", msg)
			}
		}
	}()

	c.Header("Content-Type", compressContentType)
	c.Header("Cache-Control", "no-store")
	c.Header("Accept-Ranges", "none")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	buf := make([]byte, compressReadBuffer)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				// Client seeked away or closed the connection.
				break
			}
			c.Writer.Flush()
		}
		if readErr != nil {
			if readErr != io.EOF {
				logger.L.Debug("compress stream read ended", "file", fullPath, "err", readErr)
			}
			break
		}
	}
}

// videoDurationResponse is the payload for the duration endpoint.
type videoDurationResponse struct {
	// Duration is the source video length in seconds (0 when unknown).
	Duration float64 `json:"duration"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
}

// GetVideoDuration reports the absolute duration (and dimensions) of a video.
// The player needs the real total length because every compressed stream is a
// fresh segment starting at 0, so the browser can never learn the total itself.
//
// @Summary      Video Duration
// @Description  Probe the absolute duration and dimensions of a video file.
// @Tags         Media
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Param        filepath  path  string  true  "Relative video path"
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /api/user/video/duration/file/{filepath} [get]
func GetVideoDuration(c *gin.Context, cfg *config.CloudConfig) {
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

	meta, err := probeVideoMeta(c.Request.Context(), fullPath)
	if err != nil {
		logger.L.Error("video duration probe failed", "file", fullPath, "err", err)
		httpx.Err(c, http.StatusInternalServerError, "failed to probe video")
		return
	}

	httpx.OK(c, http.StatusOK, meta)
}

type ffprobeMeta struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Duration string `json:"duration"`
	} `json:"streams"`
}

// probeVideoMeta runs ffprobe and returns the container duration (falling back
// to the first video stream when the container does not report one) plus the
// video dimensions.
func probeVideoMeta(parent context.Context, path string) (videoDurationResponse, error) {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()

	cmd := util.NewCommand(ctx,
		"ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "format=duration:stream=width,height,duration",
		"-of", "json",
		path,
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return videoDurationResponse{}, fmt.Errorf("ffprobe: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var out ffprobeMeta
	if err := json.Unmarshal([]byte(stdout.String()), &out); err != nil {
		return videoDurationResponse{}, fmt.Errorf("ffprobe json parse: %w", err)
	}

	result := videoDurationResponse{Duration: parseSeconds(out.Format.Duration)}
	if result.Duration <= 0 && len(out.Streams) > 0 {
		result.Duration = parseSeconds(out.Streams[0].Duration)
	}
	if len(out.Streams) > 0 {
		result.Width = out.Streams[0].Width
		result.Height = out.Streams[0].Height
	}

	return result, nil
}

func parseSeconds(raw string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	return value
}
