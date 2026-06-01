package util

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdjustVideoRotationTemp_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := t.TempDir()
	fileRoot := filepath.Join(workDir, "data")
	require.NoError(t, os.MkdirAll(fileRoot, 0755))

	// Create a test video that is NOT rotated (rotation tag = 0)
	videoPath := filepath.Join(fileRoot, "test_no_rotation.mp4")
	makeTestMP4(t, videoPath, "")

	originalData, err := os.ReadFile(videoPath)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Rotate by 90 degrees (counter-clockwise)
	outputPath, err := AdjustVideoRotationTemp(ctx, fileRoot, videoPath, 90)
	require.NoError(t, err)
	require.NotEmpty(t, outputPath)

	// Verify output exists
	_, err = os.Stat(outputPath)
	require.NoError(t, err, "rotated output should exist at %s", outputPath)

	// Verify original was moved to temp and cleaned up
	_, err = os.Stat(videoPath)
	assert.True(t, os.IsNotExist(err), "original file should have been moved")

	// Verify temp source is cleaned up (we can't check exact path, but temp dir should be clean of source files)
	// The main verification is that the rotation succeeded without error

	_ = originalData
}

func TestAdjustVideoRotationTemp_DataRecoveryOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := t.TempDir()
	fileRoot := filepath.Join(workDir, "data")
	require.NoError(t, os.MkdirAll(fileRoot, 0755))

	// Create a non-video file to force ffprobe to fail
	nonVideoPath := filepath.Join(fileRoot, "not_a_video.mp4")
	require.NoError(t, os.WriteFile(nonVideoPath, []byte("this is not a video file"), 0644))
	originalData, err := os.ReadFile(nonVideoPath)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should fail because the file is not a valid video
	_, err = AdjustVideoRotationTemp(ctx, fileRoot, nonVideoPath, 90)
	assert.Error(t, err, "should fail for non-video file")

	// Verify the original file was restored to its original location
	restoredData, err := os.ReadFile(nonVideoPath)
	require.NoError(t, err, "original file should have been restored at %s", nonVideoPath)
	assert.Equal(t, originalData, restoredData, "restored file should match original content")
}

func TestAdjustVideoRotationTemp_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := t.TempDir()
	fileRoot := filepath.Join(workDir, "data")
	require.NoError(t, os.MkdirAll(fileRoot, 0755))

	videoPath := filepath.Join(fileRoot, "test_cancel.mp4")
	makeTestMP4(t, videoPath, "90") // pre-rotated to force ffmpeg re-processing

	originalData, err := os.ReadFile(videoPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = AdjustVideoRotationTemp(ctx, fileRoot, videoPath, 90)
	assert.Error(t, err, "should fail with cancelled context")

	// Original file should be restored
	restoredData, err := os.ReadFile(videoPath)
	require.NoError(t, err, "original file should have been restored")
	assert.Equal(t, originalData, restoredData)
}

func makeTestMP4(t *testing.T, path string, rotation string) {
	t.Helper()

	args := []string{
		"-f", "lavfi",
		"-i", "color=c=blue:s=320x240:d=1",
		"-frames:v", "1",
		"-c:v", "libx264",
		"-preset", "ultrafast",
	}

	if rotation != "" {
		args = append(args, "-metadata:s:v:0", "rotate="+rotation)
	}

	args = append(args, "-y", path)

	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to create test video: %s", string(out))
}
