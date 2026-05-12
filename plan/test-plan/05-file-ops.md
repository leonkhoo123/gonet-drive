# 05 — File Operations Tests (Session 5)

**Detail file**: `plan/test-plan/05-file-ops.md`
**Parent**: [index.md](index.md)
**Depends on**: 01, 03 (auth flow)

---

## 5.1 `internal/controller/file_test.go`

Uses a real temp filesystem (via `t.TempDir()`). The `WORK_DIR` config points here. Tests create actual files/folders.

| Test | Description |
|------|-------------|
| `TestFileList_EmptyDir` | empty dir → 200, `items: []` |
| `TestFileList_WithFiles` | create files, list them → correct count and names |
| `TestFileList_PathTraversal` | path with `..` → error |
| `TestFileList_Unauthenticated` | no token → 401 |
| `TestCreateFolder_Success` | valid folder name → 200, dir exists on disk |
| `TestCreateFolder_AlreadyExists` | duplicate → error |
| `TestCreateFolder_InvalidName` | `..` or `/` → error |
| `TestCreateFolder_ProtectedName` | `.cloud_delete` → error |
| `TestRename_Success` | rename file → 200, old gone, new exists |
| `TestRename_ProtectedFile` | rename `.cloud_reserve` → error |
| `TestRename_SameName` | rename to same name → should work (no-op) |
| `TestCopyFiles_Success` | copy files to another dir → 200 |
| `TestCopyFiles_SameDir` | copy to same dir → should create copy with suffix |
| `TestCopyFiles_PathTraversal` | sources or dest with `..` → error |
| `TestMoveFiles_Success` | move files → 200, old gone, new there |
| `TestMoveFiles_SameDir` | move to same dir → error |
| `TestDeleteSoft_Success` | soft delete → 200, files moved to `.cloud_delete` |
| `TestDeleteSoft_ProtectedDir` | delete `.cloud_reserve` → skipped silently |
| `TestDeletePermanent_Success` | permanent delete → 200, files gone |
| `TestDeletePermanent_ProtectedDir` | skip `.cloud_reserve` and `.cloud_delete` |
| `TestFileProperties_Success` | get properties for a file → size, modtime, type |
| `TestFileProperties_MultipleFiles` | batch properties → "Multiple Locations" |
| `TestCheckDuplicates` | upload duplicate detection logic |
| `TestStorageUsage_Authenticated` | returns used/limit JSON |
| `TestCancelOperation_Exists` | cancel a running operation → 200 |
| `TestFileUpload_Success` | POST small file (e.g. 100-byte text) → 200, file exists on disk |
| `TestFileUpload_EmptyFile` | upload 0-byte file → should succeed |
| `TestFileUpload_PathTraversal` | filename with `../` → rejected |
| `TestFileUpload_Unauthenticated` | no token → 401 |
| `TestFileDownload_Success` | GET a known file → 200, body matches file content |
| `TestFileDownload_NotFound` | GET nonexistent file → 404 |
| `TestFileDownload_PathTraversal` | path with `..` → rejected |
| `TestFileProperties_NotFound` | properties on a nonexistent file → error |
| `TestFileProperties_PathTraversal` | properties with `..` in path → error |

