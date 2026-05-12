# 02 — Utility Unit Tests (Session 2)

**Detail file**: `plan/test-plan/02-utility.md`
**Parent**: [index.md](index.md)
**Depends on**: 01 (testutil)

---

## 2.1 `internal/util/sanitize_test.go`

Pure unit tests, no DB or HTTP needed. Test every edge case.

| Test | Description |
|------|-------------|
| `TestSanitizeRepoPath_Valid` | normal relative paths resolve correctly |
| `TestSanitizeRepoPath_ParentTraversal` | `../etc/passwd`, `./..`, `foo/../../bar` all rejected |
| `TestSanitizeRepoPath_AbsoluteWithinRoot` | absolute paths that happen to be inside root are fine |
| `TestSanitizeRepoPath_AbsoluteOutsideRoot` | `/etc/passwd` safely joined under root (resolves to `<root>/etc/passwd`) |
| `TestSanitizeRepoPaths_BatchValid` | multiple valid paths |
| `TestSanitizeRepoPaths_BatchWithTraversal` | one bad path invalidates the whole batch |
| `TestSanitizeRepoPaths_EmptySlice` | empty input returns empty output |
| `TestSanitizeFilename_Valid` | normal names pass |
| `TestSanitizeFilename_Empty` | `""` rejected |
| `TestSanitizeFilename_Dots` | `.`, `..`, `/` rejected |
| `TestSanitizeFilename_ReservedNames` | `.cloud_delete` rejected; `.cloud_reserve` is fine |
| `TestSanitizeFilename_PathTraversal` | `foo/bar`, `foo/../bar` → only `bar` returned |
| `TestIsSafePathComponent_Valid` | `foo`, `foo-bar`, `foo.bar` all true |
| `TestIsSafePathComponent_Invalid` | `""`, `.`, `..`, `/foo`, `foo/bar`, `foo\\bar` all false |
| `TestGenerateOpID_Uniqueness` | 1000 IDs, all unique |
| `TestTruncateString_Short` | string shorter than max left alone |
| `TestTruncateString_Long` | `"hello world"` at maxLen 8 → `"hello..."` |
| `TestTruncateString_Tiny` | at maxLen 2 → `"he"` (no room for `...`) |

---

## 2.2 `internal/middleware/jwt_test.go`

Unit tests for JWT generation and validation. Only needs `config.CloudConfig`, no DB.

| Test | Description |
|------|-------------|
| `TestGenerateAccessToken_Success` | valid token with correct claims |
| `TestGenerateAccessToken_PreAuth` | pre-auth token has `IsPreAuth=true`, shorter expiry |
| `TestValidateTokenString_Valid` | generated token validates correctly |
| `TestValidateTokenString_Expired` | expired token returns error |
| `TestValidateTokenString_WrongAlgorithm` | token signed with wrong algo fails |
| `TestValidateTokenString_WrongSecret` | token signed with different secret fails |
| `TestGenerateRefreshToken_Length` | 32 random bytes, base64 encoded |
| `TestHashToken_Deterministic` | same input → same hash |
| `TestHashToken_DifferentInputs` | different inputs → different hashes |
| `TestHashToken_NotEmpty` | hash is hex-encoded SHA-256 = 64 chars |
| `TestHashToken_EmptyInput` | empty string input → produces a valid, non-empty hash (SHA-256 of "") |
| `TestHashToken_LongInput` | 10KB string input → produces correct-length hash, no truncation |
| `TestHashToken_Unicode` | UTF-8 input (e.g. CJK, emoji) → produces consistent hash |

---

## 2.3 `internal/util/access_cookie_test.go`

Unit tests for cookie helpers.

| Test | Description |
|------|-------------|
| `TestSetAccessToken_CookieSet` | cookie present with correct name, HttpOnly, SameSite |
| `TestClearAccessToken_CookieCleared` | MaxAge=-1 |
| `TestGetAccessToken_Exists` | can read back what was set |
| `TestGetAccessToken_Missing` | returns error when no cookie |
| `TestSetRefreshToken_CookieSet` | same checks for refresh cookie |
| `TestGetRefreshToken_Exists` | can read back refresh token cookie |
| `TestGetRefreshToken_Missing` | returns error when no refresh cookie |
| `TestClearRefreshToken_CookieCleared` | MaxAge=-1 on refresh cookie |
| `TestSetMfaPendingToken_CookieSet` | MFA pending cookie set correctly |
| `TestSetShareJwt_CookieSet` | share JWT cookie includes share ID in name |
| `TestShareCookie_SameSiteLax` | share JWT cookie uses `SameSiteLaxMode` (not `Strict`) — needed for cross-origin share links |
| `TestAuthCookie_SameSiteStrict` | access/refresh cookies use `SameSiteStrictMode` |
| `TestSecureMode_Local` | `APP_ENV=local` → `secure=false` |
| `TestSecureMode_Prod` | `APP_ENV=prod` → `secure=true` |

---

## 2.4 `internal/util/http_test.go`

Pure unit tests for `GetBaseURL` — no DB, no HTTP server, just `*http.Request` construction.

| Test | Description |
|------|-------------|
| `TestGetBaseURL_BasicHTTP` | plain request → `http://localhost` |
| `TestGetBaseURL_TLS` | request with TLS → `https://hostname` |
| `TestGetBaseURL_XForwardedProto` | `X-Forwarded-Proto: https` → `https://hostname` |
| `TestGetBaseURL_CloudflareVisitor` | `Cf-Visitor` header present → `https://hostname` |
| `TestGetBaseURL_XForwardedHost` | `X-Forwarded-Host: custom.example.com` → takes precedence over `Host` |
| `TestGetBaseURL_XForwardedHostEmpty` | empty `X-Forwarded-Host` → falls back to `r.Host` |
| `TestGetBaseURL_HTTPOverXForwarded` | `X-Forwarded-Proto: http` → `http://hostname` (no false HTTPS) |

