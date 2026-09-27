package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
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
	// Bound encoder threads so a single stream cannot monopolise every core;
	// x264 otherwise sizes its pool from the CPU count. A single 1080p60 stream
	// stays comfortably faster than realtime at six threads on a 16-thread
	// desktop-class CPU (benchmarked), while leaving room for the rest of the
	// process.
	compressDefaultThreads = 6

	compressReadBuffer = 64 * 1024

	compressContentType = "video/mp4"
	// Fragmented MP4: flush an empty moov immediately, then emit fragments as
	// they are produced so the response can be streamed with chunked encoding.
	compressMovFlags = "+frag_keyframe+empty_moov+default_base_moof"

	// Pin the H.264 profile/level so the browser-side MSE (Media Source
	// Extensions) codec string is deterministic: "avc1.64002a" is High profile
	// (0x64) + level 4.2 (0x2a). CRF/preset control quality; the profile only
	// enables tools x264 already uses by default, and the level is just a
	// capability ceiling (covers up to 1080p60, and the player only requests
	// 480/720/1080 so the scale filter caps the short side at 1080).
	compressProfile = "high"
	compressLevel   = "4.2"

	// A keyframe is forced every second *of output time*. Expressing the
	// interval in time (rather than a frame count) keeps it correct at any
	// source rate: a fixed -g 24 would silently mean 1s at 24fps but 0.4s at
	// 60fps. Scene-cut keyframes stay enabled so a hard cut refreshes
	// immediately; keyint_min (in frames, derived from the output rate) stops a
	// rapid-fire cut sequence from inserting a burst of keyframes.
	compressKeyframeIntervalSecs    = 1
	compressSceneCutThreshold       = 40
	compressMinKeyframeIntervalSecs = 0.5

	// Bounds for a client-requested compression resolution (the shorter side,
	// in px). A malformed query outside this range falls back to the default.
	qualityMinShortSide = 144
	qualityMaxShortSide = 2160
)

// compressTier is a selectable playback quality. Unlike a bare resolution cap,
// each tier carries its own rate-control profile so the delivered stream has a
// predictable bandwidth ceiling (maxrate) instead of the unbounded VBR a plain
// -crf produces. That ceiling is what lets the player reason about the network.
type compressTier struct {
	ShortSide int // target shorter side, px
	FPS       int // frame-rate cap; sources at or below this pass through
	MaxRate   int // sustained video ceiling, bits/s, averaged over BufSize
	BufSize   int // VBV window, bits (BufSize/MaxRate seconds)
	AudioRate int // AAC target, bits/s
}

// BufSize is 3x MaxRate on purpose: a 3s VBV window absorbs the transient burst
// a scene cut (or a fresh segment start — every seek restarts ffmpeg) produces.
// The intended content is cut-heavy, so those bursts are frequent. Audio is
// 128k only at 1080p; 96k is transparent for video dialogue and its 32 kbps
// weighs more against the smaller lower-tier ceilings.
var compressTiers = []compressTier{
	{ShortSide: 480, FPS: 60, MaxRate: 2_200_000, BufSize: 6_600_000, AudioRate: 96_000},
	{ShortSide: 720, FPS: 60, MaxRate: 4_500_000, BufSize: 13_500_000, AudioRate: 96_000},
	{ShortSide: 1080, FPS: 60, MaxRate: 8_000_000, BufSize: 24_000_000, AudioRate: 128_000},
}

// tierForShortSide resolves a requested shorter side to a rate-control profile:
// an exact tier when one exists, otherwise the smallest tier at least as large
// as the request (so an in-between size inherits the next tier's ceiling), and
// the largest tier when the request exceeds every tier.
func tierForShortSide(shortSide int) compressTier {
	var chosen compressTier
	foundAbove := false
	for _, tier := range compressTiers {
		if tier.ShortSide == shortSide {
			return tier
		}
		if tier.ShortSide > shortSide {
			// First tier above the request wins; later (larger) tiers must not
			// overwrite it.
			if !foundAbove {
				chosen = tier
				foundAbove = true
			}
			continue
		}
		// Below the request: remember the largest seen for oversized requests.
		chosen = tier
	}
	return chosen
}

// compressParams are the resolved inputs for one transcode invocation.
type compressParams struct {
	Path      string
	Start     int // input seek offset, seconds
	ShortSide int // requested shorter side, used for scaling
	Tier      compressTier
	SourceFPS float64 // source frame rate; 0 when unknown
	CRF       int
	Threads   int
	Preset    string
}

// compressOutputFPS is the frame rate the encoder will emit: the source rate
// capped at the tier limit, never upscaled. Returns 0 when the source rate is
// unknown.
func compressOutputFPS(fpsCap int, sourceFPS float64) float64 {
	if sourceFPS <= 0 {
		return 0
	}
	if sourceFPS > float64(fpsCap) {
		return float64(fpsCap)
	}
	return sourceFPS
}

