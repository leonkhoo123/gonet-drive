package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-file-server/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveTestConfig(t *testing.T) *config.CloudConfig {
	t.Helper()
	return &config.CloudConfig{
		Server: config.ServerConfig{FileRoot: t.TempDir()},
	}
}

func writeSaveTestFile(t *testing.T, cfg *config.CloudConfig, rel, content string, perm os.FileMode) string {
	t.Helper()
	full := filepath.Join(cfg.Server.FileRoot, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), perm))
	return full
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func strPtr(s string) *string { return &s }

func TestSaveTextDocument_Success(t *testing.T) {
	cfg := saveTestConfig(t)
	full := writeSaveTestFile(t, cfg, "notes.txt", "hello", 0o644)

	info, err := os.Stat(full)
	require.NoError(t, err)

	res, err := SaveTextDocument(SaveTextReq{
		Path:         "/notes.txt",
		Content:      strPtr("hello world"),
		BaseSize:     info.Size(),
		BaseModified: info.ModTime().Format(time.RFC3339Nano),
	}, cfg)
	require.NoError(t, err)

	assert.Equal(t, int64(len("hello world")), res.Size)
	assert.Equal(t, "hello world", readFileString(t, full))
	assert.NotEmpty(t, res.Modified)

	// A second save must work using the baseline returned by the first.
	res2, err := SaveTextDocument(SaveTextReq{
		Path:         "/notes.txt",
		Content:      strPtr("second"),
		BaseSize:     res.Size,
		BaseModified: res.Modified,
	}, cfg)
	require.NoError(t, err)
	assert.Equal(t, int64(len("second")), res2.Size)
	assert.Equal(t, "second", readFileString(t, full))
}

func TestSaveTextDocument_PreservesModeAndCleansTemp(t *testing.T) {
	cfg := saveTestConfig(t)
	full := writeSaveTestFile(t, cfg, "secret.txt", "data", 0o600)

	info, err := os.Stat(full)
	require.NoError(t, err)

	_, err = SaveTextDocument(SaveTextReq{
		Path:     "/secret.txt",
		Content:  strPtr("new data"),
		BaseSize: info.Size(),
	}, cfg)
	require.NoError(t, err)

	newInfo, err := os.Stat(full)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), newInfo.Mode().Perm())

	// No leftover temp files from the atomic write.
	entries, err := os.ReadDir(filepath.Dir(full))
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".~gonet-edit-")
	}
}

func TestSaveTextDocument_MissingContentField(t *testing.T) {
	cfg := saveTestConfig(t)
	full := writeSaveTestFile(t, cfg, "a.txt", "keep me", 0o644)

	_, err := SaveTextDocument(SaveTextReq{Path: "/a.txt"}, cfg)
	assert.ErrorIs(t, err, ErrSaveContentMissing)
	assert.Equal(t, "keep me", readFileString(t, full))
}

func TestSaveTextDocument_EmptyBlockedUnlessAllowed(t *testing.T) {
	cfg := saveTestConfig(t)
	full := writeSaveTestFile(t, cfg, "a.txt", "keep me", 0o644)

	_, err := SaveTextDocument(SaveTextReq{Path: "/a.txt", Content: strPtr("")}, cfg)
	assert.ErrorIs(t, err, ErrSaveEmpty)
	assert.Equal(t, "keep me", readFileString(t, full))

	res, err := SaveTextDocument(SaveTextReq{Path: "/a.txt", Content: strPtr(""), AllowEmpty: true}, cfg)
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Size)
	assert.Equal(t, "", readFileString(t, full))
}

func TestSaveTextDocument_TooLarge(t *testing.T) {
	cfg := saveTestConfig(t)
	full := writeSaveTestFile(t, cfg, "a.txt", "small", 0o644)

	_, err := SaveTextDocument(SaveTextReq{
		Path:    "/a.txt",
		Content: strPtr(strings.Repeat("x", MaxTextSaveBytes+1)),
	}, cfg)
	assert.ErrorIs(t, err, ErrSaveTooLarge)
	assert.Equal(t, "small", readFileString(t, full))
}

func TestSaveTextDocument_PathRequiredAndForbidden(t *testing.T) {
	cfg := saveTestConfig(t)

	_, err := SaveTextDocument(SaveTextReq{Content: strPtr("x")}, cfg)
	assert.ErrorIs(t, err, ErrSavePathRequired)

	_, err = SaveTextDocument(SaveTextReq{Path: "../evil.txt", Content: strPtr("x")}, cfg)
	assert.ErrorIs(t, err, ErrSaveForbidden)

	_, err = SaveTextDocument(SaveTextReq{Path: "/.cloud_reserve/evil.txt", Content: strPtr("x")}, cfg)
	assert.ErrorIs(t, err, ErrSaveForbidden)
}

