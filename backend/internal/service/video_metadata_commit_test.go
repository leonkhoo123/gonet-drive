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

type sidecarEvents struct {
	Events         [][]float64 `json:"events"`
	EventsOriginal [][]float64 `json:"events_original"`
}

func readSidecarEvents(t *testing.T, video string) sidecarEvents {
	t.Helper()
	data, err := os.ReadFile(util.SidecarPath(video))
	require.NoError(t, err)
	var sc sidecarEvents
	require.NoError(t, json.Unmarshal(data, &sc))
	return sc
}

// makeServiceTestVideoTagged writes a one-frame MP4 that already carries an
// embedded `description` tag, so the commit takes the embed path.
func makeServiceTestVideoTagged(t *testing.T, path, description string) {
	t.Helper()
	out, err := exec.Command("ffmpeg",
		"-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=red:s=160x120:d=1",
		"-frames:v", "1", "-c:v", "libx264", "-preset", "ultrafast",
		"-metadata", "description="+description,
		"-movflags", "+faststart",
		path,
	).CombinedOutput()
	require.NoError(t, err, "failed to create test video: %s", string(out))
}

func TestProcessVideoMetadataCommit_UntaggedMP4StaysSidecarOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	makeServiceTestVideo(t, video) // no embedded tag
	before, err := os.ReadFile(video)
	require.NoError(t, err)

	// Seed a detected sidecar so events_original has a distinct value to keep.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, util.MetadataDirName), 0o755))
	require.NoError(t, os.WriteFile(
		util.SidecarPath(video),
		[]byte(`{"video":"clip.mp4","events":[[0.1,0.2]]}`),
		0o644,
	))

	tracker := util.NewProgressTracker()
	require.NoError(t, processVideoMetadataCommit(video, [][]float64{{0.0, 0.5}}, tracker))

	// Rule B: a file with no embedded tag is never remuxed, so its bytes are
	// unchanged and nothing gets embedded.
	require.FileExists(t, video)
	after, err := os.ReadFile(video)
	require.NoError(t, err)
	assert.Equal(t, before, after, "sidecar-only commit must not touch the video")
	_, embedded := readEmbeddedVideoEvents(video)
	assert.False(t, embedded)

	sc := readSidecarEvents(t, video)
	assert.Equal(t, [][]float64{{0, 0.5}}, sc.Events)
	assert.Equal(t, [][]float64{{0.1, 0.2}}, sc.EventsOriginal)

	// A second edit refreshes events but keeps the original detected spans.
	require.NoError(t, processVideoMetadataCommit(video, [][]float64{{0.2, 0.4}}, tracker))
	sc = readSidecarEvents(t, video)
	assert.Equal(t, [][]float64{{0.2, 0.4}}, sc.Events)
	assert.Equal(t, [][]float64{{0.1, 0.2}}, sc.EventsOriginal)
}

func TestProcessVideoMetadataCommit_EmbedsWhenAlreadyEmbedded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	makeServiceTestVideoTagged(t, video, `{"video":"clip.mp4","events":[[0.05,0.08]]}`)

	require.NoError(t, processVideoMetadataCommit(video, [][]float64{{0.0, 0.5}}, util.NewProgressTracker()))

	// The embedded tag is rewritten with the new events...
	events, ok := readEmbeddedVideoEvents(video)
	require.True(t, ok)
	assert.Equal(t, [][]float64{{0, 0.5}}, events)

	// ...and the sidecar is kept in sync, preserving the detected originals.
	sc := readSidecarEvents(t, video)
	assert.Equal(t, [][]float64{{0, 0.5}}, sc.Events)
	assert.Equal(t, [][]float64{{0.05, 0.08}}, sc.EventsOriginal)

	// No remux temp files left behind.
	entries, err := os.ReadDir(filepath.Join(dir, util.MetadataDirName, "temp"))
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
