package util

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
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
