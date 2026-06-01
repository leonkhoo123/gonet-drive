# 01 — Config & Middleware (Gin + Correlation IDs)

> Parent: [logger_plan_index.md](logger_plan_index.md) | Status: implemented

---

## 1. New File: `internal/logger/logger.go`

Create the core logger package. Global `L` variable with safe default.

```go
package logger

import (
    "log/slog"
    "os"
)

var L = &Logger{Slog: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))}

type Logger struct {
    Slog *slog.Logger
}

func Init(level slog.Level, env string) {
    var handler slog.Handler
    opts := &slog.HandlerOptions{Level: level}
    if env == "dev" || env == "local" {
        handler = slog.NewTextHandler(os.Stderr, opts)
    } else {
        handler = slog.NewJSONHandler(os.Stderr, opts)
    }
    L = &Logger{Slog: slog.New(handler)}
}

func (l *Logger) Debug(msg string, keyvals ...any) { l.Slog.Debug(msg, keyvals...) }
func (l *Logger) Info(msg string, keyvals ...any)  { l.Slog.Info(msg, keyvals...) }
func (l *Logger) Warn(msg string, keyvals ...any)  { l.Slog.Warn(msg, keyvals...) }
func (l *Logger) Error(msg string, keyvals ...any) { l.Slog.Error(msg, keyvals...) }
func (l *Logger) Fatal(msg string, keyvals ...any) {
    l.Slog.Error(msg, keyvals...)
    os.Exit(1)
}
func (l *Logger) With(keyvals ...any) *Logger {
    return &Logger{Slog: l.Slog.With(keyvals...)}
}
```

**File path**: `internal/logger/logger.go`

---

## 2. New File: `internal/logger/gin.go`

Custom Gin middleware replacing `gin.Logger()`.

### 2a. Correlation ID Middleware

Injects a `request_id` into:
1. `c.Set("request_id", uuid)` — available to Gin handlers
2. `c.Writer.Header().Set("X-Request-Id", uuid)` — returned to client
3. Used as a field in all log calls during the request lifecycle

```go
func RequestIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := uuid.New().String()
        c.Set("request_id", requestID)
        c.Header("X-Request-Id", requestID)
        c.Next()
    }
}
```

### 2b. HTTP Request Logger Middleware

Replaces `gin.Logger()`. Logs method, path, status, latency, request_id.

```go
func RequestLoggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        latency := time.Since(start)
        requestID, _ := c.Get("request_id")
        L.Info("http request",
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "status", c.Writer.Status(),
            "latency", latency.String(),
            "client_ip", c.ClientIP(),
            "request_id", requestID,
        )
    }
}
```

---

## 3. Config Changes

### 3a. `internal/config/config.go`

Add `LogLevel` to `ServerConfig`:

```go
type ServerConfig struct {
    // ... existing fields ...
    LogLevel string  // NEW: debug, info, warn, error
}
```

Parse in `Load()`:

```go
LogLevel: getEnv("LOG_LEVEL", "info"),
```

### 3b. `cmd/main.go`

After `cfg := config.Load()`:

```go
// Initialize structured logger
logger.Init(parseLogLevel(cfg.Server.LogLevel), cfg.Server.AppEnv)
```

With a helper:

```go
func parseLogLevel(level string) slog.Level {
    switch strings.ToLower(level) {
    case "debug":
        return slog.LevelDebug
    case "warn", "warning":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

Replace `gin.Logger()` with:

```go
router.Use(logger.RequestIDMiddleware())
router.Use(logger.RequestLoggerMiddleware())
router.Use(gin.Recovery())
```

### 3c. `.env`

Add:

```
LOG_LEVEL=debug
```

---

## 4. Startup Wiring in `cmd/main.go`

Replace all existing `log.*` calls:

| Line | Old | New |
|------|-----|-----|
| 125 | `log.Fatalf("failed to create sub filesystem...")` | `logger.L.Fatal("failed to create sub filesystem", "err", err)` |
| 154 | `log.Printf("Starting server on %s", ...)` | `logger.L.Info("starting server", "addr", cfg.Server.ListenAddr)` |
| 156 | `log.Fatalf("failed to start server: %v", err)` | `logger.L.Fatal("server start failed", "err", err)` |
| 164 | `log.Println("Shutting down...")` | `logger.L.Info("shutting down")` |
| 169 | `log.Fatalf("Server forced to shutdown...")` | `logger.L.Fatal("forced shutdown", "err", err)` |
| 172 | `log.Println("Server exiting gracefully")` | `logger.L.Info("server exited gracefully")` |

---

## 5. Correlation ID Propagation

### 5a. Through Gin Handlers

Handlers access: `c.GetString("request_id")`

### 5b. Through Async Job Queue

Currently `submitAsyncJob` passes `opID`. Add `request_id` to the job closure:

```go
func submitAsyncJob(opID, opType, opName string, tracker *util.ProgressTracker, includeSpeed bool, destDir string, action func(tracker *util.ProgressTracker) error, requestID string) {
    // ... log with requestID field ...
    logger.L.With("request_id", requestID).Info("job queued", "opId", opID)
}
```

### 5c. Through WebSocket Messages

Add `RequestId` field to `ws.OperationMessage` so the frontend can log it back, and correlate server-side logs with client-side actions.

---

## 6. Files Affected

| File | Type | Changes |
|------|------|---------|
| `internal/logger/logger.go` | **New** | Core logger package |
| `internal/logger/gin.go` | **New** | Gin middleware: request ID + request logger |
| `internal/config/config.go` | Modify | Add `LogLevel` field, parse `LOG_LEVEL` env var |
| `cmd/main.go` | Modify | Call `logger.Init()`, replace `gin.Logger()`, replace `log.*` calls |
| `.env` | Modify | Add `LOG_LEVEL=debug` |
| `internal/ws/manager.go` | Modify | `OperationMessage` struct: add `RequestId` field |
