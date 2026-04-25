package googleoauth

// Config holds Google OAuth2 web client and allowlist (env-backed).
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	StateSecret  string
	// AllowedEmails is comma-separated emails (case-insensitive match).
	AllowedEmails string
}

// Endpoints overrides default Google hosts (for tests). Zero value uses production URLs.
type Endpoints struct {
	AuthURL, TokenURL, UserInfoURL string
}
