package util

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemuxVideoWithMetadata_EmbedsTagsAndKeepsSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "src.mp4")
	dst := filepath.Join(dir, "done", "renamed.mp4")
	makeTaggedMP4(t, src, "", "")

	payload := `{"video":"renamed.mp4","events":[[1,2]],"scenes":[{"start":1,"end":2}]}`
	metadata := map[string]string{"description": payload, "comment": payload}

	err := RemuxVideoWithMetadata(context.Background(), src, dst, 0, metadata, nil)
	require.NoError(t, err)

	// Source is left for the caller to delete; destination carries the tags.
	_, err = os.Stat(src)
	assert.NoError(t, err, "source must remain untouched on success")

	tags, err := ReadMP4Tags(dst)
	require.NoError(t, err)
	assert.Equal(t, payload, tags["description"])
	assert.Equal(t, payload, tags["comment"])
}

func TestRemuxVideoWithMetadata_InvalidSourceLeavesDestinationAbsent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "broken.mp4")
	require.NoError(t, os.WriteFile(src, []byte("not a video"), 0644))
	dst := filepath.Join(dir, "out.mp4")

	err := RemuxVideoWithMetadata(context.Background(), src, dst, 0, map[string]string{"description": "x"}, nil)
	assert.Error(t, err)

	_, statErr := os.Stat(dst)
	assert.True(t, os.IsNotExist(statErr), "destination must not exist after a failed remux")
	// No temp files should be left behind.
	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".embed-", "temp file should be cleaned up")
	}
}
