# 02 — File Migration (22 files, level mapping)

> Parent: [logger_plan_index.md](logger_plan_index.md) | Status: none

---

## Migration Rules

1. `import "log"` → remove, add `import "go-file-server/internal/logger"`
2. `log.Println(...)` / `log.Printf(...)` → `logger.L.Level("msg", keyvals...)`
3. `log.Fatalf(...)` → `logger.L.Fatal(...)`
4. `fmt.Printf(...)` (warnings/errors) → `logger.L.Warn(...)` or `logger.L.Error(...)`
5. Message text: convert format-string style to structured key=value pairs
6. All `Printf` → appropriate level based on context (see level classification in [00-architecture.md](00-architecture.md))

---

## File-by-File Mapping

### 1. `cmd/main.go` (6 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 125 | `log.Fatalf("failed to create sub filesystem for ui dist: %v", err)` | Fatal | `logger.L.Fatal("failed to create sub filesystem", "err", err)` |
| 154 | `log.Printf("Starting server on %s", ...)` | Info | `logger.L.Info("starting server", "addr", cfg.Server.ListenAddr)` |
| 156 | `log.Fatalf("failed to start server: %v", err)` | Fatal | `logger.L.Fatal("server start failed", "err", err)` |
| 164 | `log.Println("Shutting down...")` | Info | `logger.L.Info("shutting down")` |
| 169 | `log.Fatalf("Server forced to shutdown...")` | Fatal | `logger.L.Fatal("forced shutdown", "err", err)` |
| 172 | `log.Println("Server exiting gracefully")` | Info | `logger.L.Info("server exited gracefully")` |

Additional: Add import for `"go-file-server/internal/logger"` and `"log/slog"` for `parseLogLevel`.

---

### 2. `internal/config/config.go` (16 calls incl. fmt.Printf)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 63 | `fmt.Println("⚠️ No .env file found...")` | Warn | `logger.L.Warn("no .env file found, using built-in defaults")` |
| 110 | `log.Println("--- Starting application with configuration ---")` | Info | `logger.L.Info("starting application with configuration")` |
| 111 | `log.Printf("FileRoot: %s", c.Server.FileRoot)` | Info | `logger.L.Info("config loaded", "file_root", c.Server.FileRoot, "listen_addr", c.Server.ListenAddr, "log_level", c.Server.LogLevel)` |
| 112 | `log.Printf("ListenAddr: %s", ...)` | — | *merged with above as structured fields* |
| 113 | `log.Println("---...---")` | — | *removed (redundant with structured log)* |
| 116 | `log.Fatalf("Missing JWT secret .env")` | Fatal | `logger.L.Fatal("missing JWT secret")` |
| 120 | `log.Fatalf("Working directory does not exist (%s)", ...)` | Fatal | `logger.L.Fatal("working directory does not exist", "path", c.Server.FileRoot)` |
| 135 | `log.Fatalf("Failed to create %s directory: %v", ...)` | Fatal | `logger.L.Fatal("failed to create directory", "path", cloudReserveDir, "err", err)` |
| 152 | `log.Printf("Failed to create %s directory: %v", ...)` | Error | `logger.L.Error("failed to create directory", "path", iconDir, "err", err)` |
| 159 | `log.Printf("Failed to write default logo...")` | Warn | `logger.L.Warn("failed to write default logo", "path", logoPath, "err", err)` |
| 161 | `log.Printf("Initialized default logo at %s", ...)` | Info | `logger.L.Info("initialized default logo", "path", logoPath)` |
| 184 | `fmt.Printf("⚠️ Invalid duration for %s...")` | Warn | `logger.L.Warn("invalid duration config", "key", key, "value", v, "default", fallback)` |
| 196 | `fmt.Printf("⚠️ Invalid integer for %s...")` | Warn | `logger.L.Warn("invalid integer config", "key", key, "value", v, "default", fallback)` |

Changes: Remove `import "log"`, `import "fmt"` (if fmt only used for removed calls), add `import "go-file-server/internal/logger"`.

---

