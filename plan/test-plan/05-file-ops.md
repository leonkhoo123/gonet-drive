# 05 — File Operations Tests (Session 5)

**Detail file**: `plan/test-plan/05-file-ops.md`
**Parent**: [index.md](index.md)
**Depends on**: 01, 03 (auth flow)

---

## 5.1 File Operations Test Files

Uses a real temp filesystem (via `t.TempDir()`). The `WORK_DIR` config points here. Tests create actual files/folders.

Shared helpers live in `file_helpers_test.go` (`setupFileRouter`, `waitForJobQueue`, `writeTestFile`).

### 5.1.1 `internal/controller/file_helpers_test.go`
Shared setup functions for all file operation tests.

### 5.1.2 `internal/controller/file_list_test.go`

| Test | Description |
|------|-------------|
| `TestFileList_EmptyDir` | empty dir → 200, `items: []` |
| `TestFileList_WithFiles` | create files, list them → correct count and names |
| `TestFileList_PathTraversal` | path with `..` → error |
| `TestFileList_Unauthenticated` | no token → 401 |

### 5.1.3 `internal/controller/file_folder_test.go`

| Test | Description |
|------|-------------|
| `TestCreateFolder_Success` | valid folder name → 200, dir exists on disk |
| `TestCreateFolder_AlreadyExists` | duplicate → error |
| `TestCreateFolder_InvalidName` | `..` or `/` → error |
| `TestCreateFolder_ProtectedName` | `.cloud_delete` → error |
| `TestRename_Success` | rename file → 200, old gone, new exists |
| `TestRename_ProtectedFile` | rename `.cloud_reserve` → error |
| `TestRename_SameName` | rename to same name → should error |

### 5.1.4 `internal/controller/file_copy_move_test.go`

| Test | Description |
|------|-------------|
| `TestCopyFiles_Success` | copy files to another dir → 200 |
| `TestCopyFiles_SameDir` | copy to same dir → should create copy with suffix |
| `TestCopyFiles_PathTraversal` | sources or dest with `..` → error |
| `TestMoveFiles_Success` | move files → 200, old gone, new there |
| `TestMoveFiles_SameDir` | move to same dir → error |

### 5.1.5 `internal/controller/file_delete_test.go`

| Test | Description |
|------|-------------|
| `TestDeleteSoft_Success` | soft delete → 200, files moved to `.cloud_delete` |
| `TestDeleteSoft_ProtectedDir` | delete `.cloud_reserve` → error |
| `TestDeletePermanent_Success` | permanent delete → 200, files gone |
| `TestDeletePermanent_ProtectedDir` | skip `.cloud_reserve` and `.cloud_delete` |

### 5.1.6 `internal/controller/file_properties_test.go`

| Test | Description |
|------|-------------|
| `TestFileProperties_Success` | get properties for a file → size, modtime, type |
| `TestFileProperties_MultipleFiles` | batch properties → "Multiple Locations" |
| `TestFileProperties_NotFound` | properties on a nonexistent file → error |
| `TestFileProperties_PathTraversal` | properties with `..` in path → error |
| `TestCheckDuplicates` | upload duplicate detection logic |
| `TestStorageUsage_Authenticated` | returns used/limit JSON |
| `TestCancelOperation_Exists` | cancel a running operation → 200 |

### 5.1.7 `internal/controller/file_upload_download_test.go`

| Test | Description |
|------|-------------|
| `TestFileUpload_Success` | POST small file (e.g. 100-byte text) → 200, file exists on disk |
| `TestFileUpload_EmptyFile` | upload 0-byte file → should succeed |
| `TestFileUpload_PathTraversal` | destination with `../` → rejected |
| `TestFileUpload_PathTraversal_Filename` | filename with `../` → sanitized to basename |
| `TestFileUpload_Unauthenticated` | no token → 401 |
| `TestFileDownload_Success` | GET a known file → 200, body matches file content |
| `TestFileDownload_NotFound` | GET nonexistent file → 404 |
| `TestFileDownload_PathTraversal` | path with `..` → rejected |

