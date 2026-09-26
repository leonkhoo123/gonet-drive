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

func makeServiceTestVideo(t *testing.T, path string) {
	t.Helper()
	out, err := exec.Command("ffmpeg",
		"-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=red:s=160x120:d=1",
		"-frames:v", "1", "-c:v", "libx264", "-preset", "ultrafast",
		path,
	).CombinedOutput()
	require.NoError(t, err, "failed to create test video: %s", string(out))
}

func TestParseEventsJSON(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect [][]float64
		ok     bool
	}{
		{
			name:   "detector events",
			input:  `{"video":"clip.mp4","events":[[14.0,29.0],[30.0,45.8]]}`,
			expect: [][]float64{{14, 29}, {30, 45.8}},
			ok:     true,
		},
		{
			name:   "compact e key",
			input:  `{"e":[[5,6]]}`,
			expect: [][]float64{{5, 6}},
			ok:     true,
		},
		{
			name:   "scenes objects",
			input:  `{"scenes":[{"start":9,"end":3}]}`,
			expect: [][]float64{{3, 9}},
			ok:     true,
		},
		{
			name:   "bare array sorted",
			input:  `[[30,40],[10,20]]`,
			expect: [][]float64{{10, 20}, {30, 40}},
			ok:     true,
		},
		{
			name:   "single object",
			input:  `{"start":1,"end":2}`,
			expect: [][]float64{{1, 2}},
			ok:     true,
		},
		{
			name:  "invalid json",
			input: `{`,
			ok:    false,
		},
		{
			name:  "empty events",
			input: `{"events":[]}`,
			ok:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseEventsJSON([]byte(tc.input))
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.expect, got)
			}
		})
	}
}

func TestBuildEmbedPayload(t *testing.T) {
	dir := t.TempDir()
	sidecar := filepath.Join(dir, "clip.mp4_timestamps.json")
	require.NoError(t, os.WriteFile(sidecar, []byte(`{"video":"clip.mp4","events":[[2.0,5.0]]}`), 0644))

	payload, ok := buildEmbedPayload(sidecar, "renamed.mp4")
	require.True(t, ok)
	assert.JSONEq(t, `{
		"video":"renamed.mp4",
		"events":[[2.0,5.0]],
		"scenes":[{"start":2.0,"end":5.0}]
	}`, payload)

	_, ok = buildEmbedPayload(filepath.Join(dir, "missing.json"), "x.mp4")
	assert.False(t, ok)
}

func TestReadVideoEvents_FallsBackToSidecar(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	require.NoError(t, os.WriteFile(video, []byte("not a real container"), 0644))

	metaDir := filepath.Join(dir, util.MetadataDirName)
	require.NoError(t, os.MkdirAll(metaDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(metaDir, "clip.mp4_timestamps.json"),
		[]byte(`{"video":"clip.mp4","events":[[14.0,29.0]]}`),
		0644,
	))

	events, ok := readSidecarVideoEvents(video)
	require.True(t, ok)
	assert.Equal(t, [][]float64{{14, 29}}, events)

	// Embedded read short-circuits on the non-container file.
	_, ok = readEmbeddedVideoEvents(video)
	assert.False(t, ok)
}

func TestProcessVideoRenameDone_EmbedsAndDeletesSidecar(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "clip.mp4")
	makeServiceTestVideo(t, src)

	metaDir := filepath.Join(dir, util.MetadataDirName)
	require.NoError(t, os.MkdirAll(metaDir, 0755))
	sidecar := filepath.Join(metaDir, "clip.mp4_timestamps.json")
	require.NoError(t, os.WriteFile(sidecar, []byte(`{"video":"clip.mp4","events":[[2.0,5.0]]}`), 0644))

	procPath, doneDir, sidecarPath := stageVideo(t, dir, "clip.mp4")

	tracker := util.NewProgressTracker()
	require.NoError(t, processVideoRenameDone(procPath, doneDir, sidecarPath, "final.mp4", 0, tracker))

	dest := filepath.Join(dir, "done", "final.mp4")
	require.FileExists(t, dest)
	assert.NoFileExists(t, src, "source should already be gone from the browse folder")
	assert.NoFileExists(t, procPath, "staging file should be removed after success")
	assert.NoFileExists(t, sidecar, "sidecar should be deleted after a successful embed")

	events, ok := readEmbeddedVideoEvents(dest)
	require.True(t, ok)
	assert.Equal(t, [][]float64{{2, 5}}, events)
}

