# GoNet Drive — Production Test Plan Index

> **For AI assistants**: This is the master index. Each section below links to a detail file. When working on a section:
> 1. Read the detail file for full test cases and implementation specifics.
> 2. Mark `status` as `in-progress` when you start working, `done` when complete.
> 3. Update the relevant checklist and timestamps. Commit changes as you progress.

## Progress Legend

| Icon | Meaning |
|------|---------|
| *blank* | not started |
| ✅ | done |
| 🚧 | in-progress |
| ⏸️ | paused/blocked |
| ❌ | skipped/cancelled |

---

## Sections

| # | Section | Status | File | Started | Done |
|---|---------|--------|------|---------|------|
| 00 | Test Framework Decision | ✅ done | [00-framework.md](00-framework.md) | 2026-05-12 | 2026-05-12 |
| 01 | Foundations (testutil, route wiring, testify) | ✅ done | [01-foundations.md](01-foundations.md) | 2026-05-12 | 2026-05-12 |
| 02 | Utility Unit Tests (sanitize, JWT, cookies, HTTP) | ✅ done | [02-utility.md](02-utility.md) | 2026-05-12 | 2026-05-12 |
| 03 | Auth & Rate Limiting Integration Tests | ✅ done | [03-auth.md](03-auth.md) | 2026-05-12 | 2026-05-12 |
| 04 | User & Admin Management Tests | ✅ done | [04-user-admin.md](04-user-admin.md) | 2026-05-12 | 2026-05-12 |
| 05 | File Operations Tests | ✅ done | [05-file-ops.md](05-file-ops.md) | 2026-05-12 | 2026-05-13 |
| 06 | Sharing Tests | *not started* | [06-sharing.md](06-sharing.md) | — | — |
| 07 | Edge Cases, Security & Sanity | *not started* | [07-security.md](07-security.md) | — | — |
| 08 | CI & Test Runner Setup | *not started* | [08-ci.md](08-ci.md) | — | — |
| 09 | Scope Boundaries & Execution Order | *not started* | [09-scope.md](09-scope.md) | — | — |

---

## Quick Summary by Session

| Session | Sections | Est. Complexity |
|---------|----------|-----------------|
| 1 | 01 (foundations + route wiring refactor) | Medium |
| 2 | 01.4 (repo tests) + 02 (utility unit tests) | Low |
| 3 | 03 (auth, MFA, bypass integration tests) | High |
| 4 | 04 (user & admin tests) | Medium |
| 5 | 05 (file operations) | High |
| 6 | 06 (sharing tests) | High |
| 7 | 07 (security & sanity) | Medium |
| 8 | 08 + 09 (CI setup, scope review) | Low |

**Total**: ~12–13 sessions, ~20 test files.

---

## Dependency Graph

```
Session 1 (01) ──must come first──► everything else
         │
         ├──► Session 2 (01.4 + 02)
         ├──► Session 3 (03)
         ├──► Session 4 (04)
         ├──► Session 5 (05)
         ├──► Session 6 (06)
         ├──► Session 7 (07)
         └──► Session 8 (08)
```
Sessions 2–7 can be done in any order after Session 1. Session 8 is last (or can be done independently).

---

## Instructions for AI

### How to use this test plan

1. **Read the index first** to understand the overall structure and pick a section to work on.
2. **Open the detail file** linked in the table above. It contains:
   - Exact test case names and descriptions
   - File paths where tests should be created
   - Implementation notes and edge cases
3. **Work through tests** in the order listed within each section.
4. **Track progress** in this `index.md`:
   - Change `status` from `*not started*` to `🚧 in-progress` when you begin.
   - Update the `Started` and `Done` timestamps (use ISO format, e.g. `2026-05-12`).
   - When all tests in a section pass, mark `✅ done`.
5. **Run tests** for the section with `CGO_ENABLED=1 go test -v ./...` from the repo root after each major batch.
6. **Use existing conventions** from the codebase (import paths use `go-file-server/...`, repos have `SQLite` prefix, etc.).

### Progress tracking protocol

When marking a section complete, update:

1. The status column in the table above.
2. Optionally, add a checklist comment in the detail file itself (e.g. `<!-- [x] Step 1 -->`).
3. Ensure all tests pass before marking `done`.