func TestSaveTextDocument_NotFoundAndNotRegular(t *testing.T) {
	cfg := saveTestConfig(t)

	_, err := SaveTextDocument(SaveTextReq{Path: "/nope.txt", Content: strPtr("x")}, cfg)
	assert.ErrorIs(t, err, ErrSaveNotFound)

	require.NoError(t, os.MkdirAll(filepath.Join(cfg.Server.FileRoot, "adir"), 0o755))
	_, err = SaveTextDocument(SaveTextReq{Path: "/adir", Content: strPtr("x")}, cfg)
	assert.ErrorIs(t, err, ErrSaveNotRegular)
}

func TestSaveTextDocument_Conflict(t *testing.T) {
	cfg := saveTestConfig(t)
	full := writeSaveTestFile(t, cfg, "a.txt", "orig", 0o644)

	info, err := os.Stat(full)
	require.NoError(t, err)
	baseModified := info.ModTime().Format(time.RFC3339Nano)

	// Wrong size (with a valid baseline).
	_, err = SaveTextDocument(SaveTextReq{
		Path:         "/a.txt",
		Content:      strPtr("new"),
		BaseSize:     info.Size() + 1,
		BaseModified: baseModified,
	}, cfg)
	assert.ErrorIs(t, err, ErrSaveConflict)
	assert.Equal(t, "orig", readFileString(t, full))

	// Right size, stale modified time.
	_, err = SaveTextDocument(SaveTextReq{
		Path:         "/a.txt",
		Content:      strPtr("new"),
		BaseSize:     info.Size(),
		BaseModified: time.Now().Add(-time.Hour).Format(time.RFC3339Nano),
	}, cfg)
	assert.ErrorIs(t, err, ErrSaveConflict)
	assert.Equal(t, "orig", readFileString(t, full))

	// The conflict error carries the current on-disk state for the client.
	var conflict *SaveConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Equal(t, info.Size(), conflict.Size)
	assert.NotEmpty(t, conflict.Modified)

	// No baseline supplied is tolerated (conflict check is opt-in).
	_, err = SaveTextDocument(SaveTextReq{
		Path:    "/a.txt",
		Content: strPtr("new"),
	}, cfg)
	require.NoError(t, err)
	assert.Equal(t, "new", readFileString(t, full))
}

func TestSaveTextDocument_SymlinkEscapeRejected(t *testing.T) {
	cfg := saveTestConfig(t)
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("secret"), 0644))
	require.NoError(t, os.Symlink(secret, filepath.Join(cfg.Server.FileRoot, "escape.txt")))

	_, err := SaveTextDocument(SaveTextReq{Path: "/escape.txt", Content: strPtr("pwned")}, cfg)
	assert.ErrorIs(t, err, ErrSaveForbidden)

	// The file the symlink points at must be untouched.
	assert.Equal(t, "secret", readFileString(t, secret))
}

func TestSaveTextDocument_InternalSymlinkWritesTarget(t *testing.T) {
	cfg := saveTestConfig(t)
	target := writeSaveTestFile(t, cfg, "real.txt", "orig", 0644)
	require.NoError(t, os.Symlink(target, filepath.Join(cfg.Server.FileRoot, "link.txt")))

	_, err := SaveTextDocument(SaveTextReq{Path: "/link.txt", Content: strPtr("updated")}, cfg)
	require.NoError(t, err)
	assert.Equal(t, "updated", readFileString(t, target))
}

func TestSaveTextDocument_NullBytePathRejected(t *testing.T) {
	cfg := saveTestConfig(t)

	_, err := SaveTextDocument(SaveTextReq{Path: "/evil\x00.txt", Content: strPtr("x")}, cfg)
	assert.ErrorIs(t, err, ErrSaveForbidden)
}

func TestSaveTextDocument_NonTextTargetRejected(t *testing.T) {
	cfg := saveTestConfig(t)
	full := writeSaveTestFile(t, cfg, "movie.mp4", "not really a video", 0o644)

	_, err := SaveTextDocument(SaveTextReq{Path: "/movie.mp4", Content: strPtr("text")}, cfg)
	assert.ErrorIs(t, err, ErrSaveNotText)
	assert.Equal(t, "not really a video", readFileString(t, full))
}
