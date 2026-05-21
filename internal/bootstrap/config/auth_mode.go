package config

// AuthMode describes web auth for the session API: off, a single method (password, google, yandex), or multi.
type AuthMode string

const (
	// AuthModeOff is open access (no session required).
	AuthModeOff AuthMode = "off"
	// AuthModePassword uses KB_LOGIN and KB_PASSWORD.
	AuthModePassword AuthMode = "password"
	// AuthModeGoogle uses Google OAuth 2.0 and KB_AUTH_ALLOWED_EMAILS.
	AuthModeGoogle AuthMode = "google"
	// AuthModeYandex uses Yandex OAuth and KB_AUTH_ALLOWED_EMAILS.
	AuthModeYandex AuthMode = "yandex"
	// AuthModeMulti is returned when several auth methods are configured (deprecated field).
	AuthModeMulti AuthMode = "multi"
)
