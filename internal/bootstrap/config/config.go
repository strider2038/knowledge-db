package config

import (
	"log/slog"
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

// Auth — опциональная сессионная авторизация: пароль, Google OAuth, Yandex OAuth (независимо по env).
type Auth struct {
	Login    string `env:"KB_LOGIN" envDefault:""`
	Password string `env:"KB_PASSWORD" envDefault:""`
	GoogleClientID     string        `env:"KB_GOOGLE_OAUTH_CLIENT_ID" envDefault:""`
	GoogleClientSecret string        `env:"KB_GOOGLE_OAUTH_CLIENT_SECRET" envDefault:""`
	GoogleRedirectURL  string        `env:"KB_GOOGLE_OAUTH_REDIRECT_URL" envDefault:""`
	YandexClientID     string        `env:"KB_YANDEX_OAUTH_CLIENT_ID" envDefault:""`
	YandexClientSecret string        `env:"KB_YANDEX_OAUTH_CLIENT_SECRET" envDefault:""`
	YandexRedirectURL  string        `env:"KB_YANDEX_OAUTH_REDIRECT_URL" envDefault:""`
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

// YandexAuthConfigured reports a complete Yandex OAuth env set (incl. non-empty allowlist and state secret).
func (a Auth) YandexAuthConfigured() bool {
	return a.YandexClientID != "" &&
		a.YandexClientSecret != "" &&
		a.YandexRedirectURL != "" &&
		a.OAuthStateSecret != "" &&
		strings.TrimSpace(a.AuthAllowedEmails) != ""
}

// AuthMethods returns configured sign-in methods in fixed order: password, google, yandex.
func (a Auth) AuthMethods() []string {
	var methods []string
	if a.PasswordAuthConfigured() {
		methods = append(methods, string(AuthModePassword))
	}
	if a.GoogleAuthConfigured() {
		methods = append(methods, string(AuthModeGoogle))
	}
	if a.YandexAuthConfigured() {
		methods = append(methods, string(AuthModeYandex))
	}

	return methods
}

// AuthMode returns off, a single method name, or multi for backward-compatible session API.
func (a Auth) AuthMode() AuthMode {
	methods := a.AuthMethods()
	switch len(methods) {
	case 0:
		return AuthModeOff
	case 1:
		return AuthMode(methods[0])
	default:
		return AuthModeMulti
	}
}

// AuthEnabled returns true when at least one auth method is configured.
func (a Auth) AuthEnabled() bool {
	return len(a.AuthMethods()) > 0
}

// ValidateAuth enforces complete groups per method and shared OAuth env rules.
func (a Auth) ValidateAuth() error {
	hasPartialPassword := (a.Login != "" || a.Password != "") && !a.PasswordAuthConfigured()
	if hasPartialPassword {
		return errors.New("auth: set both KB_LOGIN and KB_PASSWORD, or clear both")
	}
	if a.anyGoogleSpecificEnvSet() && !a.GoogleAuthConfigured() {
		return errors.New("auth: incomplete Google OAuth env — set KB_GOOGLE_OAUTH_CLIENT_ID, KB_GOOGLE_OAUTH_CLIENT_SECRET, KB_GOOGLE_OAUTH_REDIRECT_URL, KB_OAUTH_STATE_SECRET, and non-empty KB_AUTH_ALLOWED_EMAILS, or clear all Google-specific variables")
	}
	if a.anyYandexEnvSet() && !a.YandexAuthConfigured() {
		return errors.New("auth: incomplete Yandex OAuth env — set KB_YANDEX_OAUTH_CLIENT_ID, KB_YANDEX_OAUTH_CLIENT_SECRET, KB_YANDEX_OAUTH_REDIRECT_URL, KB_OAUTH_STATE_SECRET, and non-empty KB_AUTH_ALLOWED_EMAILS, or clear all Yandex-specific variables")
	}
	if a.anyOAuthCommonEnvSet() && !a.GoogleAuthConfigured() && !a.YandexAuthConfigured() {
		return errors.New("auth: KB_OAUTH_STATE_SECRET and/or KB_AUTH_ALLOWED_EMAILS set but no full OAuth provider (Google or Yandex) configured")
	}

	return nil
}

// ValidateWebPublicBaseForOAuth returns an error when any OAuth provider is configured without KB_PUBLIC_WEB_BASE_URL.
func (a Auth) ValidateWebPublicBaseForOAuth(webPublicBaseURL string) error {
	if !a.GoogleAuthConfigured() && !a.YandexAuthConfigured() {
		return nil
	}
	if strings.TrimSpace(webPublicBaseURL) == "" {
		return errors.New("auth: KB_PUBLIC_WEB_BASE_URL is required when OAuth is enabled (post-login redirect)")
	}

	return nil
}

// ValidateWebPublicBaseForGoogle is deprecated; use ValidateWebPublicBaseForOAuth.
func (a Auth) ValidateWebPublicBaseForGoogle(webPublicBaseURL string) error {
	return a.ValidateWebPublicBaseForOAuth(webPublicBaseURL)
}

func (a Auth) anyGoogleSpecificEnvSet() bool {
	return a.GoogleClientID != "" || a.GoogleClientSecret != "" || a.GoogleRedirectURL != ""
}

func (a Auth) anyYandexEnvSet() bool {
	return a.YandexClientID != "" || a.YandexClientSecret != "" || a.YandexRedirectURL != ""
}

func (a Auth) anyOAuthCommonEnvSet() bool {
	return a.OAuthStateSecret != "" || strings.TrimSpace(a.AuthAllowedEmails) != ""
}

// Embedding — конфигурация эмбеддингов и RAG (опционально).
type Embedding struct {
	Enabled              bool          `env:"KB_EMBEDDING_ENABLED" envDefault:"false"`
	APIKey               string        `env:"KB_EMBEDDING_API_KEY" envDefault:""`
	APIURL               string        `env:"KB_EMBEDDING_API_URL" envDefault:""`
	Model                string        `env:"KB_EMBEDDING_MODEL" envDefault:"text-embedding-3-small"`
	ChatModel            string        `env:"KB_CHAT_MODEL" envDefault:""`
	ChatAPIURL           string        `env:"KB_CHAT_API_URL" envDefault:""`
	ChatAPIKey           string        `env:"KB_CHAT_API_KEY" envDefault:""`
	SearchRewriteEnabled bool          `env:"KB_SEARCH_REWRITE_ENABLED" envDefault:"true"`
	RateLimit            time.Duration `env:"KB_EMBEDDING_RATE_LIMIT" envDefault:"1s"`
}

// IsConfigured возвращает true, если эмбеддинги включены и ключ API задан.
func (e Embedding) IsConfigured() bool {
	return e.Enabled && e.APIKey != "" && e.APIURL != ""
}

// ChatAPIConfig возвращает URL и key для чата. Если ChatAPIURL задан — используется он, иначе APIURL.
func (e Embedding) ChatAPIConfig() (string, string) {
	if e.ChatAPIURL != "" {
		return e.ChatAPIURL, e.ChatAPIKey
	}

	return e.APIURL, e.APIKey
}

// Validate проверяет корректность конфигурации эмбеддингов.
func (e Embedding) Validate() error {
	if !e.Enabled {
		return nil
	}
	if e.APIKey == "" {
		return errors.New("embedding: KB_EMBEDDING_API_KEY is required when KB_EMBEDDING_ENABLED=true")
	}
	if e.APIURL == "" {
		return errors.New("embedding: KB_EMBEDDING_API_URL is required when KB_EMBEDDING_ENABLED=true")
	}
	if e.ChatModel != "" && e.ChatAPIURL != "" && e.ChatAPIKey == "" {
		return errors.New("embedding: KB_CHAT_API_KEY is required when KB_CHAT_API_URL is set")
	}

	return nil
}

var validLogLevels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}

func ValidateLogLevel(level string) error {
	if !validLogLevels[level] {
		return errors.New("log: invalid LOG_LEVEL — must be one of: debug, info, warn, error")
	}

	return nil
}

func ParseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Config — конфигурация приложения.
type Config struct {
	DataPath              string `env:"KB_DATA_PATH" envDefault:""`
	UploadsDir            string `env:"KB_UPLOADS_DIR" envDefault:""`
	MCPAPIKey             string `env:"KB_MCP_API_KEY" envDefault:""`
	MCPDebugAPIKey        string `env:"KB_MCP_DEBUG_API_KEY" envDefault:""`
	TelegramRawLogEnabled bool   `env:"KB_TELEGRAM_RAW_LOG_ENABLED" envDefault:"false"`
	// WebPublicBaseURL — публичный базовый URL веб-интерфейса (без завершающего /), для ссылок в ответах Telegram.
	WebPublicBaseURL string        `env:"KB_PUBLIC_WEB_BASE_URL" envDefault:""`
	JinaAPIKey       string        `env:"JINA_API_KEY" envDefault:""`
	GitDisabled      bool          `env:"KB_GIT_DISABLED" envDefault:"false"`
	GitSyncInterval  time.Duration `env:"GIT_SYNC_INTERVAL" envDefault:"5m"`
	AutoTranslate    bool          `env:"KB_AUTO_TRANSLATE" envDefault:"true"`
	// IngestExpandURLs — после LLM раскрывать http(s) в теле и annotation (короткие ссылки, UTM).
	IngestExpandURLs bool `env:"KB_INGEST_EXPAND_URLS" envDefault:"true"`
	// LogLevel — уровень логирования: debug, info, warn, error.
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	HTTP      HTTP
	LLM       LLM
	Telegram  Telegram
	Auth      Auth
	Embedding Embedding
}

// MCPEnabled returns true when MCP endpoint should be enabled.
func (c Config) MCPEnabled() bool {
	return strings.TrimSpace(c.MCPAPIKey) != ""
}

func (c Config) MCPDebugEnabled() bool {
	return strings.TrimSpace(c.MCPDebugAPIKey) != ""
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
