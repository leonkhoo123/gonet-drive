package util

import (
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
