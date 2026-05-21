package yandexoauth

// Config holds Yandex OAuth2 web client and allowlist (env-backed).
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	StateSecret  string
	// AllowedEmails is comma-separated emails (case-insensitive match).
	AllowedEmails string
}

// Endpoints overrides default Yandex hosts (for tests). Zero value uses production URLs.
type Endpoints struct {
	AuthURL, TokenURL, UserInfoURL string
}
