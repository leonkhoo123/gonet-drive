package util

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// CloudReserveDirName is the hidden directory that holds all internal
	// application data (config, thumbnails, temp files, recycle bin, ...).
	CloudReserveDirName = ".cloud_reserve"
	// CloudDeleteDirName is the name of the recycle bin directory. It is stored
	// inside CloudReserveDirName but exposed to clients as "/.cloud_delete".
	CloudDeleteDirName = ".cloud_delete"
	// RecycleBinVirtualPath is the client-facing path of the recycle bin.
	RecycleBinVirtualPath = "/" + CloudDeleteDirName
)

// ErrPathOutsideRoot is returned when a path resolves, after following
// symbolic links, to a location outside the repository root.
var ErrPathOutsideRoot = errors.New("path resolves outside the allowed directory")

// containsNullByte reports whether s contains a NUL byte. NUL bytes are never
// valid in filesystem paths and can be used to smuggle a path past naive
// string checks, so they are rejected outright.
func containsNullByte(s string) bool {
	return strings.IndexByte(s, 0) >= 0
}

// RecycleBinPath returns the absolute on-disk path of the recycle bin directory
// for the given repository root. Physically it lives inside the cloud reserve
// directory; clients still address it as RecycleBinVirtualPath.
func RecycleBinPath(repoRoot string) string {
	return filepath.Join(repoRoot, CloudReserveDirName, CloudDeleteDirName)
}

// ToPhysicalPath maps a client-facing repository path to its on-disk location.
// The recycle bin is exposed as "/.cloud_delete" but stored under
// "/.cloud_reserve/.cloud_delete"; every other path is returned unchanged.
func ToPhysicalPath(relPath string) string {
	if relPath == RecycleBinVirtualPath {
		return "/" + CloudReserveDirName + RecycleBinVirtualPath
	}
	if strings.HasPrefix(relPath, RecycleBinVirtualPath+"/") {
		return "/" + CloudReserveDirName + relPath
	}
	return relPath
}

// ToVirtualPath is the inverse of ToPhysicalPath: it maps an on-disk path back
// to the client-facing path used by the API and UI.
func ToVirtualPath(relPath string) string {
	physical := "/" + CloudReserveDirName + RecycleBinVirtualPath
	if relPath == physical {
		return RecycleBinVirtualPath
	}
	if strings.HasPrefix(relPath, physical+"/") {
		return strings.TrimPrefix(relPath, "/"+CloudReserveDirName)
	}
	return relPath
}

// containsCloudReserveComponent reports whether any path component equals
// CloudReserveDirName. The reserve directory is an internal namespace holding
// config, thumbnails, temp uploads and the recycle bin. It must not be reachable
// through client-supplied paths; the recycle bin inside it is addressed through
// RecycleBinVirtualPath instead.
func containsCloudReserveComponent(relPath string) bool {
	for _, part := range strings.Split(relPath, "/") {
		if part == CloudReserveDirName {
			return true
		}
	}
	return false
}

// SanitizeRepoPaths resolves and validates a list of paths against a repository root.
// It ensures that none of the paths point to a location outside of the root,
// preventing directory traversal attacks. Client-facing paths are translated to
// their on-disk location first (see ToPhysicalPath), so "/.cloud_delete" resolves
// to "<root>/.cloud_reserve/.cloud_delete".
//
// Parameters:
//   - repoRoot: The absolute path to the root of the repository.
//   - paths: A slice of user-provided paths to sanitize.
//
// Returns:
//   - A slice of sanitized, absolute paths.
//   - An error if any path is invalid or falls outside the repository root.
func SanitizeRepoPaths(repoRoot string, paths []string) ([]string, error) {
	sanitizedPaths := make([]string, 0, len(paths))
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for repo root: %w", err)
	}

	for _, p := range paths {
		if containsNullByte(p) {
			return nil, fmt.Errorf("path contains a null byte")
		}
		if strings.Contains(p, "..") {
			return nil, fmt.Errorf("path contains '..', which is not allowed: %s", p)
		}
		if containsCloudReserveComponent(p) {
			return nil, fmt.Errorf("access to %s is forbidden: %s", CloudReserveDirName, p)
		}
		p = ToPhysicalPath(p)
		absPath, err := filepath.Abs(filepath.Join(absRepoRoot, p))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for '%s': %w", p, err)
		}

		rel, err := filepath.Rel(absRepoRoot, absPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("path is outside the allowed directory: %s", p)
		}

		sanitizedPaths = append(sanitizedPaths, absPath)
	}

	return sanitizedPaths, nil
}