### 3. `internal/config/db_config.go` (14 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 52 | `log.Fatalf("Failed to create %s directory: %v", ...)` | Fatal | `logger.L.Fatal("failed to create DB directory", "path", configDir, "err", err)` |
| 62 | `log.Fatalf("Failed to connect to database: %v", err)` | Fatal | `logger.L.Fatal("failed to connect to database", "err", err)` |
| 67 | `log.Fatalf("Failed to run database migrations: %v", err)` | Fatal | `logger.L.Fatal("database migrations failed", "err", err)` |
| 86 | `log.Printf("Failed to insert default config (attempt %d/5): %v", ...)` | Warn | `logger.L.Warn("failed to insert default config", "attempt", i+1, "max_attempts", 5, "err", err)` |
| 90 | `log.Fatalf("Failed to insert default config after retries: %v", err)` | Fatal | `logger.L.Fatal("failed to insert default config after retries", "err", err)` |
| 95 | `log.Printf("Warning: Failed to load cloud config cache: %v", err)` | Warn | `logger.L.Warn("failed to load cloud config cache", "err", err)` |
| 121 | `log.Printf("Failed to check if admin user exists: %v", err)` | Error | `logger.L.Error("failed to check admin user existence", "err", err)` |
| 128 | `log.Printf("Failed to hash admin password: %v", err)` | Error | `logger.L.Error("failed to hash admin password", "err", err)` |
| 138 | `log.Printf("Failed to create admin user: %v", err)` | Error | `logger.L.Error("failed to create admin user", "err", err)` |
| 140 | `log.Printf("Successfully bootstrapped superadmin user: %s", ...)` | Info | `logger.L.Info("bootstrapped superadmin user", "username", adminUser)` |

Changes: Remove `import "log"`, add `import "go-file-server/internal/logger"`.

---

### 4. `internal/service/file_queue.go` (1 call)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 16 | `log.Println("Starting sequential file operation worker...")` | Info | `logger.L.Info("starting sequential file operation worker")` |

---

### 5. `internal/service/file_operation.go` (11 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 147 | `log.Printf("Error in %s operation %s: %v", opType, opID, err)` | Error | `logger.L.Error("file operation failed", "opType", opType, "opId", opID, "err", err)` |
| 177 | `log.Printf("Received cancel request for OpID: %s", req.OpID)` | Debug | `logger.L.Debug("received cancel request", "opId", req.OpID)` |
| 188 | `log.Printf("[OpID: %s] CopyFiles: sources=%v, destDir=%s", ...)` | Debug | `logger.L.Debug("copy files requested", "opId", req.OpID, "sources", req.Sources, "destDir", req.DestDir)` |
| 255 | `log.Printf("[OpID: %s] MoveFiles: sources=%v, destDir=%s", ...)` | Debug | `logger.L.Debug("move files requested", "opId", req.OpID, "sources", req.Sources, "destDir", req.DestDir)` |
| 304 | `log.Printf("[OpID: %s] DeleteFiles: sources=%v", ...)` | Debug | `logger.L.Debug("delete files requested", "opId", req.OpID, "sources", req.Sources)` |
| 334 | `log.Printf("Skipping deletion of protected folder: %s", source)` | Warn | `logger.L.Warn("skipping protected folder", "path", source)` |
| 359 | `log.Printf("[OpID: %s] DeletePermanentFiles: sources=%v", ...)` | Debug | `logger.L.Debug("permanent delete requested", "opId", req.OpID, "sources", req.Sources)` |
| 389 | `log.Printf("Skipping permanent deletion of protected folder: %s", ...)` | Warn | `logger.L.Warn("skipping protected folder", "path", source)` |
| 412 | `log.Printf("[OpID: %s] RenameFile: source=%s, newName=%s", ...)` | Debug | `logger.L.Debug("rename file requested", "opId", req.OpID, "source", req.Source, "newName", req.NewName)` |
| 436 | `log.Printf("[OpID: %s] CreateFolder: dir=%s, folderName=%s", ...)` | Debug | `logger.L.Debug("create folder requested", "opId", req.OpID, "dir", req.Dir, "folderName", req.FolderName)` |

