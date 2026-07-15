# Shard 3: Input Validation and Injection Prevention

## Scan Instructions

Review all input handling for SQL injection, path traversal, XSS, command injection, and other injection vulnerabilities.

## Files to Review

- `backend/internal/repository/*.go` -- All repository files (SQL queries)
- `backend/internal/controller/file_controller.go` -- File operation inputs
- `backend/internal/controller/share_file_controller.go` -- Share file inputs
- `backend/internal/controller/config_controller.go` -- Config inputs
- `backend/internal/service/file_download.go` -- Download path handling
- `backend/internal/service/share_file_service.go` -- Share file path handling
- `backend/internal/util/sanitize.go` -- Sanitization utilities
- `backend/internal/controller/security_test.go` -- Security test coverage
- `backend/internal/config/db_config.go` -- Database config queries

## Checklist

### SQL Injection
- [ ] All queries use parameterized placeholders (`?`)
- [ ] No string concatenation of user input into SQL
- [ ] `fmt.Sprintf` is not used to build SQL with user input
- [ ] Dynamic IN clauses use proper placeholder generation
- [ ] Audit log queries are parameterized

### Path Traversal
- [ ] `SanitizeRepoPath` rejects `..` components
- [ ] `SanitizeRepoPath` uses `filepath.Rel` as secondary check
- [ ] Download functions use `SanitizeRepoPath` (not ad-hoc checks)
- [ ] Share file operations validate against authorized path
- [ ] `SanitizeFilename` blocks reserved names (`.cloud_reserve`, `.cloud_delete`)
- [ ] ZIP generation does not follow symlinks

### XSS Prevention
- [ ] User-Agent is HTML-escaped before storage (`sanitizeDeviceInfo`)
- [ ] Error messages are rendered as text (not HTML)
- [ ] No `dangerouslySetInnerHTML` in frontend
- [ ] No `innerHTML` usage in frontend
- [ ] JSON responses escape HTML by default (Go's `json.Marshal`)

### Command Injection
- [ ] No `os.Exec` with user input
- [ ] ffmpeg commands use argument arrays (not string interpolation)

### Request Smuggling
- [ ] Content-Length and Transfer-Encoding are not both accepted
- [ ] Request body size limits are enforced

## Prompt Questions

1. Are all SQL queries in repositories using parameterized placeholders?
2. Does `SanitizeRepoPath` handle edge cases like empty strings, `.`, and absolute paths?
3. Are file upload filenames sanitized before writing to disk?
4. Is the `video_integrity_repository.go` dynamic IN clause safe against injection?
5. Does the audit log `DeleteExpired` use parameterized queries or `fmt.Sprintf`?
6. Are share PINs validated as exactly 6 digits before bcrypt comparison?
7. Is `filepath.Walk` in download functions protected against symlink traversal?
