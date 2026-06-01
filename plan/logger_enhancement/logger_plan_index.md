# GoNet Drive — Logger Enhancement Plan

This directory contains a sharded, step-by-step implementation plan for enhancing the logging system of `gonet-drive`.  
Each `NN-name.md` file describes **what** to change for a specific phase, **how** to structure the code, and the **expected behaviour**.

## Status Legend

| Status | Meaning |
|--------|---------|
| `none` | Not yet started |
| `implemented` | AI has written the code; waiting for human review |
| `done-review` | Human reviewed, all issues fixed, tests pass |

---

## Workflow for AI Agents

### When you are asked to IMPLEMENT a module:

1. Read the relevant `NN-name.md` plan file from this directory.
2. Write code following the plan's structure and Go best practices.
3. Run `go vet ./... && go build ./...` from repo root.
4. Run the tests with `CGO_ENABLED=1 go test ./... -count=1` from repo root.
5. On success, update the status in this `index.md` from `none` → `implemented`.
6. Commit only if explicitly asked.

### When you are asked to REVIEW a module:

1. Read the plan file and the code that was written.
2. Verify all scenarios are covered.
3. Verify tests actually pass (`CGO_ENABLED=1 go test ./... -count=1`).
4. Check for race conditions, missing edge cases, import cleanup.
5. Fix any issues found.
6. Update the status in this `index.md` from `implemented` → `done-review`.
7. Commit only if explicitly asked.

### When updating status:

- The status column in the Sections table below uses the keywords: `none`, `implemented`, `done-review`.
- Timestamps use ISO format (e.g. `2026-06-01`).
- Update `Started` when you begin a section, `Done` when status reaches `done-review`.

---

## Sections

| # | Section | Status | File | Started | Done |
|---|---------|--------|------|---------|------|
| 00 | Architecture & Decisions | none | [00-architecture.md](00-architecture.md) | - | - |
| 01 | Config & Middleware (Gin + Correlation IDs) | none | [01-config-middleware.md](01-config-middleware.md) | - | - |
| 02 | File Migration (24 files, level mapping) | none | [02-file-migration.md](02-file-migration.md) | - | - |

---

## Quick Summary

| Phase | Scope | Est. Effort |
|-------|-------|-------------|
| Phase 0 | Create `internal/logger/` package (logger.go + gin.go), design decisions | Low |
| Phase 1 | Config changes, Gin middleware, correlation IDs, startup wiring in `cmd/main.go` | Medium |
| Phase 2 | Migrate 24 files from `log.*`/`fmt.Printf` → `logger.L.*`, add new debug logs | High |

**Total**: ~1 session for phases 0–1, ~1–2 sessions for phase 2.

---

## Dependency Graph

```
Phase 0 (logger package) ──must come first──► Phase 1 ──► Phase 2
                                       Phase 1 ──► Phase 2 (files can be done in any order)
```

Phase 1 depends on Phase 0 (needs `logger.L` to exist).  
Phase 2 depends on Phase 0 + 1 (needs logger API + config + middleware).  
Within Phase 2, individual file migrations are independent and can be done in any order.

---

## Instructions for AI

### How to use this plan

1. **Read the index first** to understand overall structure and dependency order.
2. **Open the detail file** linked in the Sections table above.
3. **Work through phases** in order (0 → 1 → 2).
4. **Track progress** in this `index.md` (see status update rules above).
5. **Run tests** after each phase: `CGO_ENABLED=1 go test ./... -count=1`.
6. **Never commit** unless explicitly asked.

### Key Design Goals

- **Zero new external dependencies** — use `log/slog` from stdlib (Go 1.21+, project is on 1.25.10)
- **Structured logging** — all log calls pass key=value pairs for JSON/searchability
- **Configurable log level** — `LOG_LEVEL` env var (debug/info/warn/error), default `info`
- **Text output in dev, JSON in prod** — controlled by `APP_ENV`
- **Correlation IDs** — UUID per HTTP request, propagated to async jobs and WebSocket messages
- **Gradual migration** — package-level global `logger.L` matches existing `config.DB` / `ws.Manager` singleton pattern
