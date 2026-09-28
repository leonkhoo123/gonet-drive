package service

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go-file-server/internal/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeVideoEvents(t *testing.T) {
	got := normalizeVideoEvents([][]float64{
		{5, 2},   // reversed -> [2,5]
		{-1, 3},  // clamp start -> [0,3]
		{10, 20}, // clamp to duration -> [10,12]
		{4, 4},   // degenerate -> dropped
		{2, 5},   // duplicate of the first -> dropped
		{1},      // malformed -> dropped
	}, 12)

	assert.Equal(t, [][]float64{{0, 3}, {2, 5}, {10, 12}}, got)
}

func TestProcessVideoMetadataCommit_EmbedsAndWritesSidecar(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	makeServiceTestVideo(t, video)

	// Seed a detected sidecar so events_original has a distinct value to keep.
	metaDir := filepath.Join(dir, util.MetadataDirName)
	require.NoError(t, os.MkdirAll(metaDir, 0o755))
	require.NoError(t, os.WriteFile(
		util.SidecarPath(video),
		[]byte(`{"video":"clip.mp4","events":[[0.1,0.2]]}`),
		0o644,
	))

	tracker := util.NewProgressTracker()
	require.NoError(t, processVideoMetadataCommit(video, [][]float64{{0.0, 0.5}}, tracker))

	// In place: the video keeps its path and gains the embedded tag.
	require.FileExists(t, video)
	events, ok := readEmbeddedVideoEvents(video)
	require.True(t, ok)
	assert.Equal(t, [][]float64{{0, 0.5}}, events)

	var sc struct {
		Events         [][]float64 `json:"events"`
		EventsOriginal [][]float64 `json:"events_original"`
	}
	data, err := os.ReadFile(util.SidecarPath(video))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &sc))
	assert.Equal(t, [][]float64{{0, 0.5}}, sc.Events)
	assert.Equal(t, [][]float64{{0.1, 0.2}}, sc.EventsOriginal)

	// A second edit refreshes events but keeps the original detected spans.
	require.NoError(t, processVideoMetadataCommit(video, [][]float64{{0.2, 0.4}}, tracker))
	data, err = os.ReadFile(util.SidecarPath(video))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &sc))
	assert.Equal(t, [][]float64{{0.2, 0.4}}, sc.Events)
	assert.Equal(t, [][]float64{{0.1, 0.2}}, sc.EventsOriginal)

	// No remux temp files left behind.
	entries, err := os.ReadDir(filepath.Join(metaDir, "temp"))
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".embed-")
	}
}

func TestProcessVideoMetadataCommit_NonMP4SidecarOnly(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mkv")
	require.NoError(t, os.WriteFile(video, []byte("matroska-ish bytes"), 0o644))

	tracker := util.NewProgressTracker()
	require.NoError(t, processVideoMetadataCommit(video, [][]float64{{1, 2}}, tracker))

	// Non-MP4 containers are never embedded: the file is byte-identical and the
	// events land in the sidecar only.
	got, err := os.ReadFile(video)
	require.NoError(t, err)
	assert.Equal(t, []byte("matroska-ish bytes"), got)

	events, ok := readSidecarVideoEvents(video)
	require.True(t, ok)
	assert.Equal(t, [][]float64{{1, 2}}, events)
}
