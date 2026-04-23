## ADDED Requirements

### Requirement: Google OAuth sign-in entry point

The system SHALL expose a user-visible path to start Google OAuth (for example a “Sign in with Google” action) that initiates the standard OAuth 2.0 authorization code flow with Google as the authorization server.

#### Scenario: User starts login

- **WHEN** an unauthenticated user chooses to sign in with Google
- **THEN** the system redirects the user agent to Google’s authorization endpoint with a valid `client_id`, requested OIDC scopes including `openid email`, `redirect_uri` matching configured callback, and CSRF/state protection

### Requirement: OAuth callback handling

The system SHALL implement an HTTP callback endpoint registered with Google that exchanges the authorization code for tokens, validates the response, and retrieves stable user identity (at minimum a verified email address) from Google’s userinfo or ID token claims.

#### Scenario: Successful code exchange

- **WHEN** Google redirects to the callback URL with a valid authorization code and matching state
- **THEN** the system exchanges the code, validates issuer/audience (where applicable), and obtains a canonical lowercase email string for the authenticated Google account

#### Scenario: Invalid or tampered callback

- **WHEN** the callback receives missing code, invalid state, or token validation fails
- **THEN** the system does not create a session and returns or shows a generic authentication failure without leaking whether an email was allowlisted

### Requirement: Email allowlist from environment

The system SHALL load the set of permitted sign-in emails from a single environment-backed configuration value supplied at process start (for example `GOOGLE_ALLOWED_EMAILS`). Parsing SHALL treat entries as case-insensitive for comparison, trim surrounding whitespace, and ignore empty entries. The delimiter between addresses SHALL be documented (comma and/or newline).

#### Scenario: Allowlist contains multiple emails

- **WHEN** the environment variable lists `alice@example.com, bob@example.org` (with possible whitespace)
- **THEN** both addresses are recognized as permitted after normalization

### Requirement: Deny access when email is not allowlisted

After a successful Google authentication, the system SHALL grant an application session only if the user’s verified email is present in the allowlist. If the email is not allowlisted, the system MUST NOT issue a session.

#### Scenario: Allowlisted user signs in

- **WHEN** Google returns verified email `alice@example.com` and that address is in the allowlist
- **THEN** the system establishes an authenticated session bound to that identity

#### Scenario: Authenticated Google user is not allowlisted

- **WHEN** Google returns verified email `stranger@gmail.com` and that address is not in the allowlist
- **THEN** the system refuses access (no session), logs a structured audit event if logging is enabled, and presents a clear “access not permitted” outcome to the user

### Requirement: Session after successful allowlist check

The system SHALL bind the authenticated session to the allowlisted Google identity (email and stable subject identifier when available) and use that binding for subsequent authorization decisions until the session expires or the user signs out.

#### Scenario: Subsequent requests

- **WHEN** a user with a valid session makes authenticated requests
- **THEN** the system treats the request as authenticated for that identity until logout or session expiry

### Requirement: Required configuration and safe defaults

The system SHALL refuse to start OAuth flows if mandatory configuration is missing or invalid (for example empty `GOOGLE_CLIENT_ID`, missing callback base URL, or empty allowlist when enforcement is enabled). Secrets MUST be read from environment variables or a secret store, not committed to source control.

#### Scenario: Misconfiguration at startup

- **WHEN** required OAuth or allowlist variables are missing in a deployment where auth is enabled
- **THEN** the process fails fast with an explicit configuration error OR auth routes return a controlled “misconfigured” response without attempting Google redirect
