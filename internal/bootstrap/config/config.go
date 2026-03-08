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

// Config — конфигурация приложения.
type Config struct {
	DataPath        string        `env:"KB_DATA_PATH" envDefault:""`
	JinaAPIKey      string        `env:"JINA_API_KEY" envDefault:""`
	GitDisabled     bool          `env:"KB_GIT_DISABLED" envDefault:"false"`
	GitSyncInterval time.Duration `env:"GIT_SYNC_INTERVAL" envDefault:"5m"`
	AutoTranslate   bool          `env:"KB_AUTO_TRANSLATE" envDefault:"true"`

	HTTP     HTTP
	LLM      LLM
	Telegram Telegram
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
