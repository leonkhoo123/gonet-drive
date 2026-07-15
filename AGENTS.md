# AGENTS.md — GoNet Drive

## Architecture

- **Single binary**: Frontend (React/Vite) is embedded into the Go binary via `//go:embed` in `backend/ui/embed.go:5`. The binary serves the SPA and the REST API from one process.
- **Go module name**: `go-file-server` (not `gonet-drive`). Import paths use `go-file-server/...`.
- **Database**: SQLite via `mattn/go-sqlite3` (requires CGO). Migrations run automatically on startup from `backend/database/migrations/*.sql`, which are embedded via `//go:embed`.
- **Frontend SPA routing**: `backend/cmd/main.go:96` — the `NoRoute` handler serves `index.html` for any non-`/api` path that does not match a file in the embedded dist.

## Build

### Full build (binary with embedded frontend)

```bash
# 1) Build frontend
cd frontend && npm run build

# 2) Copy dist to the embed directory
cp -r frontend/dist backend/ui/dist

# 3) Build Go binary (CGO required for SQLite)
cd backend && CGO_ENABLED=1 go build -o ../server ./cmd/main.go
```

- The `backend/ui/dist/` directory is gitignored. It must exist before `go build` (even empty), because of `//go:embed all:dist`. If missing, create it: `mkdir -p backend/ui/dist`.
- Docker build handles this sequence automatically (multi-stage Dockerfile).

### Dev — backend only (use Vite for frontend)

```bash
# Backend (from repo root, reads .env)
cd backend && go run ./cmd/main.go

# Frontend (separate terminal, from frontend/)
cd frontend && npm run dev
```

- Frontend dev mode: When `VITE_BUILD_PROFILE=local` (or when running via `vite dev`), the frontend calls `http://localhost:3333/api` instead of `/api`. See `frontend/src/config.ts:7-12`.
- The backend `.env` sets `LISTEN_ADDR=:3333` by default, matching this port.

## Lint & Typecheck

- **Frontend lint**: `cd frontend && npm run lint` (ESLint + typescript-eslint with type-aware rules).
- **Frontend typecheck**: `npx tsc -b` (included in `npm run build`).
- **Backend**: No linter or formatter configured. Standard `go vet`, `go fmt` apply.

## Tests

- **Test runner**: `go test` + `stretchr/testify`. Requires `CGO_ENABLED=1` for SQLite.
- **Database**: Each test uses in-memory SQLite (`:memory:?_busy_timeout=5000`), isolated via `t.TempDir()`.
- **Test helpers**: [`backend/internal/testutil/setup.go`](backend/internal/testutil/setup.go:65) — `SetupTestDB`, `SetupServices`, `CreateTestUser`, `TestConfig`.
- **Test files**: `*_test.go` files live alongside their production code in `internal/`.

```bash
make test          # Quick run
make test-verbose  # Verbose output
make test-cover    # With coverage
make test-race     # Race detection
```

### Rule: run tests after writing features

**After implementing or modifying any Go code**, run `make test`. All tests must pass before the change is complete. Add test cases for new behavior.

- ✅ Pass → proceed. ❌ Fail → fix first. Do not leave failing tests behind.

## Key conventions

### Config and startup

- Config is loaded from `.env` via `godotenv` at `backend/internal/config/config.go:58`.
- `WORK_DIR` (served files root) must **exist on disk** at startup or the process exits (`config.go:115`).
- The first admin is provisioned at runtime via the setup flow
  (`GET /api/setup/status` → `POST /api/setup/admin`), not from env vars.
  `ADMIN_USER`/`ADMIN_PASS`/`APP_JWTSECRET`/`TOKEN_NAME`/`COOKIE_*` are gone —
  the gonet-auth library auto-generates and rotates its own JWT signing secret
  via the `SecretStore` (`SQLiteSecretStore`, persisted in `app_settings`).
- `APP_JWT=OFF` disables JWT auth middleware. It is rejected in production and
  otherwise requires `ALLOW_UNSAFE_UNPROTECTED_MODE=true` (`cmd/main.go:148`).
- Roles are `user` and `admin` only (the `superadmin` role is removed).

### Database location

- `APP_ENV=dev` → DB defaults to `../db/config.db` (repo root `db/` directory)
- `APP_ENV=prod` or other → DB defaults to `/app/db/config.db`
- Override with `DB_DIR` env var.

### Routing conventions

