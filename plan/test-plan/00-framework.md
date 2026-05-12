# 00 — Test Framework Decision

**Detail file**: `plan/test-plan/00-framework.md`
**Parent**: [index.md](index.md)

---

**`testing` + `httptest` + `testify`**

| Component | Purpose |
|-----------|---------|
| `testing` (stdlib) | Test runner, `TestMain`, benchmarks |
| `net/http/httptest` (stdlib) | HTTP test server/recorder for Gin integration tests |
| `testify/assert` | Clean assertions (`assert.Equal`, `assert.NoError`, `assert.JSONEq`, etc.) |
| `testify/require` | Fatal-on-fail assertions for setup code |

No mocking framework needed — the codebase already uses **repository interfaces** (`UserRepository`, `RefreshTokenRepository`, `SharingRepository`, `CloudConfigRepository`), enabling clean integration tests with a **real SQLite :memory: database** and a **real filesystem temp dir**.

