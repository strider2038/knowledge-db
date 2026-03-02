package api

import "github.com/caarlos0/env/v10"

// Config — конфигурация API из переменных окружения.
type Config struct {
	DataPath string `env:"KB_DATA_PATH" envDefault:""`
	Addr    string `env:"KB_HTTP_ADDR" envDefault:":8080"`
}

// LoadConfig загружает конфигурацию из env.
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
