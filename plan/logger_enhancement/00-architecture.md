# 00 — Architecture & Decisions

> Parent: [logger_plan_index.md](logger_plan_index.md) | Status: implemented

---

## 1. Why `log/slog`

| Criteria | `log/slog` (stdlib) | `uber-go/zap` | `rs/zerolog` |
|----------|---------------------|---------------|--------------|
| New dependency | **No** (Go 1.21+) | Yes | Yes |
| Structured logging | ✅ JSON + Text handlers | ✅ | ✅ |
| Log levels | ✅ Debug/Info/Warn/Error | ✅ | ✅ |
| Context propagation | ✅ `slog.Handler` with `context.Context` | ✅ | ✅ |
| Performance | Good (alloc amortized) | Excellent | Excellent |
| Gin integration | Manual (~30 lines) | `zaphttp` | `gin-zerolog` |
| Migration effort | Same as any structured lib | Same | Same |

**Decision: `log/slog`** — no new dependency, already pre-installed (Go 1.25.10), sufficient performance for mid-scale NAS app.

---

## 2. Package Design: Singleton Pattern

The codebase uses global singletons: `config.DB`, `ws.Manager`, `state.GetProgress()`, etc. The logger follows the same pattern for minimal disruption.

```
internal/logger/
├── logger.go   — Logger struct, global L, Init(), helper methods
└── gin.go      — Gin middleware (replaces gin.Logger), correlation ID injector
```

### Core Types

```go
// internal/logger/logger.go

package logger

import "log/slog"

var L *Logger  // package-level global (nil until Init called)

type Logger struct {
    Slog *slog.Logger
}
```

### Initialization (called from `cmd/main.go`)

```go
func Init(level slog.Level, env string) {
    var handler slog.Handler
    opts := &slog.HandlerOptions{Level: level}
    if env == "dev" {
        handler = slog.NewTextHandler(os.Stderr, opts)
    } else {
        handler = slog.NewJSONHandler(os.Stderr, opts)
    }
    L = &Logger{Slog: slog.New(handler)}
}
```

### Method Signatures

All methods follow: `method(msg string, keyvals ...any)` — matching `slog` convention.

```go
func (l *Logger) Debug(msg string, keyvals ...any)  // calls l.Slog.Debug(msg, keyvals...)
func (l *Logger) Info(msg string, keyvals ...any)   // calls l.Slog.Info(msg, keyvals...)
func (l *Logger) Warn(msg string, keyvals ...any)   // calls l.Slog.Warn(msg, keyvals...)
func (l *Logger) Error(msg string, keyvals ...any)  // calls l.Slog.Error(msg, keyvals...)
func (l *Logger) Fatal(msg string, keyvals ...any)  // logs at Error + os.Exit(1)
```

### Sub-logger (for correlation IDs)

```go
// With creates a sub-logger with preset key=value pairs.
// Used to carry request_id, user, opId, etc.
func (l *Logger) With(keyvals ...any) *Logger {
    return &Logger{Slog: l.Slog.With(keyvals...)}
}
```

### Safer Logger — guard against nil

`L` may be nil before `Init`. All exported methods on the package-level wrapper must handle this:

```go
func Debug(msg string, keyvals ...any) {
    if L != nil {
        L.Slog.Debug(msg, keyvals...)
    }
}
```

**Actually better**: Make `Init` panic if already initialized, and keep `L` as a non-nil default (e.g., `slog.New(slog.NewTextHandler(os.Stderr, nil))`) so callers never need nil checks.

```go
var L = &Logger{Slog: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))}

func Init(level slog.Level, env string) {
    // ... create handler ...
    L = &Logger{Slog: slog.New(handler)}
}
```

This gives a safe default (Info level, text, stderr) even if `Init` is never called.

---

## 3. Log Level Classification

