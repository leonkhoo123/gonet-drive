# 04 — User & Admin Management Tests (Session 4)

**Detail file**: `plan/test-plan/04-user-admin.md`
**Parent**: [index.md](index.md)
**Depends on**: 01, 03 (auth flow for login cookies)

---

## 4.1 `internal/controller/user_test.go`

| Test | Description |
|------|-------------|
| `TestGetMe_Success` | `/api/user/me` returns username, role, is_super_admin |
| `TestGetMe_Unauthenticated` | no token → 401 |
| `TestGetStatus_Authenticated` | `/api/user/status` returns `{"status":"ok","message":"authenticated"}` |
| `TestGetStatus_Unauthenticated` | no token → 401 |
| `TestGetSessions_Authenticated` | returns session list |
| `TestRevokeSession_OwnSession` | revoke own session by family_id → 200 |
| `TestRevokeSession_NotOwn` | revoke someone else's family_id → should fail or not find |

---

## 4.2 `internal/controller/admin_test.go`

| Test | Description |
|------|-------------|
| `TestAdminMiddleware_NoToken` | no token → 401 |
| `TestAdminMiddleware_RegularUser` | regular user → 403 "admin access required" |
| `TestAdminMiddleware_Admin` | admin user → passes through |
| `TestAdminMiddleware_SuperAdmin` | superadmin → passes through |
| `TestGetUsers_Admin` | admin lists all users → 200 |
| `TestGetUsers_NonAdmin` | regular user → 403 |
| `TestCreateUser_Admin` | admin creates new user → 200, user exists in DB |
| `TestCreateUser_MissingFields` | empty body → 400 |
| `TestCreateUser_DuplicateUsername` | same username twice → error |
| `TestCreateUser_SuperadminOnlySuperadmin` | regular admin creates superadmin → 403 |
| `TestCreateUser_SuperadminCreatesSuperadmin` | superadmin creates superadmin → 200 |
| `TestDeleteUser_Admin` | admin deletes regular user → 200 |
| `TestDeleteUser_CannotDeleteSelf` | TODO: check if implemented |
| `TestDeleteUser_CannotDeleteSuperAdmin` | deleting superadmin user → 403 |
| `TestRevokeSessions_Admin` | admin revokes another user's sessions → 200 |
| `TestRevokeSessions_CannotRevokeSuperAdmin` | → 403 |

