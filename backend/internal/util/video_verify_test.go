package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamFingerprintMismatch(t *testing.T) {
	base := []verifyStream{
		{CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080},
		{CodecType: "audio", CodecName: "aac", SampleRate: "48000", Channels: 2},
	}
	assert.Empty(t, streamFingerprintMismatch(base, base))

	assert.Contains(t, streamFingerprintMismatch(base, base[:1]), "stream count changed")

	resized := []verifyStream{
		{CodecType: "video", CodecName: "h264", Width: 1280, Height: 720},
		{CodecType: "audio", CodecName: "aac", SampleRate: "48000", Channels: 2},
	}
	assert.Contains(t, streamFingerprintMismatch(base, resized), "size changed")

	recodec := []verifyStream{
		{CodecType: "video", CodecName: "hevc", Width: 1920, Height: 1080},
		{CodecType: "audio", CodecName: "aac", SampleRate: "48000", Channels: 2},
	}
	assert.Contains(t, streamFingerprintMismatch(base, recodec), "codec changed")

	audioChanged := []verifyStream{
		{CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080},
		{CodecType: "audio", CodecName: "aac", SampleRate: "44100", Channels: 1},
	}
	assert.Contains(t, streamFingerprintMismatch(base, audioChanged), "audio layout changed")
}

func TestMp4MoovPrecedesMdat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ffmpeg-dependent test in short mode")
	}
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	dir := t.TempDir()

	// makeTaggedMP4 writes with -movflags +faststart, so moov is front-loaded.
	fast := filepath.Join(dir, "fast.mp4")
	makeTaggedMP4(t, fast, "", "")
	checked, moovFirst, err := mp4MoovPrecedesMdat(fast)
	require.NoError(t, err)
	assert.True(t, checked)
	assert.True(t, moovFirst)

	// A non-MP4 blob has no boxes: the check is skipped, not failed.
	other := filepath.Join(dir, "blob.bin")
	require.NoError(t, os.WriteFile(other, []byte("not a video"), 0o644))
	checked, _, err = mp4MoovPrecedesMdat(other)
	require.NoError(t, err)
	assert.False(t, checked)
}

// Guard against a nil/empty stream list being reported as a match-plus-mismatch.
func TestStreamFingerprintMismatch_Empty(t *testing.T) {
	assert.Empty(t, streamFingerprintMismatch(nil, nil))
	assert.NotEmpty(t, streamFingerprintMismatch(nil, []verifyStream{{CodecType: "video"}}))
}
