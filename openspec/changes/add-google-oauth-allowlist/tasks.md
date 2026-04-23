## 1. Configuration and secrets

- [ ] 1.1 Add documented environment variables: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URL` (or derive from `PUBLIC_BASE_URL`), `GOOGLE_ALLOWED_EMAILS`
- [ ] 1.2 Implement parsing for `GOOGLE_ALLOWED_EMAILS` (comma/newline, trim, lowercase, ignore empties) with unit tests
- [ ] 1.3 Add startup validation for required vars when auth is enabled; fail fast or disable routes per product policy

## 2. OAuth routes and Google integration

- [ ] 2.1 Implement `GET /auth/google` (or framework equivalent) to redirect to Google with correct scopes, `state`, and `nonce` if using OIDC hybrid checks
- [ ] 2.2 Implement `GET /auth/google/callback` to exchange code, validate tokens/claims (`iss`, `aud`, `exp`), and read verified email
- [ ] 2.3 Map Google errors to safe user-facing messages without leaking internal details

## 3. Allowlist gate and session

- [ ] 3.1 After successful Google validation, check email against parsed allowlist; on failure, clear any partial context and show “access not permitted”
- [ ] 3.2 On success, create session bound to `sub` + email; set secure HTTP-only session cookie
- [ ] 3.3 Implement logout that revokes session server-side (if server-side store) or clears cookie

## 4. Middleware and UX

- [ ] 4.1 Add authentication middleware/guard for protected routes using session
- [ ] 4.2 Add minimal UI: sign-in button, post-login landing, error states for denied and misconfigured auth

## 5. Documentation and operations

- [ ] 5.1 Document Google Cloud Console steps (OAuth client, redirect URI, consent screen for internal/testing)
- [ ] 5.2 Document how to update `GOOGLE_ALLOWED_EMAILS` in each deployment environment
- [ ] 5.3 Add structured logging for allowlist denials and OAuth failures (redact secrets)
