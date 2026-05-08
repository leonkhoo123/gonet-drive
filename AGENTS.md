# AGENTS.md — GoNet Drive

## Architecture

- **Single binary**: Frontend (React/Vite) is embedded into the Go binary via `//go:embed` in `ui/embed.go:5`. The binary serves the SPA and the REST API from one process.
- **Go module name**: `go-file-server` (not `gonet-drive`). Import paths use `go-file-server/...`.
- **Database**: SQLite via `mattn/go-sqlite3` (requires CGO). Migrations run automatically on startup from `database/migrations/*.sql`, which are embedded via `//go:embed`.
- **Frontend SPA routing**: `cmd/main.go:96` — the `NoRoute` handler serves `index.html` for any non-`/api` path that does not match a file in the embedded dist.

## Build

### Full build (binary with embedded frontend)

```bash
# 1) Build frontend
cd frontend && npm run build

# 2) Copy dist to the embed directory
cp -r frontend/dist ui/dist

# 3) Build Go binary (CGO required for SQLite)
CGO_ENABLED=1 go build -o server ./cmd/main.go
```

- The `ui/dist/` directory is gitignored. It must exist before `go build` (even empty), because of `//go:embed all:dist`. If missing, create it: `mkdir -p ui/dist`.
- Docker build handles this sequence automatically (multi-stage Dockerfile).

### Dev — backend only (use Vite for frontend)

```bash
# Backend (from repo root, reads .env)
go run ./cmd/main.go

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

- **No test files exist** in this repo. There is no test command or framework set up.

## Key conventions

### Config and startup

- Config is loaded from `.env` via `godotenv` (from cwd) at `internal/config/config.go:58`.
- `APP_JWTSECRET` is **required** — startup fails if missing (`config.go:111`).
- `WORK_DIR` (served files root) must **exist on disk** at startup or the process exits (`config.go:115`).
- An admin user is auto-created on first run from `ADMIN_USER`/`ADMIN_PASS` env vars.
- `APP_JWT=OFF` disables JWT auth middleware (`user_controller.go:39`).

### Database location

- `APP_ENV=dev` → DB defaults to `./db/config.db`
- `APP_ENV=prod` or other → DB defaults to `/app/db/config.db`
- Override with `DB_DIR` env var.

### Routing conventions

- Public routes (no auth): `/api/login`, `/api/refresh`, `/api/mfa/verify`, `/api/logout`, `/api/config/*`, `/api/shares/*`, `/api/share-files/*`
- Authenticated routes: `/api/user/*`, `/api/user/files/*`, `/api/user/video/*`, `/api/user/photo/*`, `/api/user/music/*`, `/api/user/documents/*`, `/api/user/share/*`, `/api/user/audiobooks/*`, `/api/user/config/*`
- Admin-only: `/api/user/admin/*` (requires admin middleware)
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
- WebSocket connections are managed by `internal/ws/manager.go` and started in a goroutine at `cmd/main.go:74`. Clients receive real-time progress for file operations.
- File operations (copy/move/delete) are processed sequentially via a worker started at `cmd/main.go:77` to avoid filesystem lock contention.
- The backend creates a `.cloud_reserve` directory inside `WORK_DIR` for internal assets (logo, etc.) on startup.


## Temporary directory
- If temporary directory is needed for testing purpose or anything else - create a /temp/ dir to perform all the action inside, and do a clean up afterward.
- Try to not use Temporary directory outside project directory.