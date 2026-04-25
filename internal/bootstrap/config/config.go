package config

import (
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	"github.com/muonsoft/errors"
)

// HTTP — конфигурация HTTP-сервера.
type HTTP struct {
	Addr              string `env:"KB_HTTP_ADDR" envDefault:":8080"`
	AllowedCORSOrigin string `env:"ALLOWED_CORS_ORIGIN" envDefault:""`
}

// LLM — конфигурация LLM-провайдера (OpenAI-совместимый API).
type LLM struct {
	APIURL string `env:"LLM_API_URL" envDefault:""`
	APIKey string `env:"LLM_API_KEY" envDefault:""`
	Model  string `env:"LLM_MODEL" envDefault:"gpt-4o"`
}

// IsConfigured возвращает true, если LLM-конфигурация задана.
func (l LLM) IsConfigured() bool {
	return l.APIKey != ""
}

// Telegram — конфигурация Telegram-бота.
type Telegram struct {
	Token   string `env:"TELEGRAM_TOKEN" envDefault:""`
	OwnerID int64  `env:"TELEGRAM_OWNER_ID" envDefault:"0"`
}

// Auth — опциональная сессионная авторизация: пароль (KB_LOGIN+KB_PASSWORD) или Google OAuth.
type Auth struct {
	Login    string `env:"KB_LOGIN" envDefault:""`
	Password string `env:"KB_PASSWORD" envDefault:""`
	// Google OAuth (all required when enabling Google; mutually exclusive with password).
	GoogleClientID     string        `env:"KB_GOOGLE_OAUTH_CLIENT_ID" envDefault:""`
	GoogleClientSecret string        `env:"KB_GOOGLE_OAUTH_CLIENT_SECRET" envDefault:""`
	GoogleRedirectURL  string        `env:"KB_GOOGLE_OAUTH_REDIRECT_URL" envDefault:""`
	AuthAllowedEmails  string        `env:"KB_AUTH_ALLOWED_EMAILS" envDefault:""` // comma-separated
	OAuthStateSecret   string        `env:"KB_OAUTH_STATE_SECRET" envDefault:""`
	SessionTTL         time.Duration `env:"KB_SESSION_TTL" envDefault:"8h"`
}

// PasswordAuthConfigured reports full password mode fields (both login and password).
func (a Auth) PasswordAuthConfigured() bool {
	return a.Login != "" && a.Password != ""
}

// GoogleAuthConfigured reports a complete Google OAuth env set (incl. non-empty allowlist).
func (a Auth) GoogleAuthConfigured() bool {
	return a.GoogleClientID != "" &&
		a.GoogleClientSecret != "" &&
		a.GoogleRedirectURL != "" &&
		a.OAuthStateSecret != "" &&
		strings.TrimSpace(a.AuthAllowedEmails) != ""
}

// AuthMode returns off, password, or google based on environment (call ValidateAuth at startup first).
func (a Auth) AuthMode() AuthMode {
	switch {
	case a.GoogleAuthConfigured():
		return AuthModeGoogle
	case a.PasswordAuthConfigured():
		return AuthModePassword
	default:
		return AuthModeOff
	}
}

// AuthEnabled returns true for password or Google session mode.
func (a Auth) AuthEnabled() bool {
	return a.AuthMode() != AuthModeOff
}

// ValidateAuth enforces mutual exclusion, complete groups, and allowlist rules.
func (a Auth) ValidateAuth() error {
	hasPartialPassword := (a.Login != "" || a.Password != "") && !a.PasswordAuthConfigured()
	if hasPartialPassword {
		return errors.New("auth: set both KB_LOGIN and KB_PASSWORD, or clear both for Google OAuth")
	}
	if a.anyGoogleEnvSet() && !a.GoogleAuthConfigured() {
		return errors.New("auth: incomplete Google OAuth env — set KB_GOOGLE_OAUTH_CLIENT_ID, KB_GOOGLE_OAUTH_CLIENT_SECRET, KB_GOOGLE_OAUTH_REDIRECT_URL, KB_OAUTH_STATE_SECRET, and non-empty KB_AUTH_ALLOWED_EMAILS, or clear all")
	}
	if a.GoogleAuthConfigured() && a.PasswordAuthConfigured() {
		return errors.New("auth: password mode (KB_LOGIN/KB_PASSWORD) and Google OAuth are mutually exclusive — remove one set of variables")
	}
	if a.GoogleAuthConfigured() && (a.Login != "" || a.Password != "") {
		return errors.New("auth: Google mode requires empty KB_LOGIN and KB_PASSWORD")
	}

	return nil
}

// ValidateWebPublicBaseForGoogle returns an error in Google mode when WebPublicBaseURL is missing.
func (a Auth) ValidateWebPublicBaseForGoogle(webPublicBaseURL string) error {
	if !a.GoogleAuthConfigured() {
		return nil
	}
	if strings.TrimSpace(webPublicBaseURL) == "" {
		return errors.New("auth: KB_PUBLIC_WEB_BASE_URL is required in Google OAuth mode (post-login redirect)")
	}

	return nil
}

func (a Auth) anyGoogleEnvSet() bool {
	if a.GoogleClientID != "" || a.GoogleClientSecret != "" || a.GoogleRedirectURL != "" ||
		a.OAuthStateSecret != "" || strings.TrimSpace(a.AuthAllowedEmails) != "" {
		return true
	}

	return false
}

// Config — конфигурация приложения.
type Config struct {
	DataPath   string `env:"KB_DATA_PATH" envDefault:""`
	UploadsDir string `env:"KB_UPLOADS_DIR" envDefault:""`
	// WebPublicBaseURL — публичный базовый URL веб-интерфейса (без завершающего /), для ссылок в ответах Telegram.
	WebPublicBaseURL string        `env:"KB_PUBLIC_WEB_BASE_URL" envDefault:""`
	JinaAPIKey       string        `env:"JINA_API_KEY" envDefault:""`
	GitDisabled      bool          `env:"KB_GIT_DISABLED" envDefault:"false"`
	GitSyncInterval  time.Duration `env:"GIT_SYNC_INTERVAL" envDefault:"5m"`
	AutoTranslate    bool          `env:"KB_AUTO_TRANSLATE" envDefault:"true"`
	// IngestExpandURLs — после LLM раскрывать http(s) в теле и annotation (короткие ссылки, UTM).
	IngestExpandURLs bool `env:"KB_INGEST_EXPAND_URLS" envDefault:"true"`

	HTTP     HTTP
	LLM      LLM
	Telegram Telegram
	Auth     Auth
}

// Load загружает конфигурацию из .env и переменных окружения.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
