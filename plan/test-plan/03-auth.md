# 03 — Auth & Rate Limiting Integration Tests (Session 3)

**Detail file**: `plan/test-plan/03-auth.md`
**Parent**: [index.md](index.md)
**Depends on**: 01 (testutil, route wiring)
**Status**: ✅ done — 2026-05-12

## Implementation Notes

- 48 tests across 4 test files in `internal/controller/`
- Added `middleware.ResetLoginLimiter()` to `internal/middleware/rate_limit.go` for test isolation
- Added `TokenName` to `testutil.TestConfig()` in `internal/testutil/setup.go`
- All tests use `CGO_ENABLED=1 go test ./internal/controller/`

---

## 3.1 `internal/controller/auth_test.go`

Full HTTP integration tests against the real Gin router with in-memory DB.

| Test | Description |
|------|-------------|
| `TestLogin_Success` | valid credentials → 200, access_token + refresh_token cookies |
| `TestLogin_InvalidPassword` | wrong password → 401, "invalid credentials" |
| `TestLogin_UserNotFound` | nonexistent user → 401 (timing-attack safe, no leak) |
| `TestLogin_MissingBody` | empty/invalid JSON → 400 |
| `TestLogin_MFARequired` | user with MFA enabled → 200, `mfa_required:true`, mfa_pending cookie |
| `TestLogin_RateLimiting` | covered by section 3.3 rate limiter tests |
| `TestLogin_LockedAccount` | TODO: if account lockout is implemented |
| `TestRefresh_Success` | valid refresh token → 200, new access_token + new refresh_token |
| `TestRefresh_NoCookie` | no refresh_token cookie → 401 |
| `TestRefresh_Revoked` | use revoked refresh token → 401, "token compromised" |
| `TestRefresh_RotatesToken` | old refresh token is revoked, new one is different |
| `TestRefresh_ReuseDetection` | using a refresh token twice → 401, entire family revoked |
| `TestLogout_Success` | valid session → 200, cookies cleared |
| `TestLogout_NoSession` | no cookies → 200 (no-op, but succeeds) |
| `TestLogout_RevokesTokens` | after logout, access token is invalid (revoked cache hit) |

---

## 3.2 `internal/controller/auth_middleware_test.go`

Test the JWT middleware in isolation and on protected routes.

| Test | Description |
|------|-------------|
| `TestJWTAuth_NoToken` | no cookie → 401 |
| `TestJWTAuth_ExpiredToken` | expired JWT → 401 |
| `TestJWTAuth_InvalidSignature` | tampered token → 401 |
| `TestJWTAuth_PreAuthTokenRejected` | pre-auth token on authenticated route → 401 |
| `TestJWTAuth_RevokedSession` | token family in `RevokedSessionsCache` → 401 |
| `TestJWTAuth_TokenVersionMismatch` | increment token_version in DB, old token → 401 |
| `TestJWTAuth_ValidToken` | valid token → pass through, username in context |
| `TestJWTAuth_BearerHeader` | token in `Authorization: Bearer xxx` header works |
| `TestJWTAuth_MFAMandatoryNotSetup` | MFA mandatory user, no MFA setup → `/api/user/files/file-list` → 403 |
| `TestJWTAuth_MFAMandatoryAllowedPaths` | MFA mandatory user can still access `/api/user/me` and `/api/user/mfa/setup` |

---

## 3.3 Rate limiter tests

| Test | Description |
|------|-------------|
| `TestLoginRateLimiter_Burst` | 5 requests allowed, 6th → 429 |
| `TestLoginRateLimiter_Recovery` | wait 1100ms after burst, request allowed again |
| `TestLoginRateLimiter_DifferentIPs` | rate limiting is per-IP |

---

## 3.4 MFA integration tests `internal/controller/mfa_test.go`

Full MFA enrollment and verification flow through the router.

| Test | Description |
|------|-------------|
| `TestMFASetup_Success` | authenticated user → GET `/api/user/mfa/setup` → 200, returns `secret` and `qr_code` |
| `TestMFASetup_AlreadyEnabled` | MFA already enabled → 400 "MFA already enabled" |
| `TestMFASetup_Unauthenticated` | no token → 401 |
| `TestMFAEnable_Success` | call setup, then POST `/api/user/mfa/enable` with valid TOTP → 200, user.MFAEnabled=true |
| `TestMFAEnable_WrongCode` | POST with wrong TOTP → 400/401 |
| `TestMFAEnable_NoSetupFirst` | enable without calling setup first → error (no secret stored) |
| `TestMFAVerify_Success` | login as MFA user → gets `mfa_required:true` + mfa_pending cookie; POST `/api/mfa/verify` with correct TOTP → 200, access_token + refresh_token |
| `TestMFAVerify_WrongCode` | correct mfa_pending cookie, wrong TOTP → 401 |
| `TestMFAVerify_NoPendingCookie` | no mfa_pending cookie → 401 |
| `TestMFAVerify_ExpiredPending` | expired mfa_pending token → 401 |
| `TestMFAVerify_ReplayAttack` | same TOTP code twice → second attempt fails (TOTP replay prevention via `mfaFailedCache`) |
| `TestMFAVerify_RateLimit` | 6 failed MFA verify attempts → 429 |
| `TestMFAMandatory_NotSetup` | MFA-mandatory user without MFA → `/api/user/files/file-list` → 403; `/api/user/me` and `/api/user/mfa/setup` still work |
| `TestMFAMandatory_AfterSetup` | MFA-mandatory user completes setup → all routes accessible |

Note: TOTP code generation requires `pquerna/otp` (already a dependency). Tests compute a valid code from the secret using `totp.GenerateCode(secret, time.Now())`.

---

## 3.5 APP_JWT=OFF bypass tests `internal/controller/auth_bypass_test.go`

When `APP_JWT=OFF` (dev mode), the JWT middleware is skipped entirely. All `/api/user/*` routes must work without any token.

| Test | Description |
|------|-------------|
| `TestJWTBypass_NoTokenAccess` | GET `/api/user/status` without cookie → 200 (not 401) |
| `TestJWTBypass_NoTokenMe` | GET `/api/user/me` without cookie → 200, username is empty/default |
| `TestJWTBypass_AuthRoutesStillWork` | login, refresh, logout still work normally |
| `TestJWTBypass_RefreshWorks` | refresh with valid cookie → 200, new tokens issued |
| `TestJWTBypass_LogoutWorks` | logout with valid cookie → 200, cookies cleared |
| `TestJWTBypass_AdminRoutes` | admin routes still require admin — or check actual bypass behavior |

**Implementation**: These tests use `TestConfig()` with `APP_JWT=OFF`, set `config.AppConfig` accordingly, and wire routes with `cfg.Auth.AppJwt == "OFF"` triggering the skip in `UserRoutes()`.

