package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-file-server/internal/logger"
)

// ProbeVideoRotation returns the current display rotation (0/90/180/270) that
// ffmpeg would apply to the video stream, read from either the stream tag or
// the side data rotation matrix. A missing value is reported as 0.
func ProbeVideoRotation(ctx context.Context, path string) (int, error) {
	probeCtx, probeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer probeCancel()

	cmd := NewCommand(probeCtx,
		"prlimit", "--as=524288000",
		"ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream_tags=rotate:stream_side_data=rotation",
		"-of", "default=noprint_wrappers=1:nokey=1", path)

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe error: %w", err)
	}

	if s := strings.TrimSpace(out.String()); s != "" {
		if v, err := strconv.Atoi(strings.Split(s, "\n")[0]); err == nil {
			return v, nil
		}
	}
	return 0, nil
}

// AdjustRotationAngle mirrors the ffmpeg display_rotation convention: it applies
// a clockwise `rotateAngle` on top of the current angle and normalises the
// result into the (-360, 0] range used by the -display_rotation input option.
func AdjustRotationAngle(current, rotateAngle int) int {
	newAngle := (current - rotateAngle) % 360
	// ffmpeg uses counter clockwise rotation...
	if newAngle > 0 {
		newAngle -= 360
	}
	return newAngle
}

func AdjustVideoRotationTemp(ctx context.Context, fileRoot, srcPath string, rotateAngle int) (string, error) {
	tempDir := filepath.Join(fileRoot, ".cloud_reserve", "tmp_rotate")

	// Ensure temp folder
	if err := os.MkdirAll(tempDir, 0777); err != nil {
		return "", fmt.Errorf("failed to create temp folder: %w", err)
	}

	// Move original video into temp
	// Use a unique filename to avoid collisions when multiple files are rotated
	filename := filepath.Base(srcPath)
	tempSrc := filepath.Join(tempDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename))
	if err := os.Rename(srcPath, tempSrc); err != nil {
		return "", fmt.Errorf("failed to move source to temp: %w", err)
	}

	// On failure, move the original file back to prevent data loss
	restoreOnFail := true
	defer func() {
		if restoreOnFail {
			if err := os.Rename(tempSrc, srcPath); err != nil {
				logger.L.Error("failed to restore original video after rotation failure", "from", tempSrc, "to", srcPath, "err", err)
			}
		}
	}()

	current, err := ProbeVideoRotation(ctx, tempSrc)
	if err != nil {
		return "", err
	}

	// Calculate new angle (normalize to 0-360)
	newAngle := AdjustRotationAngle(current, rotateAngle)

	logger.L.Debug("video rotation calculation", "file", filename, "current", current, "adjust_by", rotateAngle, "new_angle", newAngle)

	// Apply new rotation using display_rotation (as INPUT option) and metadata
	ffmpegCtx, ffmpegCancel := context.WithTimeout(ctx, 120*time.Second)
	defer ffmpegCancel()

	tempOutput := filepath.Join(tempDir, fmt.Sprintf("%d_rotated_%s", time.Now().UnixNano(), filename))
	cmd := NewCommand(ffmpegCtx,
		"prlimit", "--as=524288000",
		"ffmpeg",
		"-display_rotation", fmt.Sprintf("%d", newAngle),
		"-i", tempSrc,
		"-c", "copy",
		"-metadata:s:v:0", fmt.Sprintf("rotate=%d", newAngle),
		tempOutput)
	logger.L.Debug("running ffmpeg for rotation", "new_angle", newAngle)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		ffmpegError := errBuf.String()
		logger.L.Error("ffmpeg failed", "stderr", ffmpegError, "file", filename)
		os.Remove(tempOutput)
		return "", fmt.Errorf("ffmpeg error: %v: %s", err, ffmpegError)
	}

	// Success — prevent deferred restore
	restoreOnFail = false

	// Cleanup temp source file after successful rotation
	defer func() {
		if err := os.Remove(tempSrc); err != nil {
			logger.L.Warn("failed to remove temp source file", "path", tempSrc, "err", err)
		}
	}()

	logger.L.Info("video rotation applied", "output", tempOutput, "file", filename)

	return tempOutput, nil
}
