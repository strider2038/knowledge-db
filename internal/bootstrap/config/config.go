package config

import (
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
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

// Auth — опциональная сессионная авторизация (включается при KB_LOGIN и KB_PASSWORD).
type Auth struct {
	Login      string        `env:"KB_LOGIN" envDefault:""`
	Password   string        `env:"KB_PASSWORD" envDefault:""`
	SessionTTL time.Duration `env:"KB_SESSION_TTL" envDefault:"8h"`
}

// AuthEnabled возвращает true, если авторизация включена (оба KB_LOGIN и KB_PASSWORD заданы).
func (a Auth) AuthEnabled() bool {
	return a.Login != "" && a.Password != ""
}

// Config — конфигурация приложения.
type Config struct {
	DataPath        string        `env:"KB_DATA_PATH" envDefault:""`
	UploadsDir      string        `env:"KB_UPLOADS_DIR" envDefault:""`
	// WebPublicBaseURL — публичный базовый URL веб-интерфейса (без завершающего /), для ссылок в ответах Telegram.
	WebPublicBaseURL string `env:"KB_PUBLIC_WEB_BASE_URL" envDefault:""`
	JinaAPIKey      string        `env:"JINA_API_KEY" envDefault:""`
	GitDisabled     bool          `env:"KB_GIT_DISABLED" envDefault:"false"`
	GitSyncInterval time.Duration `env:"GIT_SYNC_INTERVAL" envDefault:"5m"`
	AutoTranslate   bool          `env:"KB_AUTO_TRANSLATE" envDefault:"true"`
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