- Public routes (no auth): `/api/login`, `/api/refresh`, `/api/mfa/verify`, `/api/mfa/recovery`, `/api/logout`, `/api/setup/status`, `/api/setup/admin` (first-run admin provisioning), `/api/config/*`, `/api/shares/*`, `/api/share-files/*`
- Authenticated routes: `/api/user/*`, `/api/user/files/*`, `/api/user/video/*`, `/api/user/photo/*`, `/api/user/music/*`, `/api/user/documents/*`, `/api/user/share/*`, `/api/user/audiobooks/*`, `/api/user/config/*`, `/api/user/me`, `/api/user/me/sessions`, `/api/user/me/sessions/revoke`
- Admin-only: `/api/user/admin/*` (requires admin middleware), incl. `/api/user/admin/users` (create/list), `/api/user/admin/users/:id` (delete), `/api/user/admin/users/:id/revoke-all`
- Mobile (custom, token-in-body): `/api/mobile/login`, `/api/mobile/refresh`, `/api/mobile/mfa/verify`, `/api/mobile/logout`
- WebSocket: `/api/user/ws` (authenticated)
- Health: `/api/health` (no auth)

### Frontend stack

- React 19, TypeScript 5.9, Vite 7, Tailwind CSS 4, shadcn/ui (new-york style), react-router-dom v7
- PWA via `vite-plugin-pwa`
- Path alias: `@/` maps to `frontend/src/`
- shadcn/ui component files (`frontend/src/components/ui/*`) are excluded from lint (`eslint.config.js:11`)

### Generation note

- `npx shadcn@latest add <component>` can be used to add shadcn/ui components. It writes to `frontend/src/components/ui/`.

## Operational notes

- **ffmpeg** is required at runtime (installed in Docker image) — used by the video module for on-the-fly transcoding.
- WebSocket connections are managed by `backend/internal/ws/manager.go` and started in a goroutine at `backend/cmd/main.go:74`. Clients receive real-time progress for file operations.
- File operations (copy/move/delete) are processed sequentially via a worker started at `backend/cmd/main.go:77` to avoid filesystem lock contention.
- The backend creates a `.cloud_reserve` directory inside `WORK_DIR` for internal assets (logo, etc.) on startup.


## Temporary directory
- If temporary directory is needed for testing purpose or anything else - create a /temp/ dir to perform all the action inside, and do a clean up afterward.
- Try to not use Temporary directory outside project directory.

## Updating gonet-auth dependency

gonet-auth is a private GitHub module. It ships as **two** modules that are both
consumed here: the core (`github.com/leonkhoo123/gonet-auth`) and the Gin adapter
(`github.com/leonkhoo123/gonet-auth/adapters/gin`). Both need a `require` and a
local `replace`. To bump the version:

```bash
# 1. Update the versions in go.mod (both modules)
go mod edit -require github.com/leonkhoo123/gonet-auth@v1.1.0
go mod edit -require github.com/leonkhoo123/gonet-auth/adapters/gin@v1.1.0

# 2. Drop the local replaces so Go fetches from GitHub
go mod edit -dropreplace github.com/leonkhoo123/gonet-auth
go mod edit -dropreplace github.com/leonkhoo123/gonet-auth/adapters/gin

# 3. Fetch from GitHub and regenerate go.sum
GOPRIVATE=github.com/leonkhoo123 go mod tidy

# 4. Restore the local replaces for dev (NO go mod tidy after this!)
go mod edit -replace github.com/leonkhoo123/gonet-auth=../../gonet-auth
go mod edit -replace github.com/leonkhoo123/gonet-auth/adapters/gin=../../gonet-auth/adapters/gin

# 5. Commit the updated go.mod + go.sum
git add backend/go.mod backend/go.sum && git commit -m "bump gonet-auth to v1.1.0"
```

**Important**: Do NOT run `go mod tidy` after step 4 — it removes the remote
checksums from `go.sum` because the local replaces make them unnecessary.
The checksums must stay in the committed `go.sum` so CI and other developers
can fetch the remote modules.

Never commit the repo without the `replace` directives — **both** must be present
in every commit so local dev works. The Docker pipeline drops them at build time.

## Release pipeline

Kaniko builds run via the `kaniko-build` Tekton pipeline in `infra-git`.
The pipeline fetches a GitHub PAT from Vault and passes it as `--build-arg GITHUB_TOKEN`
so Go can download private modules during `go mod download`.
The pipeline drops **both** local `replace` directives (core + `adapters/gin`)
at build time so the tagged GitHub versions are fetched instead.