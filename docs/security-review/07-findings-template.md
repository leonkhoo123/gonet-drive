# Security Findings Template

## Usage

Copy and fill out one entry per finding discovered during the security review.

---

## Finding: [ID] -- [Short Title]

**Severity:** CRITICAL / HIGH / MEDIUM / LOW / INFO

**Category:** Authentication / Authorization / Input Validation / Cryptography / Frontend / Infrastructure

**Shard:** [Number from 01-06]

**File(s):**
- `path/to/file.go:line_number`

**Description:**
[Detailed description of the vulnerability or issue]

**Impact:**
[What an attacker could achieve by exploiting this]

**Reproduction Steps:**
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Evidence:**
```
[Code snippet, request/response, or log output]
```

**Recommendation:**
[How to fix the issue]

**References:**
- [OWASP link or CWE number if applicable]

**Status:** Open / In Progress / Fixed / Accepted Risk / Won't Fix

---

## Severity Definitions

| Severity | Definition |
|----------|------------|
| **CRITICAL** | Direct access to sensitive data, full system compromise, or authentication bypass. Exploitable with low skill. |
| **HIGH** | Significant security impact, requires some access or conditions. Could lead to data exposure or privilege escalation. |
| **MEDIUM** | Moderate impact, requires specific conditions or user interaction. Defense-in-depth issue. |
| **LOW** | Minor impact, informational, or requires unlikely conditions. Hardening opportunity. |
| **INFO** | Best practice recommendation, no direct security impact. |

## Findings Summary Table

| ID | Severity | Title | Status |
|----|----------|-------|--------|
| F-001 | | | |
| F-002 | | | |
| F-003 | | | |
| F-004 | | | |
| F-005 | | | |