**New debug logs to add:**

| Location | New Log | Purpose |
|----------|---------|---------|
| After line 154 (success branch) | `logger.L.Debug("file operation completed", "opId", opID, "opType", opType, "files", tracker.CopiedFiles, "bytes", tracker.CopiedBytes)` | Debug: operation completion summary |

---

### 6. `internal/service/share_file_service.go` (11 calls)

Same pattern as `file_operation.go` — [OpID:...] → Debug, errors → Error, protected folder → Warn.

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 84 | `log.Printf("Error in share %s operation %s: %v", ...)` | Error | `logger.L.Error("share file operation failed", "opType", opType, "opId", opID, "err", err)` |
| 113 | `log.Printf("[OpID: %s] ShareCopyFiles: ...")` | Debug | `logger.L.Debug("share copy files requested", "opId", req.OpID, "sources", req.Sources, "destDir", req.DestDir)` |
| 180 | `log.Printf("[OpID: %s] ShareMoveFiles: ...")` | Debug | `logger.L.Debug("share move files requested", "opId", req.OpID, "sources", req.Sources, "destDir", req.DestDir)` |
| 229 | `log.Printf("[OpID: %s] ShareDeleteFiles: ...")` | Debug | `logger.L.Debug("share delete files requested", "opId", req.OpID, "sources", req.Sources)` |
| 259 | `log.Printf("Skipping deletion of protected folder in share context: %s", ...)` | Warn | `logger.L.Warn("skipping protected folder in share context", "path", source)` |
| 284 | `log.Printf("[OpID: %s] ShareDeletePermanentFiles: ...")` | Debug | `logger.L.Debug("share permanent delete requested", "opId", req.OpID, "sources", req.Sources)` |
| 314 | `log.Printf("Skipping permanent deletion of protected folder in share context: %s", ...)` | Warn | `logger.L.Warn("skipping protected folder in share context", "path", source)` |
| 337 | `log.Printf("[OpID: %s] ShareRenameFile: ...")` | Debug | `logger.L.Debug("share rename file requested", "opId", req.OpID, "source", req.Source, "newName", req.NewName)` |
| 361 | `log.Printf("[OpID: %s] ShareCreateFolder: ...")` | Debug | `logger.L.Debug("share create folder requested", "opId", req.OpID, "dir", req.Dir, "folderName", req.FolderName)` |
| 705 | `fmt.Printf("Error zipping %s: %v\n", fullPath, err)` | Error | `logger.L.Error("share zip download failed", "path", fullPath, "err", err)` |

Changes: Remove `import "log"`, add `import "go-file-server/internal/logger"`. (`import "fmt"` stays — still used by `fmt.Sprintf`/`fmt.Errorf`.)

---

### 7. `internal/service/sharing_manage_service.go` (4 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 114 | `log.Printf("Failed to insert share link: %v", err)` | Error | `logger.L.Error("failed to insert share link", "err", err)` |
| 145 | `log.Printf("Failed to query shares: %v", err)` | Error | `logger.L.Error("failed to query shares", "err", err)` |
| 159 | `log.Printf("Failed to stat share path: %s, error: %v", ...)` | Warn | `logger.L.Warn("failed to stat share path", "path", fullPath, "err", statErr)` |
| 162 | `log.Printf("Failed to sanitize share path: %s, error: %v", ...)` | Warn | `logger.L.Warn("failed to sanitize share path", "path", shares[i].Path, "err", err)` |

---

### 8. `internal/service/video_rename_done.go` (2 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 40 | `log.Printf("[OpID: %s] VideoRenameDone: path=%s, newName=%s, angle=%d", ...)` | Debug | `logger.L.Debug("video rename requested", "opId", req.OpID, "path", req.Path, "newName", req.NewName, "angle", req.RotateAngle)` |
| 64 | `log.Printf("Will add [%d°] to [%s]", ...)` | Debug | `logger.L.Debug("applying video rotation", "angle", req.RotateAngle, "file", newName)` |

