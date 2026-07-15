# Shard 4: Cryptographic Practices and Secret Management

## Scan Instructions

Review all cryptographic operations, key management, hashing, and secret storage.

## Files to Review

- `backend/internal/config/db_config.go` -- SQLiteSecretStore, SQLiteAuditLogStore
- `backend/internal/config/config.go` -- Configuration loading
- `backend/internal/middleware/share_jwt.go` -- Share JWT signing/validation
- `backend/internal/service/sharing_manage_service.go` -- PIN generation, ID generation
- `backend/internal/service/sharing_verify_service.go` -- PIN verification
- `backend/internal/repository/refresh_token_repository.go` -- Token hash storage
- `backend/internal/repository/user_repository.go` -- Password hash, recovery code hash
- `backend/database/migrations/` -- Schema for secrets storage

## Checklist

### Password Hashing
- [ ] bcrypt is used for password hashing
- [ ] bcrypt cost is consistent across codebase (not just tests)
- [ ] Share PINs use bcrypt with appropriate cost
- [ ] No plaintext password storage anywhere

### JWT Security
- [ ] JWT signing secret is managed by gonet-auth (not hardcoded)
- [ ] Secret is stored via SecretStore (not in env vars)
- [ ] Key rotation is supported and working
- [ ] Algorithm confusion attacks are prevented (library handles this)
- [ ] `alg: none` tokens are rejected
- [ ] Token claims are validated against database (not just signature)

### Secret Storage
- [ ] JWT signing secret is stored in `app_settings` table
- [ ] Refresh tokens are stored as hashes
- [ ] Recovery codes are stored as hashes
- [ ] TOTP MFA secrets are stored securely
- [ ] No secrets in source code, `.env.example`, or git history

### Random Number Generation
- [ ] `crypto/rand` is used for all security-sensitive random values
- [ ] PIN generation does not fall back to guessable values on error
- [ ] Operation IDs use CSPRNG
- [ ] Share link IDs use CSPRNG

### Encryption
- [ ] Cookies use `Secure` flag in production
- [ ] `SameSite` attribute is set appropriately
- [ ] `HttpOnly` flag prevents JavaScript access to auth cookies
- [ ] HSTS header is set in production

## Prompt Questions

1. Is the bcrypt cost consistent between test utilities (`CreateTestUser`) and production code?
2. Does `generateRandomPin` fall back to a hardcoded value on CSPRNG failure?
3. Is the JWT signing secret encrypted at rest in `app_settings`?
4. Are share PINs compared using bcrypt (not string equality)?
5. Does the `RotateTx` method in refresh token repository check `RowsAffected`?
6. Is the `SQLiteSecretStore` using parameterized queries?
7. Are recovery codes hashed before storage (not stored in plaintext)?
