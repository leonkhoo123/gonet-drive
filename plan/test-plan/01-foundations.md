# 01 — Foundations (Session 1)

**Detail file**: `plan/test-plan/01-foundations.md`
**Parent**: [index.md](index.md)
**Depends on**: nothing
**Required by**: everything

---

## 1.1 Add `testify` dependency

**File**: `go.mod`
**Action**: `go get github.com/stretchr/testify`

Nothing else — testify's `assert` and `require` subpackages are all we need.

---

## 1.2 Test helper: `internal/testutil/setup.go`

Create a shared test utility package that every test file imports.

### Responsibilities:

- **`TestConfig() *config.CloudConfig`** — builds a minimal config with:
  - `APP_JWTSECRET=test-secret-key-for-testing-only`
  - `APP_JWT=ON` (auth enabled by default; override for bypass tests)
  - `ADMIN_USER=admin` / `ADMIN_PASS=admin123` (for bootstrap awareness)
  - `WORK_DIR` = a unique temp directory (created via `os.MkdirTemp` or `t.TempDir()`)
  - `APP_ENV=local` (no secure cookies)
  - Sensible token durations (15min access, 7d refresh, 3min MFA pending)
  - Allowed origins `*`

- **`SetupTestDB(t *testing.T) *sql.DB`** — creates an in-memory SQLite DB (`:memory:`), runs migrations, seeds default cloud config rows, calls `config.RefreshCloudConfigCache()` to populate `config.AppCloudConfig`. Does NOT call `bootstrapAdmin()` — admin user creation is explicit in tests.

- **`SetupServices(t, db *sql.DB, workDir string)`** — constructs the three core service instances from the real constructors:
  - `service.NewUserService(NewSQLiteUserRepo(db), NewSQLiteRefreshTokenRepo(db))`
  - `service.NewSharingService(NewSQLiteSharingRepo(db), workDir)`
  - The caller gets back `(*UserService, *SharingService, repository.CloudConfigRepository)` for wiring into routes.