---

### 9. `internal/service/video_disqualified.go` (1 call)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 39 | `log.Printf("[OpID: %s] VideoDisqualified: path=%s", ...)` | Debug | `logger.L.Debug("video disqualified requested", "opId", req.OpID, "path", req.Path)` |

---

### 10. `internal/service/video_compress_serve.go` (1 call)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 65 | `fmt.Printf("Stream error: %v\n", err)` | Error | `logger.L.Error("video stream error", "err", err)` |

Changes: Add `import "go-file-server/internal/logger"`. (`import "fmt"` stays — still used by `fmt.Sprintf`.)

**New debug log to add:**

| Location | New Log | Purpose |
|----------|---------|---------|
| Before line 29 (building target URL) | `logger.L.Debug("proxying video stream", "file", fullPath, "start", start, "target", targetURL)` | Debug: track proxy to transcoder |

---

### 11. `internal/service/thumbnail_generation.go` (1 call)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 68 | `fmt.Printf("thumbnail upsert failed for %s: %v\n", ...)` | Warn | `logger.L.Warn("thumbnail upsert failed", "file", filePath, "err", err)` |

Changes: Add `import "go-file-server/internal/logger"`. (`import "fmt"` stays — still used by `fmt.Errorf`.)

**New debug logs to add:**

| Location | New Log | Purpose |
|----------|---------|---------|
| After line 22 (GenerateVideoThumbnail) | `logger.L.Debug("generating video thumbnail", "input", fullPath, "output", thumbPath)` | Debug: ffmpeg thumbnail command context |
| After line 44 (GeneratePhotoThumbnail) | `logger.L.Debug("generating photo thumbnail", "input", fullPath, "output", thumbPath)` | Debug: photo thumbnail |

---

### 12. `internal/service/file_download.go` (1 call)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 168 | `fmt.Printf("Error zipping %s: %v\n", fullPath, err)` | Error | `logger.L.Error("zip download failed", "path", fullPath, "err", err)` |

Changes: Add `import "go-file-server/internal/logger"`. (`import "fmt"` stays — still used by `fmt.Sprintf`.)

**New debug log to add:**

| Location | New Log | Purpose |
|----------|---------|---------|
| Before zip creation (after line 82) | `logger.L.Debug("creating zip download", "count", len(validPaths), "zipName", zipFileName)` | Debug: download context |

---

### 13. `internal/service/file_upload.go` (0 calls — add new debug logs)

Currently NO logging. Add:

| Location | New Log | Level | Purpose |
|----------|---------|-------|---------|
| After line 75 (upload start) | `logger.L.Debug("upload chunk", "identifier", identifier, "status", status, "filename", filename, "chunk", chunkNumber, "total", totalChunks)` | Debug | Chunk upload progress |
| After line 201 (checksum verified) | `logger.L.Debug("chunk checksum verified", "identifier", identifier, "chunk", chunkNumber)` | Debug | Checksum pass |
| After line 260 (upload complete) | `logger.L.Info("upload complete", "identifier", identifier, "file", cleanFilename, "dest", destPath, "size", finalSize)` | Info | Upload completion summary |

---

### 14. `internal/util/file_copy.go` (3 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 58 | `log.Println("Calculating total size...")` | Debug | `logger.L.Debug("calculating total size", "opId", shortID)` |
| 73 | `log.Printf("Starting copy operation: %s across %d files", ...)` | Info | `logger.L.Info("starting copy operation", "size", FormatBytes(tracker.TotalBytes), "files", tracker.TotalFiles)` |
| 123 | `log.Printf("Copy failed halfway. Cleaning up...")` | Warn | `logger.L.Warn("copy failed halfway, cleaning up copied items")` |

**New debug logs to add:**

| Location | New Log | Purpose |
|----------|---------|---------|
| After temp dir creation (line 54) | `logger.L.Debug("created copy temp dir", "path", tempDir)` | Debug: temp dir |

---

