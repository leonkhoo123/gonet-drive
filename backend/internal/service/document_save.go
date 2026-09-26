package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/util"
)

// MaxTextSaveBytes is a generous hard ceiling for a single text-save request.
// It is intentionally NOT the viewer's preview limit (2 MB): editing may grow
// a file past what the viewer can preview. This constant only bounds memory
// usage and rejects absurd payloads.
const MaxTextSaveBytes = 50 * 1024 * 1024

var (
	// ErrSavePathRequired means no path was supplied.
	ErrSavePathRequired = errors.New("path is required")
	// ErrSaveForbidden means the path failed sanitization (traversal, reserved dirs, ...).
	ErrSaveForbidden = errors.New("path is not allowed")
	// ErrSaveNotFound means the target does not exist.
	ErrSaveNotFound = errors.New("file not found")
	// ErrSaveNotRegular means the target exists but is not a regular file.
	ErrSaveNotRegular = errors.New("target is not a regular file")
	// ErrSaveNotText means the target is not a recognised text file, so the
	// text editor must not overwrite it (e.g. a binary or media file).
	ErrSaveNotText = errors.New("target is not a text file")
	// ErrSaveContentMissing means the content field was absent from the request.
	// It is deliberately distinct from an intentional empty string.
	ErrSaveContentMissing = errors.New("content field is required")
	// ErrSaveEmpty guards against accidentally wiping a file with an empty payload.
	ErrSaveEmpty = errors.New("refusing to save empty content")
	// ErrSaveTooLarge means the payload exceeds MaxTextSaveBytes.
	ErrSaveTooLarge = errors.New("content exceeds maximum size")
	// ErrSaveConflict means the on-disk file changed since the client loaded it.
	ErrSaveConflict = errors.New("file changed on disk")
)

// SaveTextReq is the JSON body for POST /api/user/files/save-text.
type SaveTextReq struct {
	Path         string  `json:"path"`
	Content      *string `json:"content"`
	AllowEmpty   bool    `json:"allowEmpty"`
	BaseSize     int64   `json:"baseSize"`
	BaseModified string  `json:"baseModified"`
}

// SaveTextResp reports the new file state so the client can refresh its baseline.
type SaveTextResp struct {
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// SaveTextDocument overwrites an existing text file with the supplied content.
//
// The write is atomic (temp file + rename), preserves the original file mode,
// and refuses to clobber a file that changed on disk since the client loaded it
// (see BaseSize / BaseModified).
func SaveTextDocument(req SaveTextReq, cfg *config.CloudConfig) (SaveTextResp, error) {
	if req.Path == "" {
		return SaveTextResp{}, ErrSavePathRequired
	}
	// A missing content field must never be treated as "empty content".
	if req.Content == nil {
		return SaveTextResp{}, ErrSaveContentMissing
	}

	content := *req.Content
	if content == "" && !req.AllowEmpty {
		return SaveTextResp{}, ErrSaveEmpty
	}
	if len(content) > MaxTextSaveBytes {
		return SaveTextResp{}, ErrSaveTooLarge
	}

	fullPath, err := util.SanitizeRepoPathResolved(cfg.Server.FileRoot, req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return SaveTextResp{}, ErrSaveNotFound
		}
		return SaveTextResp{}, fmt.Errorf("%w: %v", ErrSaveForbidden, err)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return SaveTextResp{}, ErrSaveNotFound
		}
		return SaveTextResp{}, err
	}
	if info.IsDir() || !info.Mode().IsRegular() {
		return SaveTextResp{}, ErrSaveNotRegular
	}
	if !isTextFile(filepath.Base(fullPath)) {
		return SaveTextResp{}, ErrSaveNotText
	}

	if err := checkSaveConflict(req, info); err != nil {
		return SaveTextResp{}, err
	}

	if err := atomicWriteFile(fullPath, []byte(content), info.Mode().Perm()); err != nil {
		return SaveTextResp{}, err
	}

	newInfo, err := os.Stat(fullPath)
	if err != nil {
		return SaveTextResp{}, err
	}

	return SaveTextResp{
		Size:     newInfo.Size(),
		Modified: newInfo.ModTime().Format(time.RFC3339Nano),
	}, nil
}

// SaveConflictError is returned when the on-disk file changed since the client
// loaded it. It carries the current file state so the client can refresh its
// baseline before retrying.
type SaveConflictError struct {
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

func (e *SaveConflictError) Error() string { return ErrSaveConflict.Error() }

// Is allows errors.Is(err, ErrSaveConflict) to match.
func (e *SaveConflictError) Is(target error) bool { return target == ErrSaveConflict }

// checkSaveConflict compares the client's baseline against the current file.
// The baseline is opt-in: when BaseModified is empty the client did not supply
// one, so no conflict check is performed. When present, both size and mtime
// must match. A malformed BaseModified is treated as no baseline.
func checkSaveConflict(req SaveTextReq, info os.FileInfo) error {
	if req.BaseModified == "" {
		return nil
	}
	baseTime, err := time.Parse(time.RFC3339Nano, req.BaseModified)
	if err != nil {
		return nil
	}
	if req.BaseSize != info.Size() || !baseTime.Equal(info.ModTime()) {
		return &SaveConflictError{
			Size:     info.Size(),
			Modified: info.ModTime().Format(time.RFC3339Nano),
		}
	}
	return nil
}

// atomicWriteFile writes data to a temp file in the same directory and renames
// it over path, so a crash mid-write can never leave a truncated file.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".~gonet-edit-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	cleanup = false
	return nil
}
