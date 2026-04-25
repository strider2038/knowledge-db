package config

// AuthMode describes how the web app authenticates: disabled, password, or Google OAuth.
type AuthMode string

const (
	// AuthModeOff is open access (no session required).
	AuthModeOff AuthMode = "off"
	// AuthModePassword uses KB_LOGIN and KB_PASSWORD.
	AuthModePassword AuthMode = "password"
	// AuthModeGoogle uses Google OAuth 2.0 and KB_AUTH_ALLOWED_EMAILS.
	AuthModeGoogle AuthMode = "google"
)
