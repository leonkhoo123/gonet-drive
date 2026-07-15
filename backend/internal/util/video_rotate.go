package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-file-server/internal/logger"
)

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

	// Use a context with timeout to prevent runaway ffprobe/ffmpeg
	probeCtx, probeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer probeCancel()

	cmd := exec.CommandContext(probeCtx,
		"prlimit", "--as=524288000",
		"ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream_tags=rotate:stream_side_data=rotation",
		"-of", "default=noprint_wrappers=1:nokey=1", tempSrc)
	logger.L.Debug("running ffprobe", "input", tempSrc)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe error: %w", err)
	}

	ffprobeOutput := out.String()
	logger.L.Debug("ffprobe output", "output", ffprobeOutput, "file", filename)

	current := 0
	if s := strings.TrimSpace(ffprobeOutput); s != "" {
		if v, err := strconv.Atoi(strings.Split(s, "\n")[0]); err == nil {
			current = v
		}
	}

	// Calculate new angle (normalize to 0-360)
	newAngle := (current - rotateAngle) % 360
	// ffmpeg use counter clockwise rotation...
	if newAngle > 0 {
		newAngle -= 360
	}

	logger.L.Debug("video rotation calculation", "file", filename, "current", current, "adjust_by", rotateAngle, "new_angle", newAngle)

	// Apply new rotation using display_rotation (as INPUT option) and metadata
	ffmpegCtx, ffmpegCancel := context.WithTimeout(ctx, 120*time.Second)
	defer ffmpegCancel()

	tempOutput := filepath.Join(tempDir, fmt.Sprintf("%d_rotated_%s", time.Now().UnixNano(), filename))
	cmd = exec.CommandContext(ffmpegCtx,
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