// compressKeyintMin is the minimum keyframe interval in frames — half a second
// at the *output* frame rate. When the source rate is unknown it assumes the
// cap, which only ever loosens the floor.
func compressKeyintMin(fpsCap int, sourceFPS float64) int {
	out := compressOutputFPS(fpsCap, sourceFPS)
	if out <= 0 {
		out = float64(fpsCap)
	}
	minimum := int(math.Round(out * compressMinKeyframeIntervalSecs))
	if minimum < 1 {
		minimum = 1
	}
	return minimum
}

// compressVideoFilter builds the scale + frame-rate filtergraph.
//
// The frame-rate filter is applied *only* when the source exceeds the cap.
// Forcing a constant output rate when there is nothing to cap is not harmless:
// a variable-frame-rate source (common on phones/action cams) gets its frame
// timing re-quantized onto a fixed grid, which shows up as subtle judder. When
// the source is already at or below the cap its cadence is left untouched.
func compressVideoFilter(shortSide, fpsCap int, sourceFPS float64) string {
	scale := compressScaleFilter(shortSide)
	if sourceFPS <= 0 {
		// Unknown rate: clamp so a high-rate source is still bounded, but the
		// expression leaves a low-rate source untouched.
		return fmt.Sprintf("%s,fps='min(source_fps,%d)'", scale, fpsCap)
	}
	if sourceFPS <= float64(fpsCap) {
		return scale
	}
	return fmt.Sprintf("%s,fps=%d", scale, fpsCap)
}

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

func compressThreads(cfg *config.CloudConfig) int {
	if cfg.Server.VideoCompressThreads > 0 {
		return cfg.Server.VideoCompressThreads
	}
	return compressDefaultThreads
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
// of the source to a fragmented MP4 on stdout. start>0 is applied as an *input*
// seek (-ss before -i) so ffmpeg decodes only from the nearest keyframe and
// output timestamps are rebased to 0. threads bounds both the decoder and the
// x264 encoder pool.
func buildCompressArgs(p compressParams) []string {
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
	if p.Start > 0 {
		args = append(args, "-ss", strconv.Itoa(p.Start))
	}
	args = append(args, "-i", p.Path)
	args = append(args,
		"-threads", strconv.Itoa(p.Threads),
		"-vf", compressVideoFilter(p.ShortSide, p.Tier.FPS, p.SourceFPS),
		"-c:v", "libx264",
		"-profile:v", compressProfile,
		"-level:v", compressLevel,
		"-preset", p.Preset,
		"-crf", strconv.Itoa(p.CRF),
		"-pix_fmt", "yuv420p",
		// No B-frames: fragmented MP4 + reordering produced a ~0.1s timestamp
		// gap at the start of every segment, visible as a hitch on each seek.
		// B-frames are also pointless for a low-latency, restart-on-seek stream.
		"-bf", "0",
		// Predictable bandwidth ceiling: maxrate is the sustained ceiling and
		// bufsize the window it is enforced over (3x, see compressTiers).
		"-maxrate", strconv.Itoa(p.Tier.MaxRate),
		"-bufsize", strconv.Itoa(p.Tier.BufSize),
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", compressKeyframeIntervalSecs),
		"-sc_threshold", strconv.Itoa(compressSceneCutThreshold),
		"-keyint_min", strconv.Itoa(compressKeyintMin(p.Tier.FPS, p.SourceFPS)),
		"-c:a", "aac",
		"-b:a", strconv.Itoa(p.Tier.AudioRate),
		"-ac", "2",
		"-movflags", compressMovFlags,
		"-f", "mp4",
		"pipe:1",
	)
	return args
}

// newCompressedCommand builds the transcode command, wrapping ffmpeg in prlimit
// when a per-process address-space cap is configured. prlimit is verified at
// startup alongside ffmpeg (see cmd/main.go), so the wrapper is always present.
func newCompressedCommand(ctx context.Context, cfg *config.CloudConfig, args []string) *exec.Cmd {
	if mb := cfg.Server.VideoCompressMemoryLimitMB; mb > 0 {
		asBytes := strconv.FormatInt(int64(mb)*1024*1024, 10)
		return util.NewCommand(ctx, "prlimit", append([]string{"--as=" + asBytes, "ffmpeg"}, args...)...)
	}
	return util.NewCommand(ctx, "ffmpeg", args...)
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
// @Failure      429  {object}  map[string]interface{}
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

	// Bound the number of concurrent transcodes. Each ffmpeg can use
	// significant CPU and RAM, so reject immediately when the host is at
	// capacity instead of queueing a long-lived stream. The client surfaces
	// this as "server busy" and falls back to original playback.
	sem := GetVideoCompressSemaphore()
	if !sem.TryAcquire() {
		c.Header("Retry-After", "5")
		httpx.Err(c, http.StatusTooManyRequests, "video transcoding is busy, retry shortly")
		return
	}
	defer sem.Release()

	start, _ := parseStartParam(c.Query("start"))
	requested := parseQualityParam(c.Query("quality"), compressShortSide(cfg))
	tier := tierForShortSide(requested)

	// The source frame rate drives keyframe-interval math and fps
	// normalization. A probe failure is non-fatal: fall back to the tier cap
	// and the filter's source_fps clamp.
	sourceFPS := 0.0
	if meta, err := probeVideoMeta(c.Request.Context(), fullPath); err == nil {
		sourceFPS = meta.FPS
	} else {
		logger.L.Debug("compress probe failed; using frame-rate cap", "file", fullPath, "err", err)
	}

	args := buildCompressArgs(compressParams{
		Path:      fullPath,
		Start:     start,
		ShortSide: requested,
		Tier:      tier,
		SourceFPS: sourceFPS,
		CRF:       compressCRF(cfg),
		Threads:   compressThreads(cfg),
		Preset:    compressPreset(cfg),
	})

	// The request context is cancelled when the client disconnects (seek, tab
	// close, page navigation); util.NewCommand kills the whole ffmpeg process
	// group in that case, so no transcode is left running in the background.
	cmd := newCompressedCommand(c.Request.Context(), cfg, args)
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
	// FPS is the source frame rate (0 when unknown). The transcode endpoint
	// uses it to normalize frame rate and size the keyframe floor.
	FPS float64 `json:"fps,omitempty"`
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
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		Duration     string `json:"duration"`
		AvgFrameRate string `json:"avg_frame_rate"`
		RFrameRate   string `json:"r_frame_rate"`
	} `json:"streams"`
}