- **`SetupRouter(t, cfg *config.CloudConfig, ...services)`** — builds a Gin router with **only the routes needed for the test**. This is a **modular** design: separate functions for each route group so tests don't wire everything at once:
  - `SetupPublicAuthRoutes(router, cfg, userService)` — /login, /refresh, /mfa/verify, /logout
  - `SetupAuthenticatedRoutes(router, cfg, userService, sharingService, configRepo)` — all /api/user/** routes with JWTAuthMiddleware
  - `SetupAdminRoutes(router, cfg, userService)` — /api/user/admin/** routes with AdminMiddleware
  - `SetupShareRoutes(router, cfg, sharingService)` — /api/share/** public routes, /api/user/share/** authenticated routes
  - `SetupShareFileRoutes(router, cfg, sharingService, shareRepo)` — /api/share-file/** routes with ShareAuthMiddleware
  - `SetupFileRoutes(router, cfg, sharingService)` — /api/user/files/** routes
  - `SetupConfigRoutes(router, cfg, configRepo)` — /api/config/** and /api/user/config/** routes
  - This also forces route wiring to be extracted from `cmd/main.go` into testable functions — **a design improvement the plan mandates** (see Section 1.2a below).

- **`CreateTestUser(t, db *sql.DB, username, password, role string) (*model.User, string)`** — inserts a user with a bcrypt-hashed password. Returns the created user and the raw (plaintext) password. Does NOT call `bootstrapAdmin()` — admin user creation is explicit.

- **`LoginAndGetCookie(t, router, username, password) *http.Cookie`** — performs POST `/api/login` with JSON body, extracts the `access_token` Set-Cookie from `*httptest.ResponseRecorder`, returns it.

- **`LoginAndGetCookies(t, router, username, password) (*http.Cookie, *http.Cookie)`** — same but also returns the `refresh_token` cookie.

- **`MakeAuthRequest(t, router, method, path string, body io.Reader, accessCookie *http.Cookie) *httptest.ResponseRecorder`** — builds an `*http.Request` with the access cookie attached, returns the recorder.

- **`MakeAuthRequestJSON(t, router, method, path string, bodyJSON interface{}, accessCookie *http.Cookie) *httptest.ResponseRecorder`** — convenience variant that marshals JSON body.

### Key design decisions:
- Each test gets its own `t.TempDir()` for `WORK_DIR` — completely isolated filesystem
- Each test gets its own `:memory:` SQLite DB — completely isolated data
- `config.AppConfig` and `config.DB` global variables are set in setup and restored in teardown via `t.Cleanup()`

---

## 1.2a Route wiring extraction (mandatory prerequisite)

Before any tests can be written, `cmd/main.go` must be refactored to extract route wiring into named functions in the controller package. Currently `PublicUserRoutes` and `UserRoutes` exist, but they don't cover all routes and they accept too many parameters. The refactoring creates one function per route group — matching the modular `Setup*` functions listed above.

**Why this must come first**: Without modular route wiring, each test file would duplicate partial wiring, leading to inconsistencies. By extracting route wiring, the test helpers call the **exact same functions** as `main()`, guaranteeing parity.

### Global state isolation strategy:

The codebase has three package-level globals that tests must manage:

| Global | Type | Package | Used by |
|--------|------|---------|---------|
| `config.DB` | `*sql.DB` | `internal/config` | JWT middleware (token_version query), some services |
| `config.AppConfig` | `*CloudConfig` | `internal/config` | All middleware and controllers |
| `config.AppCloudConfig` | `*DatabaseConfig` | `internal/config` | Config routes, manifest endpoint (via `config.GetCloudConfig()`) |
| `middleware.RevokedSessionsCache` | `*cache.Cache` | `internal/middleware` | JWT middleware (revocation check) |
| `state.ShareStatusCache` | `*cache.Cache` | `internal/state` | Share auth middleware |

**Strategy — serialized globals, no `t.Parallel()`:**

```
// In testutil/setup.go — a mutex that serializes global state mutation

var testGlobalMu sync.Mutex

func SetupTestDB(t *testing.T) *sql.DB {
    testGlobalMu.Lock()
    t.Cleanup(testGlobalMu.Unlock)
    // ... create DB, set config.DB, set config.AppConfig ...
    // ... run migrations, seed defaults, call RefreshCloudConfigCache() ...
    t.Cleanup(func() {
        config.DB = nil
        config.AppConfig = nil
        config.AppCloudConfig = nil
        middleware.RevokedSessionsCache.Flush()
        state.ShareStatusCache.Flush()
    })
}
```

This approach:
- Serializes all tests (tests can still run in any order, just not concurrently)
- Cheap — `:memory:` SQLite setup takes < 10ms per test
- No global-variable injection refactoring needed
- Tests can freely mutate globals without cross-test contamination

**If parallel tests are desired later**, the codebase would need to be refactored to pass `*sql.DB` and `*cache.Cache` through dependency injection rather than accessing package globals. That's a separate refactoring effort, not part of this test plan.

---

## 1.3 Test helper: `internal/testutil/assertions.go`

Custom test assertions beyond what testify provides:

- `AssertJSONResponse(t, recorder, expectedStatus, expectedBody)` — unmarshals response, checks status code and JSON body match
- `AssertAuthError(t, recorder, expectedStatus)` — ensures response is a JSON `{"error":"..."}` with proper auth error
- `AssertNoAuthCookie(t, recorder, cookieName)` — ensures the response has cleared (MaxAge=-1) the named cookie

---

## 1.4 Repository layer tests: `internal/repository/*_test.go`

Testing repositories directly against `:memory:` SQLite catches SQL bugs at the source, is extremely fast, and does not require HTTP wiring. Every repository interface method should have at least one test.

### `internal/repository/user_repository_test.go`:

| Test | Description |
|------|-------------|
| `TestUserRepo_Create` | insert user → row exists, fields roundtrip correctly |
| `TestUserRepo_GetByUsername_Found` | create user → retrieve by username → correct fields |
| `TestUserRepo_GetByUsername_NotFound` | query nonexistent → `sql.ErrNoRows` |
| `TestUserRepo_GetByID` | create user → retrieve by generated UUID |
| `TestUserRepo_ListAll_Empty` | no users → empty slice |
| `TestUserRepo_ListAll_WithUsers` | create 3 users → all returned |
| `TestUserRepo_Update` | create, update role → retrieved role matches |
| `TestUserRepo_Delete` | create, delete → GetByUsername returns `sql.ErrNoRows` |
| `TestUserRepo_Exists_True` | create user → Exists returns true |
| `TestUserRepo_Exists_False` | no user → Exists returns false |
| `TestUserRepo_IncrementTokenVersion` | create, increment → token_version increases by 1 |
| `TestUserRepo_IncrementTokenVersionByID` | same as above, by UUID |
| `TestUserRepo_UpdateMFASecret` | create, set secret → retrieve confirms secret stored |
| `TestUserRepo_EnableMFA` | create, enable → MFAEnabled becomes true |
| `TestUserRepo_DuplicateUsername` | create twice with same username → error |

### `internal/repository/refresh_token_repository_test.go`:

| Test | Description |
|------|-------------|
| `TestTokenRepo_Create` | insert token → retrievable by hash |
| `TestTokenRepo_GetByTokenHash_NotFound` | query nonexistent hash → error |
| `TestTokenRepo_GetActiveSessions_Empty` | no tokens for user → empty |
| `TestTokenRepo_GetActiveSessions_WithSessions` | create 3 tokens for user → all 3 returned |
| `TestTokenRepo_RevokeByID` | create token, revoke → GetByTokenHash returns error |
| `TestTokenRepo_RevokeByFamilyID` | create 2 tokens same family → revoke family → both gone |
| `TestTokenRepo_RevokeByUsername` | create tokens for 2 users → revoke one → only that user's gone |
| `TestTokenRepo_RevokeByUsernameAndFamilyID` | targeted revocation |

### `internal/repository/sharing_repository_test.go`:

| Test | Description |
|------|-------------|
| `TestShareRepo_Create` | insert share → retrievable with all fields |
| `TestShareRepo_GetByID_Found` | create → retrieve by ID |
| `TestShareRepo_GetByID_NotFound` | nonexistent ID → error |
| `TestShareRepo_ListByUsername_OnlyOwn` | create shares for user A and B → list A → only A's shares |
| `TestShareRepo_UpdateBlockedStatus` | create share → block → IsBlocked=true |
| `TestShareRepo_Delete` | create share → delete → GetByID returns error |

### `internal/repository/cloud_config_repository_test.go`:

| Test | Description |
|------|-------------|
| `TestConfigRepo_ListEnabled` | seed defaults → returns only enabled rows |
| `TestConfigRepo_ListAllNotDeleted` | seed defaults → returns all non-deleted rows |
| `TestConfigRepo_Update` | update config value → retrieved value matches |

**Design note**: Repository tests use `SetupTestDB(t)` (from Section 1.2) and the real concrete implementations (`NewSQLiteUserRepo(db)`, etc.). No HTTP, no Gin, no cookies — pure DB unit tests. These should run in < 1s total.

---

## 1.5 Service layer tests (optional, low-priority)

Construction of service instances is trivial (they're dependency-injected structs). Most service methods take `*gin.Context` directly, making them harder to test without HTTP wiring. However, any **pure-logic helper methods** that take plain data types (not `*gin.Context`) should be tested here.

**Check before writing**: Grep service files for methods that don't take `*gin.Context` as their first parameter. If none exist, skip this section entirely — the integration tests in Sections 03–06 will cover service logic through the HTTP layer.

