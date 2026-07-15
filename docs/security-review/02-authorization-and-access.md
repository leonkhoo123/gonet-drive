# Shard 2: Authorization and Access Control

## Scan Instructions

Review all authorization logic, role-based access control, route protection, and resource ownership verification.

## Files to Review

- `backend/internal/controller/routes_setup.go` -- Route groups, middleware application
- `backend/internal/controller/config_controller.go` -- Config endpoints
- `backend/internal/controller/share_controller.go` -- Share endpoints
- `backend/internal/controller/share_file_controller.go` -- Share file endpoints
- `backend/internal/controller/file_controller.go` -- File operation endpoints
- `backend/internal/middleware/share_jwt.go` -- Share JWT middleware
- `backend/internal/service/sharing_verify_service.go` -- Share permission checks
- `backend/internal/service/file_operation.go` -- File operation ownership

## Checklist

### Route Protection
- [ ] Public routes are explicitly defined (no accidental exposure)
- [ ] Authenticated routes use JWT auth middleware
- [ ] Admin routes use admin middleware (role check)
- [ ] Share routes use share JWT middleware
- [ ] Rate limiters are applied to all auth endpoints

### RBAC
- [ ] Only two roles: `user` and `admin`
- [ ] Admin middleware checks role from database (not just JWT claim)
- [ ] Config endpoints require admin role
- [ ] User management endpoints require admin role
- [ ] File operations are user-scoped (users can only access their own files)

### Share Authorization
- [ ] Share tokens are validated before granting access
- [ ] Share permissions (view vs modify) are enforced server-side
- [ ] Share access is scoped to the authorized path
- [ ] Deleted/blocked shares are immediately invalidated
- [ ] Share token does not grant access to other shares

### Resource Ownership
- [ ] File operations verify the requesting user owns the resource
- [ ] Cancel operation verifies ownership of the operation
- [ ] WebSocket messages are scoped to the connected user/session
- [ ] Users cannot access other users' files via path manipulation

### Path Authorization
- [ ] `SanitizeRepoPath` is used consistently on all file operations
- [ ] `ensureWithinAuthorizedPath` correctly handles root path (`.`)
- [ ] Download functions use the same path validation as other operations
- [ ] `.cloud_reserve` and `.cloud_delete` directories are blocked

## Prompt Questions

1. Are `ConfigRoutes` correctly mounted under the admin router group?
2. Can a regular user cancel another user's file operations via `CancelOperation`?
3. Are WebSocket broadcasts scoped per-user or global?
4. Does the share permission endpoint leak share existence via distinct error codes?
5. Is the `ShareModifyAuthorityMiddleware` safe against nil/wrong-type context values?
6. Can a user with a share JWT for share A probe share B's existence?
7. Are all admin endpoints consistently protected, including user management and config?