### 15. `internal/util/file_move.go` (5 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 44 | `log.Println("Counting files to move...")` | Debug | `logger.L.Debug("counting files to move")` |
| 53 | `log.Printf("Starting move operation: %d files to move", ...)` | Info | `logger.L.Info("starting move operation", "files", tracker.TotalFiles)` |
| 65 | `log.Printf("Move failed halfway. Reverting %d...")` | Warn | `logger.L.Warn("move failed halfway, reverting", "items", len(movedItems))` |
| 71 | `log.Printf("Failed to revert move from %s to %s: %v", ...)` | Warn | `logger.L.Warn("failed to revert move", "source", item.src, "dest", item.dst, "err", err)` |
| 263 | `log.Printf("Warning: failed to remove source directory '%s': %v", ...)` | Warn | `logger.L.Warn("failed to remove source directory after merge", "path", src, "err", err)` |

---

### 16. `internal/util/file_delete.go` (4 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 13 | `log.Println("Counting files to permanently delete...")` | Debug | `logger.L.Debug("counting files to permanently delete")` |
| 19 | `log.Printf("Warning: failed to count files for '%s': %v", ...)` | Warn | `logger.L.Warn("failed to count files for deletion", "path", source, "err", err)` |
| 23 | `log.Printf("Starting permanent delete operation: %d files...")` | Info | `logger.L.Info("starting permanent delete operation", "files", tracker.TotalFiles)` |
| 34 | `log.Println(finalErr)` | Error | `logger.L.Error("failed to permanently delete", "path", source, "err", err)` |

---

### 17. `internal/util/progress.go` (4 calls — all to Debug)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 118 | `log.Printf("Progress: %.2f%% ...")` | Debug | `logger.L.Debug("progress", "percentage", percentage, "copied", FormatBytes(pt.CopiedBytes), "total", FormatBytes(pt.TotalBytes), "speed", speed, "files_done", pt.CopiedFiles, "files_total", pt.TotalFiles, "eta", eta)` |
| 129 | `log.Printf("Progress: %.2f%% ...")` | Debug | `logger.L.Debug("progress", "percentage", percentage, "files_done", pt.CopiedFiles, "files_total", pt.TotalFiles, "eta", eta)` |
| 147 | `log.Printf("✓ Operation completed! ...")` | Info | `logger.L.Info("operation completed", "bytes", pt.TotalBytes, "files", pt.TotalFiles, "duration", elapsed.String(), "speed", FormatBytes(int64(avgSpeed)))` |
| 154 | `log.Printf("✓ Operation completed! ...")` | Info | `logger.L.Info("operation completed", "files", pt.TotalFiles, "duration", elapsed.String())` |

Note: The progress tick logs at lines 118/129 fire every 100–500ms. Moving to Debug keeps Info clean.

---

### 18. `internal/util/video_rotate.go` (6 calls + new debug)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 42 | `log.Printf("ffprobe output: [%s]", ffprobeOutput)` | Debug | `logger.L.Debug("ffprobe output", "output", ffprobeOutput, "file", filename)` |
| 59 | `log.Printf("Video [%s]: Current=%d°, Adjusting=-%d°, New=%d°", ...)` | Debug | `logger.L.Debug("video rotation calculation", "file", filename, "current", current, "adjust_by", rotateAngle, "new_angle", newAngle)` |
| 74 | `log.Printf("ffmpeg stderr: %s", ffmpegError)` | Error | `logger.L.Error("ffmpeg failed", "stderr", ffmpegError, "file", filename)` |
| 82 | `log.Printf("Warning: failed to remove temp source file %s: %v", ...)` | Warn | `logger.L.Warn("failed to remove temp source file", "path", tempSrc, "err", err)` |
| 86 | `log.Printf("Rotation applied successfully. Output: %s", ...)` | Info | `logger.L.Info("video rotation applied", "output", tempOutput, "file", filename)` |

**New debug logs to add:**

