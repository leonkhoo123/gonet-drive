# 07 — Edge Cases, Security & Sanity (Session 7)

**Detail file**: `plan/test-plan/07-security.md`
**Parent**: [index.md](index.md)
**Depends on**: 01

---

## 7.1 `internal/controller/health_test.go`

| Test | Description |
|------|-------------|
| `TestHealthEndpoint_NoAuth` | GET `/api/health` returns 200 (no auth needed) |
| `TestHealthEndpoint_NoRoute` | GET on unknown `/api/xxx` → 404 |
| `TestSPAFallback` | GET `/` or `/some-page` → serves `index.html` |

---

## 7.2 Configuration tests `internal/controller/config_test.go`

| Test | Description |
|------|-------------|
| `TestGetManifest_Valid` | GET `/api/config/manifest` → JSON with PWA fields |
| `TestGetLogo_ReturnsImage` | GET `/api/config/logo` → PNG image |
| `TestUpdateLogo_Success` | PUT as admin → 200 |
| `TestUpdateLogo_NonPNG` | non-.png → 400 |
| `TestUpdateLogo_NonAdmin` | regular user → 403 |
| `TestUpdateLogo_ExceedsSize` | file > 5MB → 400 |
| `TestListConfigs_Valid` | GET `/api/user/config` → list of configs |
| `TestUpdateConfig_Valid` | PUT with new value → 200, cache refreshed |
| `TestUpdateConfig_InvalidID` | non-numeric ID → 400 |

---

## 7.3 Security-focused tests

| Test | Description |
|------|-------------|
| `TestSQLInjection_Username` | login with `' OR '1'='1` → 401 (parameterized queries) |
| `TestXSS_Path` | path with `<script>` in file list → sanitized/escaped |
| `TestCSRF_NoTokenCookie` | POST without CSRF token → (NOTE: JWT in cookie + CORS already mitigates, but document) |
| `TestPathTraversal_AllEndpoints` | fuzz every file endpoint with `../../` variants |
| `TestTokenInCookieNotAccessibleToJS` | verify HttpOnly flag on all auth cookies |
| `TestCORS_HeadersPresent` | OPTIONS preflight → correct headers |

