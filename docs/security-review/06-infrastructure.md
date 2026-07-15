# Shard 6: Infrastructure and Configuration Security

## Scan Instructions

Review all infrastructure, configuration, deployment, and operational security aspects.

## Files to Review

- `backend/internal/config/config.go` -- Configuration loading and validation
- `backend/internal/config/db_config.go` -- Database initialization
- `backend/database/migrations/` -- All migration files
- `backend/internal/controller/routes_setup.go` -- CORS, security headers, trusted proxies
- `backend/internal/controller/ratelimit_store.go` -- Rate limiter implementation
- `Dockerfile` -- Container build
- `.env` -- Environment configuration
- `.gitignore` -- Git exclusions
- `Makefile` -- Build targets

## Checklist

### Configuration
- [ ] `.env` is gitignored (not tracked)
- [ ] No secrets in `.env.example`
- [ ] CORS rejects wildcard origins with credentials
- [ ] Allowed origins are validated on startup
- [ ] `WORK_DIR` must exist on disk at startup
- [ ] Config logging does not output secrets

### Security Headers
- [ ] `X-Content-Type-Options: nosniff` is set
- [ ] `X-Frame-Options: DENY` is set
- [ ] HSTS is set in production
- [ ] `Referrer-Policy` is set
- [ ] `Server` header is suppressed
- [ ] Content-Security-Policy is considered

### Rate Limiting
- [ ] All auth endpoints have rate limiting
- [ ] Rate limiters are per-IP
- [ ] Trusted proxy configuration is enforced in production
- [ ] Rate limiter stores are cleaned up periodically
- [ ] Rate limiter does not use exclusive locks for all requests

### Database Security
- [ ] Migrations run in transactions
- [ ] Migrations are idempotent
- [ ] Token hash index exists
- [ ] Audit log indexes exist
- [ ] Recovery code index exists
- [ ] No plaintext secrets in database schema

### Deployment
- [ ] Docker image does not include `.env` or secrets
- [ ] Build does not embed secrets in binary
- [ ] `go:embed` does not include sensitive files
- [ ] `backend/ui/dist/` is gitignored (must be built)
- [ ] CGO is required for SQLite (correct)

### Audit Logging
- [ ] Security events are audit-logged (login, logout, MFA, token rotation)
- [ ] Audit logs include IP, device info, timestamp
- [ ] Audit logs are retained for configurable period
- [ ] Audit log queries are indexed
- [ ] `json.Marshal` errors in audit logging are handled

## Prompt Questions

1. Does the CORS configuration reject `*` and `null` origins when credentials are allowed?
2. Is the trusted proxy configuration enforced in production (fatal if not set)?
3. Are all migrations idempotent (using `IF NOT EXISTS`)?
4. Does the Dockerfile properly handle the `go:embed` requirement for `backend/ui/dist/`?
5. Are audit logs written asynchronously (non-blocking)?
6. Is the `DeleteExpired` audit log cleanup using parameterized queries?
7. Does the rate limiter have a performance-optimized read path for existing IPs?
