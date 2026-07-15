# Shard 1: Authentication and Session Management

## Scan Instructions

Review all authentication, session management, and token lifecycle code for security issues. Check both the `gonet-auth` library integration and the custom mobile auth implementation.

## Files to Review

- `backend/cmd/main.go` -- Auth initialization, library wiring
- `backend/internal/auth/storage.go` -- Auth storage adapters
- `backend/internal/controller/routes_setup.go` -- Auth route definitions, rate limiters
- `backend/internal/controller/auth_test.go` -- Auth test coverage
- `backend/internal/controller/auth_middleware_test.go` -- Middleware test coverage
- `backend/internal/controller/auth_bypass_test.go` -- JWT bypass mode tests
- `backend/internal/controller/mobile_auth_test.go` -- Mobile auth tests
- `backend/internal/repository/refresh_token_repository.go` -- Token storage
- `backend/internal/repository/user_repository.go` -- User storage

## Checklist

### Login Flow
- [ ] Password comparison uses constant-time comparison (bcrypt)
- [ ] Generic error messages (no "user not found" vs "wrong password" differentiation)
- [ ] Rate limiting on login endpoint (per-IP)
- [ ] Account lockout after N failed attempts
- [ ] Login attempts are audit-logged

### Token Management
- [ ] Refresh tokens are stored as hashes (not plaintext)
- [ ] Token rotation on each refresh (old token invalidated)
- [ ] Refresh token reuse detection (family-based revocation)
- [ ] Tokens have appropriate expiry
- [ ] Token hash index exists for efficient lookups

### Session Management
- [ ] Sessions can be revoked individually
- [ ] All sessions can be revoked at once (revoke-all)
- [ ] Active session listing works correctly
- [ ] Session metadata (device_id, device_info, IP) is recorded

### MFA
- [ ] TOTP secrets are stored securely (hashed or encrypted)
- [ ] Recovery codes are stored as hashes
- [ ] Recovery codes are consumed atomically (single-use)
- [ ] MFA lockout after N failed attempts
- [ ] MFA setup requires authentication
- [ ] MFA bypass paths are correctly defined

### Mobile Auth (Custom Implementation)
- [ ] Tokens returned in JSON body (not cookies)
- [ ] Device ID is captured from body or header
- [ ] Device info is sanitized (XSS prevention)
- [ ] Refresh token rotation works for mobile
- [ ] MFA flow works for mobile
- [ ] Pre-auth tokens are rejected on protected endpoints

### JWT Bypass Mode (APP_JWT=OFF)
- [ ] Rejected in production environment
- [ ] Requires explicit ALLOW_UNSAFE_UNPROTECTED_MODE=true
- [ ] Admin routes remain protected even in bypass mode
- [ ] Login/refresh/logout still function

## Prompt Questions

1. Is the `gonet-auth` library correctly initialized with all required stores (UserLookup, TokenStore, MFAStore, LockoutStore, SecretStore, AuditLogStore)?
2. Are there any residual custom auth implementations that should have been removed?
3. Does the `RotateTx` method properly check `RowsAffected` to prevent orphaned tokens?
4. Is the `authInstance.Start()` lifecycle properly managed with `defer authInstance.Shutdown()`?
5. Are all auth-related error paths audit-logged?