| Location | New Log | Purpose |
|----------|---------|---------|
| Before ffprobe (line 32) | `logger.L.Debug("running ffprobe", "cmd", cmd.String(), "input", tempSrc)` | Debug: ffprobe command |
| Before ffmpeg (line 63) | `logger.L.Debug("running ffmpeg for rotation", "cmd", cmd.String(), "new_angle", newAngle)` | Debug: ffmpeg command |

---

### 19. `internal/util/resolve_duplicate_path.go` (2 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 20 | `log.Printf("File name [%s] duplicate at %s", ...)` | Debug | `logger.L.Debug("file name duplicate", "name", filename, "dir", filepath.Dir(dest))` |
| 27 | `log.Printf("Rename file to %s", ...)` | Debug | `logger.L.Debug("renamed file to resolve duplicate", "name", filepath.Base(newPath))` |

---

### 20. `internal/ws/manager.go` (5 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 54 | `log.Println("New WebSocket client connected")` | Info | `logger.L.Info("websocket client connected")` |
| 61 | `log.Println("WebSocket client disconnected")` | Info | `logger.L.Info("websocket client disconnected")` |
| 70 | `log.Printf("Websocket error: %v", err)` | Error | `logger.L.Error("websocket write error", "err", err)` |
| 96 | `log.Printf("Failed to upgrade to websocket: %v", err)` | Error | `logger.L.Error("failed to upgrade to websocket", "err", err)` |
| 129 | `log.Printf("Error unmarshalling client message: %v", err)` | Error | `logger.L.Error("failed to unmarshal websocket message", "err", err)` |

**Struct change for correlation ID:**

Add `RequestId` field to `OperationMessage` (line 182):

```go
type OperationMessage struct {
    // ... existing fields ...
    RequestId  *string  `json:"requestId,omitempty"` // NEW: correlation ID from HTTP request
}
```

---

### 21. `internal/storage/manager.go` (5 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 34 | `log.Printf("StorageManager: Starting background scan of %s...", ...)` | Info | `logger.L.Info("storage scan started", "path", rootPath)` |
| 40 | `log.Printf("StorageManager: Error accessing path %s: %v", ...)` | Warn | `logger.L.Warn("storage scan path access error", "path", path, "err", err)` |
| 54 | `log.Printf("StorageManager: Background scan failed: %v", err)` | Error | `logger.L.Error("storage scan failed", "err", err)` |
| 60 | `log.Printf("StorageManager: Background scan complete in %v. Total used storage: %d bytes (%.2f GB)", ...)` | Info | `logger.L.Info("storage scan complete", "duration", duration.String(), "total_bytes", totalSize)` |

Note: The 5th call on line 60 is one merged log line — keep it as one Info call.

---

### 22. `internal/schedule/token_cleanup.go` (3 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 22 | `log.Printf("Token cleanup scheduled in %s ...")` | Debug | `logger.L.Debug("token cleanup scheduled", "delay", delay.Round(time.Second).String())` |
| 27 | `log.Printf("Token cleanup failed: %v", err)` | Error | `logger.L.Error("token cleanup failed", "err", err)` |
| 29 | `log.Printf("Token cleanup: removed %d expired/revoked rows", ...)` | Info | `logger.L.Info("token cleanup completed", "removed", deleted)` |

---

### 23. `internal/schedule/thumbnail_maintenance.go` (8 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 33 | `log.Printf("Thumbnail maintenance scheduled in %s ...")` | Debug | `logger.L.Debug("thumbnail maintenance scheduled", "delay", delay.Round(time.Second).String())` |
| 38 | `log.Printf("Thumbnail maintenance failed: %v", err)` | Error | `logger.L.Error("thumbnail maintenance failed", "err", err)` |
| 40 | `log.Printf("Thumbnail maintenance: %d orphaned removed, %d pre-generated", ...)` | Info | `logger.L.Info("thumbnail maintenance complete", "orphaned_removed", deleted, "pre_generated", generated)` |
| 48 | `log.Println("Storage calibration: Re-scanning...")` | Debug | `logger.L.Debug("storage calibration re-scanning")` |
| 87 | `log.Printf("Thumbnail maintenance: upsert error for %s: %v", ...)` | Warn | `logger.L.Warn("thumbnail upsert error during maintenance", "path", path, "err", uerr)` |
| 98 | `log.Printf("Thumbnail maintenance: upsert error for %s: %v", ...)` | Warn | `logger.L.Warn("thumbnail upsert error during maintenance", "path", path, "err", uerr)` |
| 122 | `log.Printf("Thumbnail maintenance: failed to remove orphan file %s: %v", ...)` | Warn | `logger.L.Warn("failed to remove orphan thumbnail", "path", thumbFile, "err", err)` |
| 139,143,147,152 | Various pause/resume/generation logs | Debug + Warn | See detail below |

