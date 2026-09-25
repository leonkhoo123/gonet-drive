package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeRepoPath_Valid(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "foo", "bar.txt")
	os.MkdirAll(filepath.Dir(filePath), 0755)
	os.WriteFile(filePath, []byte("test"), 0644)

	result, err := SanitizeRepoPath(root, "foo/bar.txt")
	assert.NoError(t, err)
	assert.Equal(t, filePath, result)
}

func TestSanitizeRepoPath_ParentTraversal(t *testing.T) {
	root := t.TempDir()

	_, err := SanitizeRepoPath(root, "../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "..")

	_, err = SanitizeRepoPath(root, "./..")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "..")

	_, err = SanitizeRepoPath(root, "foo/../../bar")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "..")
}

func TestSanitizeRepoPath_AbsoluteWithinRoot(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "sub", "file.txt")

	result, err := SanitizeRepoPath(root, absPath)
	assert.NoError(t, err)
	assert.Contains(t, result, root)
}

func TestSanitizeRepoPath_AbsoluteOutsideRoot(t *testing.T) {
	root := t.TempDir()

	_, err := SanitizeRepoPath(root, "/etc/passwd")
	assert.NoError(t, err)
}

func TestSanitizeRepoPaths_BatchValid(t *testing.T) {
	root := t.TempDir()
	p1 := filepath.Join(root, "a.txt")
	p2 := filepath.Join(root, "b.txt")
	os.WriteFile(p1, []byte("a"), 0644)
	os.WriteFile(p2, []byte("b"), 0644)

	results, err := SanitizeRepoPaths(root, []string{"a.txt", "b.txt"})
	assert.NoError(t, err)
	assert.Equal(t, []string{p1, p2}, results)
}

func TestSanitizeRepoPaths_BatchWithTraversal(t *testing.T) {
	root := t.TempDir()

	_, err := SanitizeRepoPaths(root, []string{"safe.txt", "../etc/passwd"})
	assert.Error(t, err)
}

func TestSanitizeRepoPaths_EmptySlice(t *testing.T) {
	root := t.TempDir()
	results, err := SanitizeRepoPaths(root, []string{})
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestToPhysicalPath_RecycleBin(t *testing.T) {
	assert.Equal(t, "/.cloud_reserve/.cloud_delete", ToPhysicalPath("/.cloud_delete"))
	assert.Equal(t, "/.cloud_reserve/.cloud_delete/a/b.txt", ToPhysicalPath("/.cloud_delete/a/b.txt"))
	// Unrelated paths are untouched.
	assert.Equal(t, "/photos/a.jpg", ToPhysicalPath("/photos/a.jpg"))
	assert.Equal(t, "/.cloud_reserve/logo.png", ToPhysicalPath("/.cloud_reserve/logo.png"))
}

func TestToVirtualPath_RecycleBin(t *testing.T) {
	assert.Equal(t, "/.cloud_delete", ToVirtualPath("/.cloud_reserve/.cloud_delete"))
	assert.Equal(t, "/.cloud_delete/a/b.txt", ToVirtualPath("/.cloud_reserve/.cloud_delete/a/b.txt"))
	// Unrelated paths are untouched.
	assert.Equal(t, "/photos/a.jpg", ToVirtualPath("/photos/a.jpg"))
	assert.Equal(t, "/.cloud_reserve", ToVirtualPath("/.cloud_reserve"))
}

func TestRecycleBinPath(t *testing.T) {
	root := t.TempDir()
	assert.Equal(t, filepath.Join(root, ".cloud_reserve", ".cloud_delete"), RecycleBinPath(root))
}

func TestSanitizeRepoPath_RecycleBinMapping(t *testing.T) {
	root := t.TempDir()

	result, err := SanitizeRepoPath(root, "/.cloud_delete")
	assert.NoError(t, err)
	assert.Equal(t, RecycleBinPath(root), result)

	result, err = SanitizeRepoPath(root, "/.cloud_delete/deleted.txt")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(RecycleBinPath(root), "deleted.txt"), result)
}

func TestSanitizeRepoPaths_RecycleBinMapping(t *testing.T) {
	root := t.TempDir()

	results, err := SanitizeRepoPaths(root, []string{"/a.txt", "/.cloud_delete/b.txt"})
	assert.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(root, "a.txt"),
		filepath.Join(RecycleBinPath(root), "b.txt"),
	}, results)
}

