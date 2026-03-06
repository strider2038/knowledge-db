package bootstrap

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/muonsoft/errors"
	"github.com/pior/runnable"

	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/mcp"
	"github.com/strider2038/knowledge-db/internal/telegram"
)

// Run запускает приложение: загружает конфиг, собирает зависимости, регистрирует сервисы и запускает runnable.Manager.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return errors.Errorf("load config: %w", err)
	}
	if cfg.DataPath == "" {
		return errors.New("KB_DATA_PATH is required")
	}

	ingester := &ingestion.StubIngester{}
	handler := api.NewHandler(cfg.DataPath, ingester)
	mux, err := api.NewMux(handler)
	if err != nil {
		return errors.Errorf("new mux: %w", err)
	}

	mux.Handle("GET /api/mcp", mcp.NewHandler(cfg.DataPath))
	mux.Handle("POST /api/mcp", mcp.NewHandler(cfg.DataPath))

	httpHandler := api.CORS(mux, cfg.HTTP.AllowedCORSOrigin)

	srv := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: httpHandler,
	}

	runnable.SetLogger(slog.Default())
	m := runnable.Manager().ShutdownTimeout(30 * time.Second)
	m.Register(runnable.HTTPServer(srv).ShutdownTimeout(30 * time.Second))
	if cfg.Telegram.Token != "" {
		bot := telegram.NewBot(cfg.Telegram.Token, cfg.DataPath, ingester)
		m.Register(bot)
	}

	runnable.Run(m)

	return nil
}
