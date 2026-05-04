package util

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SanitizeRepoPaths resolves and validates a list of paths against a repository root.
// It ensures that none of the paths point to a location outside of the root,
// preventing directory traversal attacks.
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
// preventing directory traversal attacks.
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

	if baseName == ".cloud_delete" {
		return "", fmt.Errorf("filename '.cloud_delete' is reserved")
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
