# 06 — Sharing Tests (Session 6)

**Detail file**: `plan/test-plan/06-sharing.md`
**Parent**: [index.md](index.md)
**Depends on**: 01, 03 (auth flow)

---

## 6.1 `internal/controller/share_test.go`

| Test | Description |
|------|-------------|
| `TestCreateShare_Success` | create share link → 200, share data + PIN returned |
| `TestCreateShare_WithExpiry` | expires_in_hours=N → correct expires_at |
| `TestCreateShare_NeverExpires` | expires_in_hours=-1 → NeverExpires sentinel |
| `TestCreateShare_InvalidExpiry` | expires_in_hours=0 → 400 |
| `TestCreateShare_ModifyAuthority` | authority=modify → stored correctly |
| `TestCreateShare_ViewAuthorityDefault` | no authority → defaults to "view" |
| `TestCreateShare_Unauthenticated` | no token → 401 |
| `TestListShares_Success` | user lists own shares → 200 |
| `TestListShares_EmptyList` | user with no shares → `shares: []` |
| `TestListShares_OnlyOwnShares` | user A can't see user B's shares |
| `TestToggleBlock_Success` | block/unblock share → 200 |
| `TestToggleBlock_NotFound` | nonexistent share → 404 |
| `TestDeleteShare_Success` | delete own share → 200 |
| `TestDeleteShare_NotFound` | nonexistent share → 404 |
| `TestDeleteShare_NotOwn` | delete someone else's share → 404 (own-username SQL filter) |

---

## 6.2 `internal/controller/share_verify_test.go`

| Test | Description |
|------|-------------|
| `TestVerifySharePIN_Success` | correct PIN → 200, shareJwt cookie set |
| `TestVerifySharePIN_Invalid` | wrong PIN → 401 |
| `TestVerifySharePIN_Blocked` | blocked share → 403 |
| `TestVerifySharePIN_Expired` | expired share → 410 Gone |
| `TestVerifySharePIN_NotFound` | nonexistent share ID → 404 |
| `TestVerifySharePIN_RateLimit` | 6 rapid attempts → 429 |
| `TestCheckSharePermission_ValidToken` | valid shareJwt cookie → 200 |
| `TestCheckSharePermission_NoToken` | no token → 401 |
| `TestCheckSharePermission_WrongShareID` | token for share A, check share B → 403 |

---

## 6.3 `internal/controller/share_file_test.go`

| Test | Description |
|------|-------------|
| `TestShareFileList_Valid` | valid share JWT → list authorized directory |
| `TestShareFileList_PathTraversal` | request path `../../other` → 403 |
| `TestShareFileList_NoToken` | no shareJwt → 401 |
| `TestShareFileDownload_Valid` | download file within share → 200 |
| `TestShareFileDownload_OutsidePath` | download file outside authorized path → 403 |
| `TestShareFileModify_DeniedView` | view-only share attempts upload/delete → 403 |
| `TestShareFileModify_AllowedModify` | modify share can upload/delete/rename/copy/move |
| `TestShareFileRename_Valid` | rename within share → 200 |
| `TestShareFileDelete_Authored` | delete file within allowed path → 200 |

