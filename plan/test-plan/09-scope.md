# 09 — Scope Boundaries & Execution Order

**Detail file**: `plan/test-plan/09-scope.md`
**Parent**: [index.md](index.md)

---

## What We Are NOT Testing (Scope Boundaries)

These areas are intentionally excluded from the first pass:

1. **Video transcoding / ffmpeg** — requires ffmpeg binary, binary video fixtures, too slow for CI
2. **Photo thumbnail generation** — requires `imaging` + `webp` libs, visual output verification is complex
3. **WebSocket real-time messaging** — `gorilla/websocket` integration tests are stateful and race-prone; better done with a separate integration test suite that runs a real server
4. **AudioBook streaming** — MP3 parsing is edge-case heavy, requires real audio files
5. **Chunked file upload** — multipart form upload with large binary data is complex; covered by frontend e2e
6. **File download via ZIP streaming** — requires checking ZIP content integrity; medium complexity
7. **Document serving** — simple file read, low value
8. **Music streaming (byte-range)** — standard HTTP range request, well-tested by Go stdlib

**Clarification**: Basic file upload (small single-part files) and basic file download (plain GET) ARE included in Section 05. These test the upload/download controller wiring, path sanitization, auth enforcement, and basic file I/O without the complexity of chunked uploads or ZIP packaging.

---

## Estimated Effort by Section

| Section | Sessions | Complexity | Files Created |
|---------|----------|------------|---------------|
| 01. Foundations | 1 | Medium | 2–3 |
| 01.4 Repository Tests | 0.5 | Low | 3–4 |
| 02. Util Unit Tests | 1 | Low | 4 |
| 03. Auth Tests | 1.5 | High | 3 |
| 03.4 MFA Tests | 0.5 | Medium | 1 |
| 03.5 APP_JWT=OFF Tests | 0.5 | Low | 1 |
| 04. User/Admin Tests | 1 | Medium | 2 |
| 05. File Ops Tests | 2 | High | 1 |
| 06. Sharing Tests | 2 | High | 3 |
| 07. Security/Sanity | 1 | Medium | 2 |
| 08. CI Setup | 1 | Low | 2–3 |
| **Total** | **12–13 sessions** | | **~20 test files** |

---

## Session Execution Order

Recommended order (each session is self-contained):

```
Session 1  → 01.1 + 01.2 + 01.2a + 01.3  (testutil foundation + route wiring refactor)
Session 2  → 01.4 + 02.1 + 02.2 + 02.3 + 02.4 (repo tests + utility unit tests)
Session 3  → 03.1 + 03.2 + 03.3 + 03.4 + 03.5  (auth, MFA, bypass integration tests)
Session 4  → 04.1 + 04.2        (user & admin tests)
Session 5  → 05.1              (file operations + basic upload/download)
Session 6  → 06.1 + 06.2 + 06.3  (sharing tests)
Session 7  → 07.1 + 07.2 + 07.3  (security & sanity)
Session 8  → 08.1 + 08.2 + 08.3  (CI & runner setup)
```

**Dependencies**: Session 1 must come first (everything uses testutil). Session 01.4 (repos) can be done immediately after 01.2 (they share SetupTestDB). Sessions 03.5 (APP_JWT=OFF) depends on 01.2 and 03.1 (needs router). Session 08 is last (or can be done first independently). All other sessions can be done in any order after Session 1.

