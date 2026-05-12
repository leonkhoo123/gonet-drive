package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"video.mp4", true},
		{"movie.mkv", true},
		{"clip.mov", true},
		{"old.avi", true},
		{"stream.webm", true},
		{"image.jpg", false},
		{"song.mp3", false},
		{"document.pdf", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, isVideoFile(tt.name), "isVideoFile(%q)", tt.name)
	}
}

func TestIsPhotoFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"icon.png", true},
		{"anim.gif", true},
		{"img.bmp", true},
		{"img.webp", true},
		{"vector.svg", true},
		{"heic_image.heic", true},
		{"video.mp4", false},
		{"text.txt", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, isPhotoFile(tt.name), "isPhotoFile(%q)", tt.name)
	}
}

func TestIsMusicFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"song.mp3", true},
		{"lossless.flac", true},
		{"recording.wav", true},
		{"audio.ogg", true},
		{"podcast.m4a", true},
		{"stream.aac", true},
		{"old.wma", true},
		{"video.mp4", false},
		{"document.pdf", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, isMusicFile(tt.name), "isMusicFile(%q)", tt.name)
	}
}

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"readme.txt", true},
		{"notes.md", true},
		{"config.json", true},
		{"data.yaml", true},
		{"data.csv", true},
		{"main.go", true},
		{"script.sh", true},
		{"Dockerfile", true},
		{"Makefile", true},
		{"Caddyfile", true},
		{"image.png", false},
		{"video.mp4", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, isTextFile(tt.name), "isTextFile(%q)", tt.name)
	}
}

func TestIsPdfFile(t *testing.T) {
	assert.True(t, isPdfFile("document.pdf"))
	assert.True(t, isPdfFile("report.PDF"))
	assert.False(t, isPdfFile("image.png"))
	assert.False(t, isPdfFile("video.mp4"))
	assert.False(t, isPdfFile(""))
}
