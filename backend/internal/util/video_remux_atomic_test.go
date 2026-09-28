package util

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemuxVideoAtomic_InPlaceReplace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	makeTaggedMP4(t, video, "old-description", "")
	// Group-writable so we can assert the mode survives the in-place swap.
	require.NoError(t, os.Chmod(video, 0o664))

	tmpDir := filepath.Join(dir, MetadataDirName, "temp")
	payload := `{"video":"clip.mp4","events":[[1,2]],"scenes":[{"start":1,"end":2}]}`
	metadata := map[string]string{"description": payload, "comment": payload}

	err := RemuxVideoAtomic(context.Background(), video, video, tmpDir, 0, metadata, nil)
	require.NoError(t, err)

	require.FileExists(t, video, "in-place replace must keep the original path")
	tags, err := ReadMP4Tags(video)
	require.NoError(t, err)
	assert.Equal(t, payload, tags["description"])
	assert.Equal(t, payload, tags["comment"])

	// Temp lives in tmpDir (hidden) and is cleaned up.
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".embed-", "temp file should be cleaned up")
	}

	info, err := os.Stat(video)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o664), info.Mode().Perm(), "source mode must be preserved")
}

func TestRemuxVideoAtomic_FailureKeepsOriginal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "broken.mp4")
	require.NoError(t, os.WriteFile(video, []byte("not a video"), 0o644))

	tmpDir := filepath.Join(dir, MetadataDirName, "temp")
	err := RemuxVideoAtomic(context.Background(), video, video, tmpDir, 0,
		map[string]string{"description": "x", "comment": "x"}, nil)
	assert.Error(t, err)

	// The original bytes must be untouched.
	got, readErr := os.ReadFile(video)
	require.NoError(t, readErr)
	assert.Equal(t, []byte("not a video"), got)
}

func TestVerifyVideoOutput_RejectsEmptyAndNonVideo(t *testing.T) {
	dir := t.TempDir()

	empty := filepath.Join(dir, "empty.mp4")
	require.NoError(t, os.WriteFile(empty, nil, 0o644))
	assert.Error(t, VerifyVideoOutput(empty, empty, nil))

	notVideo := filepath.Join(dir, "nope.mp4")
	require.NoError(t, os.WriteFile(notVideo, []byte("not a video"), 0o644))
	assert.Error(t, VerifyVideoOutput(notVideo, notVideo, nil))
}
