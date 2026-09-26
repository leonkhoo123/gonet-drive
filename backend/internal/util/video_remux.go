package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go-file-server/internal/logger"
)

// remuxTimeout bounds a single remux. It is intentionally generous because the
// WORK_DIR may be a slow network share: a 10 GB clip at ~90 MB/s takes minutes.
const remuxTimeout = 1 * time.Hour

// MetadataDirName is the sibling directory (relative to the video) that holds
// the AI pipeline's `<video filename>_timestamps.json` sidecar files.
const MetadataDirName = ".vid_metadata"

// SidecarPath returns the on-disk path of the metadata sidecar that belongs to
// the given video file: <dir>/.vid_metadata/<filename>_timestamps.json.
func SidecarPath(fullPath string) string {
	return filepath.Join(
		filepath.Dir(fullPath),
		MetadataDirName,
		filepath.Base(fullPath)+"_timestamps.json",
	)
}

// RelocateSidecar moves a metadata sidecar next to destVideoPath using the same
// `.vid_metadata/<video filename>_timestamps.json` layout, and updates the
// payload's `video` field so it points at the renamed file. It is used when the
// metadata could not be embedded: finding the sidecar under
// `done/.vid_metadata/` is the visible signal that embedding was skipped.
func RelocateSidecar(sidecarPath, destVideoPath string) error {
	destDir := filepath.Join(filepath.Dir(destVideoPath), MetadataDirName)
	if err := os.MkdirAll(destDir, 0777); err != nil {
		return err
	}
	destName := filepath.Base(destVideoPath)
	destSidecar := filepath.Join(destDir, destName+"_timestamps.json")

	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		return err
	}

	// Keep the `video` field in sync with the renamed file. The filename is the
	// source of truth, but a stale field trips up tools that read it instead.
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil && obj != nil {
		obj["video"] = destName
		if updated, err := json.MarshalIndent(obj, "", "  "); err == nil {
			data = updated
		}
	}

	if err := os.WriteFile(destSidecar, data, 0644); err != nil {
		return err
	}
	if err := os.Remove(sidecarPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IsMP4FamilyExt reports whether the extension belongs to a container that the
// ffmpeg MP4/MOV muxer can write and that stores descriptive tags as
// `description`/`comment`. Other containers (e.g. MKV) are intentionally
// excluded because their tag key casing differs and remuxing an arbitrary codec
// into MP4 is unsafe.
func IsMP4FamilyExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".mp4", ".mov", ".m4v":
		return true
	default:
		return false
	}
}

// RemuxVideoWithMetadata stream-copies srcPath into destPath, optionally
// applying a clockwise rotation delta and embedding format-level metadata tags
// (e.g. description/comment). It performs a single ffmpeg pass for both, writes
// to a hidden temp file in destPath's directory (same filesystem, so the final
// rename is atomic), then renames it over destPath.
//
// srcPath is left untouched on any failure. The caller is responsible for
// removing srcPath after a successful return. onProgress (optional) receives
// byte deltas as the output grows.
func RemuxVideoWithMetadata(
	ctx context.Context,
	srcPath, destPath string,
	rotateAngle int,
	metadata map[string]string,
	onProgress func(delta int64),
) error {
	ctx, cancel := context.WithTimeout(ctx, remuxTimeout)
	defer cancel()

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0777); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	ext := filepath.Ext(destPath)
	tmpPath := filepath.Join(destDir, fmt.Sprintf(".embed-%d-%d.tmp%s", os.Getpid(), time.Now().UnixNano(), ext))
	defer func() {
		if _, err := os.Stat(tmpPath); err == nil {
			_ = os.Remove(tmpPath)
		}
	}()

	args := []string{"--as=524288000", "ffmpeg", "-y", "-loglevel", "error"}

	newAngle := 0
	if rotateAngle != 0 {
		current, err := ProbeVideoRotation(ctx, srcPath)
		if err != nil {
			return err
		}
		newAngle = AdjustRotationAngle(current, rotateAngle)
		logger.L.Debug("video remux rotation", "file", filepath.Base(srcPath), "current", current, "adjust_by", rotateAngle, "new_angle", newAngle)
		args = append(args, "-display_rotation", fmt.Sprintf("%d", newAngle))
	}

	args = append(args, "-i", srcPath, "-map", "0", "-c", "copy")

	if rotateAngle != 0 {
		args = append(args, "-metadata:s:v:0", fmt.Sprintf("rotate=%d", newAngle))
	}

	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-metadata", fmt.Sprintf("%s=%s", k, metadata[k]))
	}

	args = append(args, "-movflags", "+faststart", tmpPath)

	cmd := NewCommand(ctx, "prlimit", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	logger.L.Debug("running video remux", "input", filepath.Base(srcPath), "output", filepath.Base(destPath), "rotate", rotateAngle, "tags", keys)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var lastSize int64
	var waitErr error
	running := true
	for running {
		select {
		case waitErr = <-done:
			running = false
		case <-ticker.C:
			if onProgress == nil {
				continue
			}
			if fi, err := os.Stat(tmpPath); err == nil {
				if size := fi.Size(); size > lastSize {
					onProgress(size - lastSize)
					lastSize = size
				}
			}
		case <-ctx.Done():
			// CommandContext kills the process; wait for it to reap.
			waitErr = <-done
			running = false
		}
	}

	if waitErr != nil {
		return fmt.Errorf("ffmpeg remux failed: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to finalize remuxed file: %w", err)
	}
	return nil
}