Breakdown of `preGenerateThumbnails`:

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 139 | `log.Println("Thumbnail maintenance: pausing — API activity detected")` | Info | `logger.L.Info("thumbnail maintenance pausing, API activity detected")` |
| 143 | `log.Println("Thumbnail maintenance: resuming pre-generation")` | Info | `logger.L.Info("thumbnail maintenance resuming")` |
| 147 | `log.Printf("Thumbnail maintenance: failed to generate thumbnail for %s: %v", ...)` | Error | `logger.L.Error("failed to generate thumbnail", "path", f.fullPath, "err", err)` |
| 152 | `log.Printf("Thumbnail maintenance: upsert error after generation for %s: %v", ...)` | Warn | `logger.L.Warn("thumbnail upsert error after generation", "path", f.fullPath, "err", err)` |

---

### 24. `database/migrations.go` (3 calls)

| Line | Old | New Level | New Structured |
|------|-----|-----------|----------------|
| 68 | `log.Println("No new migrations to apply.")` | Info | `logger.L.Info("no new migrations to apply")` |
| 74 | `log.Printf("Applying migration: %s", file)` | Info | `logger.L.Info("applying migration", "file", file)` |
| 101 | `log.Printf("Successfully applied migration: %s", file)` | Info | `logger.L.Info("migration applied successfully", "file", file)` |

---

## Summary: Files to Modify

| # | File | Log Calls | New Debug Added |
|---|------|-----------|-----------------|
| 1 | `cmd/main.go` | 6 | - |
| 2 | `internal/config/config.go` | 16 | - |
| 3 | `internal/config/db_config.go` | 14 | - |
| 4 | `internal/service/file_queue.go` | 1 | - |
| 5 | `internal/service/file_operation.go` | 11 | 1 |
| 6 | `internal/service/share_file_service.go` | 11 | - |
| 7 | `internal/service/sharing_manage_service.go` | 4 | - |
| 8 | `internal/service/video_rename_done.go` | 2 | - |
| 9 | `internal/service/video_disqualified.go` | 1 | - |
| 10 | `internal/service/video_compress_serve.go` | 1 | 1 |
| 11 | `internal/service/thumbnail_generation.go` | 1 | 2 |
| 12 | `internal/service/file_download.go` | 1 | 1 |
| 13 | `internal/service/file_upload.go` | 0 | 3 |
| 14 | `internal/util/file_copy.go` | 3 | 1 |
| 15 | `internal/util/file_move.go` | 5 | - |
| 16 | `internal/util/file_delete.go` | 4 | - |
| 17 | `internal/util/progress.go` | 4 | - |
| 18 | `internal/util/video_rotate.go` | 6 | 2 |
| 19 | `internal/util/resolve_duplicate_path.go` | 2 | - |
| 20 | `internal/ws/manager.go` | 5 | - |
| 21 | `internal/storage/manager.go` | 5 | - |
| 22 | `internal/schedule/token_cleanup.go` | 3 | - |
| 23 | `internal/schedule/thumbnail_maintenance.go` | 12 | - |
| 24 | `database/migrations.go` | 3 | - |
| — | **New files** (logger.go, gin.go) | — | — |
| — | `.env` | — | add `LOG_LEVEL` |
| **Total** | **26 files** | **~120 calls** | **~10 new debug logs** |