// probeVideoMeta runs ffprobe and returns the container duration (falling back
// to the first video stream when the container does not report one) plus the
// video dimensions and frame rate.
func probeVideoMeta(parent context.Context, path string) (videoDurationResponse, error) {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()

	cmd := util.NewCommand(ctx,
		"ffprobe", "-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "format=duration:stream=width,height,duration,avg_frame_rate,r_frame_rate",
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
		// avg_frame_rate is the effective rate (VFR-friendly); fall back to the
		// base rate only when it is unusable.
		result.FPS = parseFrameRate(out.Streams[0].AvgFrameRate)
		if result.FPS <= 0 {
			result.FPS = parseFrameRate(out.Streams[0].RFrameRate)
		}
	}

	return result, nil
}

// parseFrameRate parses an ffprobe rational frame rate ("30000/1001", "60/1")
// into frames per second. Missing, malformed and non-positive values yield 0.
func parseFrameRate(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	num, den := raw, "1"
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		num, den = raw[:i], raw[i+1:]
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
	if err != nil {
		return 0
	}
	d, err := strconv.ParseFloat(strings.TrimSpace(den), 64)
	if err != nil || d == 0 {
		return 0
	}
	fps := n / d
	if math.IsNaN(fps) || math.IsInf(fps, 0) || fps <= 0 {
		return 0
	}
	return fps
}

// videoTierResponse is one selectable compressed-playback quality exposed to
// the client, with the bandwidth ceiling it may rely on.
type videoTierResponse struct {
	ShortSide    int `json:"shortSide"`
	FPS          int `json:"fps"`
	MaxBitrate   int `json:"maxBitrate"`   // video ceiling, bits/s
	AudioBitrate int `json:"audioBitrate"` // audio target, bits/s
	Ceiling      int `json:"ceiling"`      // total ceiling incl. audio + container
}

// videoTierCeiling turns a tier's rate profile into the total bandwidth ceiling
// the player may assume: the video maxrate plus audio, with a small allowance
// for container overhead.
func videoTierCeiling(t compressTier) int {
	total := t.MaxRate + t.AudioRate
	return total + total/100
}

// GetVideoQualityTiers lists the selectable compressed-playback qualities and
// the bandwidth ceiling of each, so the player can decide whether a tier fits
// the current network before switching to it.
//
// @Summary      Compressed Video Quality Tiers
// @Description  List the available compressed-playback qualities and their bandwidth ceilings.
// @Tags         Media
// @Produce      json
// @Security     BearerAuth
// @Security     CookieAuth
// @Success      200  {object}  map[string]interface{}
// @Router       /api/user/video/capabilities [get]
func GetVideoQualityTiers(c *gin.Context) {
	tiers := make([]videoTierResponse, 0, len(compressTiers))
	for _, tier := range compressTiers {
		tiers = append(tiers, videoTierResponse{
			ShortSide:    tier.ShortSide,
			FPS:          tier.FPS,
			MaxBitrate:   tier.MaxRate,
			AudioBitrate: tier.AudioRate,
			Ceiling:      videoTierCeiling(tier),
		})
	}
	httpx.OK(c, http.StatusOK, gin.H{"tiers": tiers})
}

func parseSeconds(raw string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	return value
}
