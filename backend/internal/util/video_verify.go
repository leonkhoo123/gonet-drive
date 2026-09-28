package util

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// verifyProbeTimeout bounds the ffprobe calls used to check a remux output.
const verifyProbeTimeout = 30 * time.Second

type verifyStream struct {
	CodecType  string `json:"codec_type"`
	CodecName  string `json:"codec_name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	SampleRate string `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Duration   string `json:"duration"`
}

type verifyProbeResult struct {
	// Duration is the container duration (falling back to the first video
	// stream), 0 when unknown.
	Duration float64
	HasVideo bool
	Streams  []verifyStream
}

// VerifyVideoOutput checks that a freshly written remux output is a usable video
// before the caller atomically replaces the original. ffmpeg can exit 0 on a
// truncated, wrong-stream or tag-less write, so this guards against:
//
//   - an empty output file;
//   - an output ffprobe cannot open or that has no video stream;
//   - a stream inventory that differs from the source (count / codec / type /
//     dimensions / audio layout) — a `-c copy` remux must be identical;
//   - a duration that drifts from the source beyond a small tolerance;
//   - description/comment tags that were requested but did not round-trip;
//   - for MP4-family output, a missing faststart (moov after mdat).
//
// All checks are header-only (ffprobe + box scan); nothing is decoded. A nil
// error means the output is safe to rename over srcPath. When metadata is empty
// (rotation-only remux) the tag check is skipped.
func VerifyVideoOutput(tmpPath, srcPath string, metadata map[string]string) error {
	info, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("verify remux output: %w", err)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("verify remux output: empty file")
	}

	tmp, err := probeVideoForVerify(tmpPath)
	if err != nil {
		return fmt.Errorf("verify remux output: %w", err)
	}
	if !tmp.HasVideo {
		return fmt.Errorf("verify remux output: no video stream")
	}

	// Compare against the source when it is probeable. A stream-copy remux keeps
	// every stream byte-for-byte, so the inventories must match and the duration
	// should only differ by container rounding.
	if src, srcErr := probeVideoForVerify(srcPath); srcErr == nil {
		if src.Duration > 0 && tmp.Duration > 0 {
			tolerance := math.Max(1.0, src.Duration*0.002)
			if math.Abs(tmp.Duration-src.Duration) > tolerance {
				return fmt.Errorf("verify remux output: duration mismatch (src=%.3fs tmp=%.3fs)", src.Duration, tmp.Duration)
			}
		}
		if msg := streamFingerprintMismatch(src.Streams, tmp.Streams); msg != "" {
			return fmt.Errorf("verify remux output: %s", msg)
		}
	}

	if err := verifyEmbeddedTags(tmpPath, metadata); err != nil {
		return err
	}

	if IsMP4FamilyExt(filepath.Ext(tmpPath)) {
		checked, moovFirst, err := mp4MoovPrecedesMdat(tmpPath)
		if err == nil && checked && !moovFirst {
			return fmt.Errorf("verify remux output: moov not before mdat (faststart failed)")
		}
	}

	return nil
}

// streamFingerprintMismatch returns a human-readable description of the first
// way the remux output's streams differ from the source, or "" when they match.
func streamFingerprintMismatch(src, tmp []verifyStream) string {
	if len(src) != len(tmp) {
		return fmt.Sprintf("stream count changed (%d -> %d)", len(src), len(tmp))
	}
	for i := range src {
		a, b := src[i], tmp[i]
		if a.CodecType != b.CodecType || a.CodecName != b.CodecName {
			return fmt.Sprintf("stream %d codec changed (%s/%s -> %s/%s)",
				i, a.CodecType, a.CodecName, b.CodecType, b.CodecName)
		}
		switch a.CodecType {
		case "video":
			if a.Width != b.Width || a.Height != b.Height {
				return fmt.Sprintf("stream %d size changed (%dx%d -> %dx%d)",
					i, a.Width, a.Height, b.Width, b.Height)
			}
		case "audio":
			if a.SampleRate != b.SampleRate || a.Channels != b.Channels {
				return fmt.Sprintf("stream %d audio layout changed (%sHz/%dch -> %sHz/%dch)",
					i, a.SampleRate, a.Channels, b.SampleRate, b.Channels)
			}
		}
	}
	return ""
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

// mp4MoovPrecedesMdat reports whether the top-level `moov` box appears before
// `mdat` (i.e. the file is faststart/front-loaded). checked is false when the
// layout is not a plain MP4 (either box absent), so callers can skip the check.
// Only box headers are read; payloads are seeked over.
func mp4MoovPrecedesMdat(path string) (checked, moovFirst bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return false, false, err
	}
	end := stat.Size()

	var moovPos, mdatPos int64 = -1, -1
	var pos int64
	for pos+8 <= end {
		if _, err := f.Seek(pos, io.SeekStart); err != nil {
			return false, false, err
		}
		var header [8]byte
		if _, err := io.ReadFull(f, header[:]); err != nil {
			break
		}
		size := int64(binary.BigEndian.Uint32(header[0:4]))
		boxType := string(header[4:8])
		headerLen := int64(8)
		switch size {
		case 1:
			var large [8]byte
			if _, err := io.ReadFull(f, large[:]); err != nil {
				break
			}
			size = int64(binary.BigEndian.Uint64(large[:]))
			headerLen = 16
		case 0:
			size = end - pos
		}
		if size < headerLen || pos+size > end {
			break
		}
		if boxType == "moov" && moovPos < 0 {
			moovPos = pos
		}
		if boxType == "mdat" && mdatPos < 0 {
			mdatPos = pos
		}
		if moovPos >= 0 && mdatPos >= 0 {
			break
		}
		pos += size
	}

	if moovPos < 0 || mdatPos < 0 {
		return false, false, nil
	}
	return true, moovPos < mdatPos, nil
}

// probeVideoForVerify runs a short header-only ffprobe and returns the container
// duration, whether a video stream is present, and the stream inventory.
func probeVideoForVerify(path string) (verifyProbeResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), verifyProbeTimeout)
	defer cancel()

	cmd := NewCommand(ctx,
		"ffprobe", "-v", "error",
		"-show_entries", "format=duration:stream=codec_type,codec_name,width,height,sample_rate,channels,duration",
		"-of", "json", path)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return verifyProbeResult{}, fmt.Errorf("ffprobe: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var out struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []verifyStream `json:"streams"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return verifyProbeResult{}, fmt.Errorf("ffprobe json parse: %w", err)
	}

	result := verifyProbeResult{Streams: out.Streams}
	result.Duration = parseProbeSeconds(out.Format.Duration)
	for _, s := range out.Streams {
		if s.CodecType != "video" {
			continue
		}
		result.HasVideo = true
		if result.Duration <= 0 {
			if d := parseProbeSeconds(s.Duration); d > 0 {
				result.Duration = d
			}
		}
	}
	return result, nil
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
