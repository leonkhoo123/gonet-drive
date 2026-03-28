package util

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func AdjustVideoRotationTemp(fileRoot, srcPath string, rotateAngle int) (string, error) {
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

	// Get current rotation from both metadata and display matrix
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream_tags=rotate:stream_side_data=rotation",
		"-of", "default=noprint_wrappers=1:nokey=1", tempSrc)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe error: %w", err)
	}

	ffprobeOutput := out.String()
	log.Printf("ffprobe output: [%s]", ffprobeOutput)

	current := 0
	if s := strings.TrimSpace(ffprobeOutput); s != "" {
		// Try to parse the first number found
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

	log.Printf("Video [%s]: Current=%d°, Adjusting=-%d°, New=%d°", filename, current, rotateAngle, newAngle)

	// Apply new rotation using display_rotation (as INPUT option) and metadata
	tempOutput := filepath.Join(tempDir, fmt.Sprintf("%d_rotated_%s", time.Now().UnixNano(), filename))
	cmd = exec.Command("ffmpeg",
		"-display_rotation", fmt.Sprintf("%d", newAngle),
		"-i", tempSrc,
		"-c", "copy",
		"-metadata:s:v:0", fmt.Sprintf("rotate=%d", newAngle),
		tempOutput)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		ffmpegError := errBuf.String()
		log.Printf("ffmpeg stderr: %s", ffmpegError)
		os.Remove(tempOutput)
		return "", fmt.Errorf("ffmpeg error: %v: %s", err, ffmpegError)
	}

	// Defer cleanup of temp source file
	defer func() {
		if err := os.Remove(tempSrc); err != nil {
			log.Printf("Warning: failed to remove temp source file %s: %v", tempSrc, err)
		}
	}()

	log.Printf("Rotation applied successfully. Output: %s", tempOutput)

	return tempOutput, nil
}
