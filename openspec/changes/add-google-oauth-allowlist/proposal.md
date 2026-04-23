## Why

The product needs a practical way to let a small, known set of people sign in without managing passwords. Google OAuth provides familiar login while an explicit email allowlist in configuration keeps access limited to trusted addresses.

## What Changes

- Add Google (OIDC/OAuth 2.0) as the primary sign-in path for human users.
- After Google returns identity claims, the application MUST compare the verified email to a configured allowlist before establishing a session.
- Document required environment variables (Google client credentials, redirect URL, and allowlisted emails).
- No self-service registration: users not on the allowlist receive a clear denial without receiving an application session.

## Capabilities

### New Capabilities

- `google-oauth-allowlist`: End-to-end rules for Google-based authentication, email verification against an environment-defined allowlist, session establishment, and failure handling.

### Modified Capabilities

- _(none — greenfield capability)_

## Impact

- Authentication middleware, login/callback routes, and session issuance for the target application.
- New secrets and configuration in deployment (Google OAuth client, redirect URI, allowlist).
- Operational docs for rotating credentials and updating the allowlist without code changes.
