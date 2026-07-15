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