| Level | When to Use | Examples |
|-------|-------------|----------|
| **Debug** | Internal details useful for troubleshooting: ffmpeg commands, file paths per operation, progress ticks, chunk upload steps, duplicate path resolutions, thumbnail generation steps | `Debug("copying file", "src", src, "dst", dst)`, `Debug("ffmpeg executing", "cmd", cmd)` |
| **Info** | Normal lifecycle events: server start/stop, config loaded, operations started/completed, migrations applied, admin bootstrap, scheduler runs, WS connect/disconnect, HTTP requests | `Info("server started", "addr", addr)`, `Info("copy completed", "files", n, "bytes", b)` |
| **Warn** | Non-fatal anomalies: stat failures, temp file cleanup, protected folder skip, retries, orphan cleanup, config cache miss | `Warn("failed to stat path", "path", p, "err", err)` |
| **Error** | Recoverable failures: file operation errors, DB query errors, WS errors, share creation failures, upload errors | `Error("copy failed", "opId", id, "err", err)` |
| **Fatal** | Startup failures only: missing JWT secret, work dir not found, DB connection failed | `Fatal("missing JWT secret")` |

### Special Cases

- **Progress logs** (`Progress: %.2f%% ...`): Move to **Debug** level. These fire every 100–500ms and are too noisy for Info.
- **`fmt.Printf` warnings** in config: Convert to `Warn` level.
- **`log.Fatalf`** in non-startup code (e.g., `cmd/main.go:169` forced shutdown): Convert to `Error` + `os.Exit(1)` via helper.

---

## 4. Structured Field Naming Convention

Use lowercase snake_case keys. Common fields:

| Field | Example Value | Usage |
|-------|---------------|-------|
| `opId` | `"abc123"` | File operation ID |
| `source` | `"/videos/a.mp4"` | Source file path |
| `dest` | `"/backup/a.mp4"` | Destination path |
| `file` | `"/data/foo.txt"` | Single file path |
| `path` | `"/data/dir"` | Directory path |
| `size` | `1048576` | Size in bytes |
| `count` | `42` | Count of items |
| `duration` | `"2.4s"` | Duration string |
| `err` | error value | Error (slog auto-extracts) |
| `request_id` | `"req-abc123"` | Correlation ID |
| `user` | `"admin"` | Username |
| `addr` | `":3333"` | Listen address |
| `method` | `"GET"` | HTTP method |
| `path` | `"/api/files"` | HTTP path |
| `status` | `200` | HTTP status code |
| `latency` | `"1.2ms"` | Request duration |

---

## 5. Dev Output Format

When `APP_ENV=dev` (or `local`), use **text handler** to stdout:

```
2026-06-01T15:04:05.123+08:00 INFO server started addr=:3333
2026-06-01T15:04:06.456 DEBUG file operation sources=[/videos/a.mp4] destDir=/backup opId=abc123
2026-06-01T15:04:06.789 DEBUG ffmpeg executing cmd="/usr/bin/ffmpeg -i /tmp/x.mp4 ..." input=/videos/a.mp4
2026-06-01T15:04:08.901 INFO copy completed files=1 bytes=104857600 duration=2.4s
2026-06-01T15:04:09.123 INFO http request method=GET path=/api/files status=200 latency=1.2ms request_id=req-abc
2026-06-01T15:04:10.000 WARN failed to stat path path=/data/missing err=no such file
2026-06-01T15:04:10.100 ERROR copy failed opId=def456 err=operation canceled
```

When `APP_ENV=prod` (or anything else), use **JSON handler**:

```json
{"time":"2026-06-01T15:04:06.456+08:00","level":"DEBUG","msg":"file operation","sources":["/videos/a.mp4"],"destDir":"/backup","opId":"abc123"}
```

---

## 6. Non-Goals

- No log rotation (defer to systemd/Docker logging driver)
- No log aggregation (defer to external tools)
- No sampling or rate-limiting (simplicity over premature optimization)
- No context-aware logging from `context.Context` (can be added later if needed)
- No changing `gin.Recovery()` — keep it as-is for panic handling
