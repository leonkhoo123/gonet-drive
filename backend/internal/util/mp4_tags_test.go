package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func makeTaggedMP4(t *testing.T, path, description, comment string) {
	t.Helper()
	args := []string{
		"-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=blue:s=320x240:d=1",
		"-frames:v", "1", "-c:v", "libx264", "-preset", "ultrafast",
	}
	if description != "" {
		args = append(args, "-metadata", "description="+description)
	}
	if comment != "" {
		args = append(args, "-metadata", "comment="+comment)
	}
	args = append(args, "-movflags", "+faststart", path)

	out, err := exec.Command("ffmpeg", args...).CombinedOutput()
	require.NoError(t, err, "failed to create test video: %s", string(out))
}

func TestReadMP4Tags_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	path := filepath.Join(t.TempDir(), "tagged.mp4")
	desc := `{"video":"clip.mp4","events":[[14.0,29.0],[30.0,45.8]],"scenes":[{"start":14,"end":29}]}`
	comment := `café & "quoted" <tag> 柚子`
	makeTaggedMP4(t, path, desc, comment)

	tags, err := ReadMP4Tags(path)
	require.NoError(t, err)
	assert.Equal(t, desc, tags["description"])
	assert.Equal(t, comment, tags["comment"])

	got, ok := ReadMP4Tag(path, "description")
	assert.True(t, ok)
	assert.Equal(t, desc, got)
}

func TestReadMP4Tags_NoTags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	path := filepath.Join(t.TempDir(), "clean.mp4")
	makeTaggedMP4(t, path, "", "")

	tags, err := ReadMP4Tags(path)
	require.NoError(t, err)
	assert.Empty(t, tags["description"])
	assert.Empty(t, tags["comment"])
}

func TestReadMP4Tags_MoovAtEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	path := filepath.Join(t.TempDir(), "no_faststart.mp4")
	out, err := exec.Command("ffmpeg",
		"-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=green:s=160x120:d=1",
		"-frames:v", "1", "-c:v", "libx264", "-preset", "ultrafast",
		"-metadata", "description=tail-moov-value",
		path,
	).CombinedOutput()
	require.NoError(t, err, "ffmpeg: %s", string(out))

	tags, err := ReadMP4Tags(path)
	require.NoError(t, err)
	assert.Equal(t, "tail-moov-value", tags["description"])
}

func TestReadMP4Tags_MissingFile(t *testing.T) {
	_, err := ReadMP4Tags(filepath.Join(t.TempDir(), "nope.mp4"))
	assert.Error(t, err)
}

func TestReadMP4Tags_NotAVideo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not_a_video.mp4")
	require.NoError(t, os.WriteFile(path, []byte("this is not an mp4"), 0644))

	tags, err := ReadMP4Tags(path)
	// A non-MP4 file yields no boxes; the parser should not error or hang.
	require.NoError(t, err)
	assert.Empty(t, tags)
}