func TestProcessVideoRenameDone_NoSidecarJustMoves(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "clip.mp4")
	// No ffmpeg needed: with no metadata and no rotation this is a plain rename.
	require.NoError(t, os.WriteFile(src, []byte("plain bytes"), 0644))

	procPath, doneDir, sidecarPath := stageVideo(t, dir, "clip.mp4")

	tracker := util.NewProgressTracker()
	require.NoError(t, processVideoRenameDone(procPath, doneDir, sidecarPath, "moved.mp4", 0, tracker))

	dest := filepath.Join(dir, "done", "moved.mp4")
	require.FileExists(t, dest)
	assert.NoFileExists(t, src)
	assert.NoFileExists(t, procPath)
}

func TestProcessVideoRenameDone_EmbedFailureLeavesInTmp(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "broken.mp4")
	// An invalid container forces the remux to fail: the original must stay in
	// done/tmp for recovery, with its sidecar brought alongside it.
	require.NoError(t, os.WriteFile(src, []byte("not a video"), 0644))

	metaDir := filepath.Join(dir, util.MetadataDirName)
	require.NoError(t, os.MkdirAll(metaDir, 0755))
	sidecar := filepath.Join(metaDir, "broken.mp4_timestamps.json")
	require.NoError(t, os.WriteFile(sidecar, []byte(`{"events":[[1,2]]}`), 0644))

	procPath, doneDir, sidecarPath := stageVideo(t, dir, "broken.mp4")

	tracker := util.NewProgressTracker()
	err := processVideoRenameDone(procPath, doneDir, sidecarPath, "renamed.mp4", 0, tracker)
	require.Error(t, err)

	// The original stays put; nothing lands in done/.
	require.FileExists(t, procPath, "failed file must remain in done/tmp")
	assert.NoFileExists(t, filepath.Join(dir, "done", "renamed.mp4"))

	// Its sidecar is relocated next to it, keyed to the staged file name.
	relocated := filepath.Join(doneDir, VideoProcessingDirName, util.MetadataDirName, "broken.mp4_timestamps.json")
	require.FileExists(t, relocated)
	assert.NoFileExists(t, sidecar)
	assert.Equal(t, "broken.mp4", readSidecarVideoField(t, relocated))

	events, ok := readSidecarVideoEvents(procPath)
	require.True(t, ok)
	assert.Equal(t, [][]float64{{1, 2}}, events)
}

func TestProcessVideoRenameDone_NonMP4RelocatesSidecar(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "clip.mkv")
	require.NoError(t, os.WriteFile(src, []byte("matroska-ish bytes"), 0644))

	metaDir := filepath.Join(dir, util.MetadataDirName)
	require.NoError(t, os.MkdirAll(metaDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(metaDir, "clip.mkv_timestamps.json"), []byte(`{"events":[[3,4]]}`), 0644))

	procPath, doneDir, sidecarPath := stageVideo(t, dir, "clip.mkv")

	tracker := util.NewProgressTracker()
	require.NoError(t, processVideoRenameDone(procPath, doneDir, sidecarPath, "clip.mkv", 0, tracker))

	dest := filepath.Join(dir, "done", "clip.mkv")
	require.FileExists(t, dest)
	assert.NoFileExists(t, procPath)

	// Non-MP4 containers are never embedded, so the sidecar travels along.
	relocated := filepath.Join(dir, "done", util.MetadataDirName, "clip.mkv_timestamps.json")
	require.FileExists(t, relocated)
	assert.NoFileExists(t, filepath.Join(metaDir, "clip.mkv_timestamps.json"))
	assert.Equal(t, "clip.mkv", readSidecarVideoField(t, relocated))
}

// stageVideo mirrors StartVideoRenameDone's first step: atomically move the
// source video into done/tmp and return the staging path, done dir and the
// sidecar path keyed to the original location.
func stageVideo(t *testing.T, dir, name string) (procPath, doneDir, sidecarPath string) {
	t.Helper()
	src := filepath.Join(dir, name)
	doneDir = filepath.Join(dir, "done")
	procDir := filepath.Join(doneDir, VideoProcessingDirName)
	require.NoError(t, os.MkdirAll(procDir, 0777))
	procPath = filepath.Join(procDir, name)
	require.NoError(t, os.Rename(src, procPath))
	return procPath, doneDir, util.SidecarPath(src)
}

// readSidecarVideoField returns the `video` field of a sidecar JSON file.
func readSidecarVideoField(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var obj struct {
		Video string `json:"video"`
	}
	require.NoError(t, json.Unmarshal(data, &obj))
	return obj.Video
}
