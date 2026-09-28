package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// verifyProbeTimeout bounds the ffprobe calls used to check a remux output.
const verifyProbeTimeout = 30 * time.Second

type verifyProbe struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
		Duration  string `json:"duration"`
	} `json:"streams"`
}

// VerifyVideoOutput checks that a freshly written remux output is a usable video
// before the caller atomically replaces the original. ffmpeg can exit 0 on a
// truncated or tag-less write, so this guards against:
//
//   - an empty output file;
//   - an output ffprobe cannot open or that has no video stream;
//   - a duration that drifts from the source beyond a small tolerance;
//   - description/comment tags that were requested but did not round-trip.
//
// A nil error means the output is safe to rename over srcPath. When metadata is
// empty (rotation-only remux) the tag check is skipped.
func VerifyVideoOutput(tmpPath, srcPath string, metadata map[string]string) error {
	info, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("verify remux output: %w", err)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("verify remux output: empty file")
	}

	tmpDur, hasVideo, err := probeVideoForVerify(tmpPath)
	if err != nil {
		return fmt.Errorf("verify remux output: %w", err)
	}
	if !hasVideo {
		return fmt.Errorf("verify remux output: no video stream")
	}

	// Duration is only a sanity check: some containers do not report one, and
	// the source may already be 0/unknown. Only compare when both are known.
	if srcDur, _, srcErr := probeVideoForVerify(srcPath); srcErr == nil && srcDur > 0 && tmpDur > 0 {
		tolerance := math.Max(2.0, srcDur*0.02)
		if math.Abs(tmpDur-srcDur) > tolerance {
			return fmt.Errorf("verify remux output: duration mismatch (src=%.3fs tmp=%.3fs)", srcDur, tmpDur)
		}
	}

	return verifyEmbeddedTags(tmpPath, metadata)
}

// verifyEmbeddedTags round-trips the description/comment tags the caller asked
// ffmpeg to embed, catching silent tag drops.
func verifyEmbeddedTags(tmpPath string, metadata map[string]string) error {
	wantDesc := metadata["description"]
	wantComment := metadata["comment"]
	if wantDesc == "" && wantComment == "" {
		return nil
	}

	tags, err := ReadMP4Tags(tmpPath)
	if err != nil {
		return fmt.Errorf("verify remux output: read tags: %w", err)
	}
	if wantDesc != "" && tags["description"] != wantDesc {
		return fmt.Errorf("verify remux output: description tag missing or altered")
	}
	if wantComment != "" && tags["comment"] != wantComment {
		return fmt.Errorf("verify remux output: comment tag missing or altered")
	}
	return nil
}

// probeVideoForVerify runs a short ffprobe and reports the container duration
// (0 when unknown) and whether a video stream is present.
func probeVideoForVerify(path string) (float64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), verifyProbeTimeout)
	defer cancel()

	cmd := NewCommand(ctx,
		"ffprobe", "-v", "error",
		"-show_entries", "format=duration:stream=codec_type,duration",
		"-of", "json", path)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, false, fmt.Errorf("ffprobe: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var out verifyProbe
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return 0, false, fmt.Errorf("ffprobe json parse: %w", err)
	}

	hasVideo := false
	dur := parseProbeSeconds(out.Format.Duration)
	for _, s := range out.Streams {
		if s.CodecType != "video" {
			continue
		}
		hasVideo = true
		if dur <= 0 {
			if d := parseProbeSeconds(s.Duration); d > 0 {
				dur = d
			}
		}
	}
	return dur, hasVideo, nil
}

func parseProbeSeconds(raw string) float64 {
	if raw == "" || raw == "N/A" {
		return 0
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}