func TestSanitizeRepoPath_RecycleBinTraversal(t *testing.T) {
	root := t.TempDir()

	payloads := []string{
		"/.cloud_delete/../../etc/passwd",
		"/.cloud_delete/../.cloud_reserve/config.db",
		"/.cloud_delete/sub/../../outside.txt",
	}
	for _, p := range payloads {
		_, err := SanitizeRepoPath(root, p)
		assert.Error(t, err, "path %q must be rejected", p)
		assert.Contains(t, err.Error(), "..")
	}
}

func TestSanitizeRepoPaths_RecycleBinTraversal(t *testing.T) {
	root := t.TempDir()

	_, err := SanitizeRepoPaths(root, []string{"/.cloud_delete/../../etc/passwd"})
	assert.Error(t, err)
}

func TestSanitizeRepoPath_ReserveBlocked(t *testing.T) {
	root := t.TempDir()

	payloads := []string{
		"/.cloud_reserve",
		"/.cloud_reserve/logo.png",
		"/.cloud_reserve/config/db",
		"/.cloud_reserve/.thumbnails/x.webp",
		"/.cloud_reserve/.cloud_delete/leaked.txt",
		"sub/.cloud_reserve/file",
	}
	for _, p := range payloads {
		_, err := SanitizeRepoPath(root, p)
		assert.Error(t, err, "path %q must be rejected", p)
		assert.Contains(t, err.Error(), "forbidden")
	}
}

func TestSanitizeRepoPaths_ReserveBlocked(t *testing.T) {
	root := t.TempDir()

	_, err := SanitizeRepoPaths(root, []string{"/a.txt", "/.cloud_reserve/logo.png"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestSanitizeRepoPath_ReserveSubstringAllowed(t *testing.T) {
	root := t.TempDir()

	// A name that merely contains the reserved string is not a path component.
	result, err := SanitizeRepoPath(root, "/notes.cloud_reserve.txt")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "notes.cloud_reserve.txt"), result)

	result, err = SanitizeRepoPath(root, "/.cloud_deleteish/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(root, ".cloud_deleteish", "file.txt"), result)
}

func TestSanitizeFilename_Valid(t *testing.T) {
	result, err := SanitizeFilename("myfile.txt")
	assert.NoError(t, err)
	assert.Equal(t, "myfile.txt", result)
}

func TestSanitizeFilename_Empty(t *testing.T) {
	_, err := SanitizeFilename("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filename cannot be empty")
}

func TestSanitizeFilename_Dots(t *testing.T) {
	_, err := SanitizeFilename(".")
	assert.Error(t, err)

	_, err = SanitizeFilename("..")
	assert.Error(t, err)

	_, err = SanitizeFilename("/")
	assert.Error(t, err)
}

func TestSanitizeFilename_ReservedNames(t *testing.T) {
	_, err := SanitizeFilename(".cloud_delete")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reserved")

	_, err = SanitizeFilename(".cloud_reserve")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reserved")
}

func TestSanitizeFilename_PathTraversal(t *testing.T) {
	result, err := SanitizeFilename("foo/bar")
	assert.NoError(t, err)
	assert.Equal(t, "bar", result)

	result, err = SanitizeFilename("foo/../bar")
	assert.NoError(t, err)
	assert.Equal(t, "bar", result)
}

func TestIsSafePathComponent_Valid(t *testing.T) {
	assert.True(t, IsSafePathComponent("foo"))
	assert.True(t, IsSafePathComponent("foo-bar"))
	assert.True(t, IsSafePathComponent("foo.bar"))
}

func TestIsSafePathComponent_Invalid(t *testing.T) {
	assert.False(t, IsSafePathComponent(""))
	assert.False(t, IsSafePathComponent("."))
	assert.False(t, IsSafePathComponent(".."))
	assert.False(t, IsSafePathComponent("/foo"))
	assert.False(t, IsSafePathComponent("foo/bar"))
	assert.False(t, IsSafePathComponent("foo\\bar"))
}

func TestGenerateOpID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateOpID()
		assert.NotEmpty(t, id)
		assert.False(t, ids[id], "duplicate ID: %s", id)
		ids[id] = true
	}
}

func TestTruncateString_Short(t *testing.T) {
	result := TruncateString("hello", 10)
	assert.Equal(t, "hello", result)
}

func TestTruncateString_Long(t *testing.T) {
	result := TruncateString("hello world", 8)
	assert.Equal(t, "hello...", result)
}

func TestTruncateString_Tiny(t *testing.T) {
	result := TruncateString("hello world", 2)
	assert.Equal(t, "he", result)
}
