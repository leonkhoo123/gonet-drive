# Security Review Prompt -- GoNet Drive

## Purpose

This document provides a structured security scanning prompt for reviewing the GoNet Drive codebase. It is designed to be used by security reviewers, AI assistants, or automated tools to systematically identify security vulnerabilities.

## How to Use

1. **Sharded structure**: Each shard covers a specific security domain. Review them in order or target specific areas.
2. **Findings template**: Use `07-findings-template.md` to record discovered issues.
3. **Severity scale**: All findings use `CRITICAL > HIGH > MEDIUM > LOW > INFO`.

## Scope

This review covers the GoNet Drive refactoring that:
- Replaces custom auth with the `gonet-auth` library
- Preserves custom mobile auth (not supported by gonet-auth)
- Adds MFA, audit logging, recovery codes, and rate limiting
- Restructures the project from root to `backend/` directory

## Shards

| Shard | File | Focus |
|-------|------|-------|
| 1 | `01-auth-and-session.md` | Authentication, session management, token lifecycle |
| 2 | `02-authorization-and-access.md` | Authorization, RBAC, route protection |
| 3 | `03-input-validation.md` | Input validation, injection, path traversal |
| 4 | `04-crypto-and-secrets.md` | Cryptographic practices, secret management |
| 5 | `05-frontend-security.md` | Client-side security, XSS, token storage |
| 6 | `06-infrastructure.md` | Infrastructure, config, deployment security |
| 7 | `07-findings-template.md` | Standardized findings recording template |

## Quick Checklist

- [ ] All auth endpoints are rate-limited
- [ ] No hardcoded secrets in source code
- [ ] Refresh tokens are stored as hashes
- [ ] JWT signing secret is managed by gonet-auth via SecretStore
- [ ] Admin routes require admin middleware
- [ ] Path traversal is blocked on all file operations
- [ ] CORS does not allow wildcard origins with credentials
- [ ] Cookies use HttpOnly, Secure, SameSite attributes
- [ ] MFA secrets and recovery codes are hashed in DB
- [ ] No residual custom auth code remains
- [ ] Mobile auth correctly returns tokens in body (not cookies)
- [ ] Audit logging captures security-relevant events
