## Context

There is no checked-in application code in this workspace; repository checkout was not available. This design is stack-agnostic so it can be applied to the target service (for example the `aist` repo) when integrated. Google Sign-In for server-rendered apps and APIs typically uses OAuth 2.0 authorization code with optional PKCE; Google issues OIDC ID tokens and provides userinfo for email verification.

## Goals / Non-Goals

**Goals:**

- Use Google as the identity provider and treat Google’s verified `email` claim as the primary identifier for allowlisting.
- Keep the allowlist entirely in configuration (`GOOGLE_ALLOWED_EMAILS` or equivalent) so operators can add/remove users without redeploying application logic (only config reload or restart, depending on runtime).
- Minimize session fixation and CSRF risk on the OAuth dance (state parameter, secure cookies).
- Fail closed: unknown or unallowlisted users never receive a session.

**Non-Goals:**

- Full user directory, roles, or RBAC (can be layered later).
- Social login beyond Google.
- Self-service signup or invitations inside the app (operators edit env allowlist).

## Decisions

1. **Flow: authorization code on the server (BFF pattern)**  
   _Rationale:_ The server holds `client_secret`, performs code exchange, validates ID token or userinfo, then sets an HTTP-only session cookie. This avoids exposing secrets to browsers and simplifies token validation.  
   _Alternative:_ PKCE-only public client in SPA — acceptable if the product is SPA-only; then use PKCE and prefer backend-for-frontend for cookie session anyway.

2. **Allowlist format: comma-separated, case-insensitive**  
   Example: `GOOGLE_ALLOWED_EMAILS="alice@corp.com,bob@corp.com"`. Support optional newline splitting for `.env` readability. Normalize to lowercase before set membership.  
   _Alternative:_ JSON array in a single env var — clearer for complex tooling but heavier for operators; comma list is enough for small teams.

3. **Where to validate email:** Immediately after token/userinfo retrieval and before any session cookie is written.  
   _Rationale:_ Single gate; aligns with spec scenarios.

4. **Session mechanism:** Signed, HTTP-only, `Secure`, `SameSite=Lax` (or `Strict` if flows allow) session cookie referencing server-side session or signed JWT with short TTL — choose based on existing framework conventions in the target repo.

5. **Google client setup:** Create OAuth 2.0 Client (Web) in Google Cloud Console; authorized redirect URI must exactly match the app callback URL. Scopes: `openid email profile` (minimum `openid email`).

6. **Observability:** Log allowlist denials at `WARN` with email hashed or redacted if policy requires; never log `client_secret`.

## Risks / Trade-offs

- **[Risk] Allowlist drift** — Operators forget to remove departed users.  
  _Mitigation:_ Document periodic review; optional future work: external IdP groups.

- **[Risk] Email change on Google side** — Rare for Workspace; consumer Gmail could theoretically conflict.  
  _Mitigation:_ Prefer stable `sub` from ID token for session binding while still enforcing allowlist on `email` at login time.

- **[Risk] Misconfigured redirect URI** — Google rejects or users see Google error.  
  _Mitigation:_ Document exact URLs; validate config at startup.

- **[Trade-off] Env var size** — Very large lists are unwieldy. Acceptable for small teams; otherwise move to file or external store in a follow-up change.

## Migration Plan

1. Create Google OAuth client and set env vars in staging.  
2. Ship code behind feature flag or deploy with auth disabled until vars present.  
3. Enable auth in staging, verify allowlisted and non-allowlisted accounts.  
4. Production rollout with monitoring on auth errors.  
5. Rollback: remove/empty client credentials or disable route registration (feature flag).

## Open Questions

- Exact web framework in the target repository (affects middleware and cookie APIs).  
- Whether the product needs API keys for machine clients in addition to browser OAuth (out of scope unless specified).  
- Session store: in-memory vs Redis for multi-instance deployments.