// SanitizeRepoPath resolves and validates a single path against a repository root.
// It ensures that the path does not point to a location outside of the root,
// preventing directory traversal attacks. Client-facing paths are translated to
// their on-disk location first (see ToPhysicalPath), so "/.cloud_delete" resolves
// to "<root>/.cloud_reserve/.cloud_delete".
//
// Parameters:
//   - repoRoot: The absolute path to the root of the repository.
//   - path: A user-provided path to sanitize.
//
// Returns:
//   - The sanitized, absolute path.
//   - An error if the path is invalid or falls outside the repository root.
func SanitizeRepoPath(repoRoot string, path string) (string, error) {
	if containsNullByte(path) {
		return "", fmt.Errorf("path contains a null byte")
	}
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path contains '..', which is not allowed: %s", path)
	}
	if containsCloudReserveComponent(path) {
		return "", fmt.Errorf("access to %s is forbidden: %s", CloudReserveDirName, path)
	}
	path = ToPhysicalPath(path)
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for repo root: %w", err)
	}

	absPath, err := filepath.Abs(filepath.Join(absRepoRoot, path))
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for '%s': %w", path, err)
	}

	rel, err := filepath.Rel(absRepoRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside the allowed directory: %s", path)
	}

	return absPath, nil
}

// SanitizeRepoPathResolved is like SanitizeRepoPath but additionally resolves
// symbolic links and verifies that the *real* target still lives inside the
// repository root. The purely lexical SanitizeRepoPath cannot detect a symlink
// placed inside the root that points at, say, /etc or the internal
// .cloud_reserve namespace, so callers that read or write file contents should
// use this variant.
//
// The path must exist; a non-existent target returns an error wrapping
// os.ErrNotExist so callers can map it to a 404 response.
func SanitizeRepoPathResolved(repoRoot string, path string) (string, error) {
	absPath, err := SanitizeRepoPath(repoRoot, path)
	if err != nil {
		return "", err
	}

	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for repo root: %w", err)
	}

	realRoot, err := filepath.EvalSymlinks(absRepoRoot)
	if err != nil {
		return "", err
	}
	realTarget, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(realRoot, realTarget)
	if err != nil || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s", ErrPathOutsideRoot, path)
	}

	// A symlink must not be able to reach the internal reserve namespace, except
	// through the intended recycle-bin alias (".cloud_reserve/.cloud_delete").
	if containsCloudReserveComponent("/"+filepath.ToSlash(rel)) && !isRecycleBinRel(rel) {
		return "", fmt.Errorf("%w: %s", ErrPathOutsideRoot, path)
	}

	return realTarget, nil
}

// isRecycleBinRel reports whether rel (relative to the repo root) is the
// on-disk location of the recycle bin or something inside it.
func isRecycleBinRel(rel string) bool {
	rel = filepath.ToSlash(rel)
	prefix := CloudReserveDirName + "/" + CloudDeleteDirName
	return rel == prefix || strings.HasPrefix(rel, prefix+"/")
}

// SanitizeFilename ensures that the provided string is a safe filename.
// It uses filepath.Base to extract just the filename and prevents directory traversal.
func SanitizeFilename(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}

	baseName := filepath.Base(name)
	if baseName == "." || baseName == ".." || baseName == "/" || baseName == "\\" {
		return "", fmt.Errorf("invalid filename")
	}

	if baseName == ".cloud_delete" || baseName == ".cloud_reserve" {
		return "", fmt.Errorf("filename '%s' is reserved", baseName)
	}

	return baseName, nil
}

// IsSafePathComponent checks if a given string is safe to be used as a single path component (like a directory or file name).
// It prevents path traversal by rejecting paths containing directory separators, ".." sequences, or absolute paths.
func IsSafePathComponent(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if filepath.IsAbs(name) {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return false
	}
	return true
}

// GenerateOpID creates a unique operation ID.
func GenerateOpID() string {
	id, err := uuid.NewRandom()
	if err != nil {
		// Fallback to timestamp if UUID fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return id.String()
}

// TruncateString truncates a string to the specified max length, appending "..." if it was truncated.
func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		if maxLen > 3 {
			return string(runes[:maxLen-3]) + "..."
		}
		return string(runes[:maxLen])
	}
	return s
}
