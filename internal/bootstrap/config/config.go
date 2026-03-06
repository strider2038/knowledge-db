package config

import (
	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// HTTP — конфигурация HTTP-сервера.
type HTTP struct {
	Addr              string `env:"KB_HTTP_ADDR" envDefault:":8080"`
	AllowedCORSOrigin string `env:"ALLOWED_CORS_ORIGIN" envDefault:""`
}

// Telegram — конфигурация Telegram-бота.
type Telegram struct {
	Token string `env:"TELEGRAM_TOKEN" envDefault:""`
}

// Config — конфигурация приложения.
type Config struct {
	DataPath string `env:"KB_DATA_PATH" envDefault:""`

	HTTP     HTTP
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
