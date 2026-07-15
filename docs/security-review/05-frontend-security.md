# Shard 5: Frontend Security

## Scan Instructions

Review all frontend code for client-side security issues including XSS, token handling, sensitive data exposure, and debug information leakage.

## Files to Review

- `frontend/src/api/axiosLayer.tsx` -- Axios interceptors, token refresh
- `frontend/src/api/api-auth.tsx` -- Auth API calls
- `frontend/src/App.tsx` -- Route protection, auth event handling
- `frontend/src/components/auth/AuthGate.tsx` -- Auth gate component
- `frontend/src/pages/LoginPage.tsx` -- Login, MFA setup, recovery codes
- `frontend/src/pages/SetupPage.tsx` -- First-run setup
- `frontend/src/pages/AdminPage.tsx` -- Admin page
- `frontend/src/pages/ShareVerifyPage.tsx` -- Share verification
- `frontend/src/main.tsx` -- App entry, debug console
- `frontend/src/hooks/useFileManager/` -- File manager hooks

## Checklist

### Token Handling
- [ ] No JWTs stored in localStorage or sessionStorage
- [ ] Auth is cookie-based with `withCredentials: true`
- [ ] Token refresh uses request queuing (no thundering herd)
- [ ] Refresh loop prevention (`_retry` flag)
- [ ] Refresh is not attempted for login/refresh/logout endpoints

### XSS Prevention
- [ ] No `dangerouslySetInnerHTML` usage
- [ ] No `innerHTML` usage
- [ ] No `document.write` usage
- [ ] Error messages from server are rendered as text
- [ ] QR code URLs are validated (not `javascript:` URIs)
- [ ] Recovery codes are rendered as React text content

### Sensitive Data Exposure
- [ ] MFA secrets only held in component state (not persisted)
- [ ] Recovery codes only held in component state
- [ ] Share authority stored in sessionStorage is UI-only (server enforces)
- [ ] No API keys or secrets in frontend source code
- [ ] Debug console (`eruda`) is not loaded in production builds

### Auth State Management
- [ ] Protected routes are wrapped in `<AuthGate>`
- [ ] Admin page has client-side auth guard
- [ ] `auth:unauthorized` event triggers redirect to login
- [ ] Redirect loop prevention (exclude `/login`, `/setup`)
- [ ] Setup page redirects away when setup is complete

### Error Handling
- [ ] Error messages don't leak internal server details
- [ ] Network errors are handled gracefully
- [ ] 401 responses trigger token refresh
- [ ] 403 `mfa_setup_required` triggers MFA setup flow
- [ ] Console error statements are removed in production

## Prompt Questions

1. Is `withCredentials: true` set on the axios instance for cookie-based auth?
2. Does the token refresh interceptor correctly handle concurrent requests (queue pattern)?
3. Are there any `console.log`/`console.error` statements that leak info in production?
4. Is the `eruda` debug console gated behind a production profile check?
5. Are `/admin` and `/audio-book` routes wrapped in `<AuthGate>`?
6. Is the `auth:unauthorized` event listener correctly preventing redirect loops?
7. Are MFA recovery codes cleared from component state after acknowledgment?
