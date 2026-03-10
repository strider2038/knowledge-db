package bootstrap

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/muonsoft/errors"
	"github.com/pior/runnable"
	"github.com/spf13/afero"

	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
	igit "github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/ingestion/translation"
	"github.com/strider2038/knowledge-db/internal/kb"
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

	if err := validateConfig(cfg); err != nil {
		return err
	}

	ingester := buildIngester(cfg)
	handler := api.NewHandlerWithUploads(cfg.DataPath, cfg.UploadsDir, ingester)
	mux, err := api.NewMux(handler)
	if err != nil {
		return errors.Errorf("new mux: %w", err)
	}

	mux.Handle("GET /api/mcp", mcp.NewHandler(cfg.DataPath))
	mux.Handle("POST /api/mcp", mcp.NewHandler(cfg.DataPath))

	httpHandler := api.Gzip(api.CORS(mux, cfg.HTTP.AllowedCORSOrigin))

	srv := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: httpHandler,
	}

	runnable.SetLogger(slog.Default())
	m := runnable.Manager().ShutdownTimeout(30 * time.Second)
	m.Register(runnable.HTTPServer(srv).ShutdownTimeout(30 * time.Second))

	if cfg.Telegram.Token != "" {
		bot := telegram.NewBot(cfg.Telegram.Token, cfg.Telegram.OwnerID, ingester)
		m.Register(bot)
	}

	if !cfg.GitDisabled && cfg.LLM.IsConfigured() {
		committer := igit.NewExecGitCommitter(cfg.DataPath)
		syncRunner := igit.NewGitSyncRunner(committer, cfg.GitSyncInterval)
		m.Register(syncRunner)
	}

	runnable.Run(m)

	return nil
}

func buildIngester(cfg *config.Config) ingestion.Ingester {
	if !cfg.LLM.IsConfigured() {
		slog.Warn("LLM configuration not set, ingestion pipeline disabled (using stub)")

		return &ingestion.StubIngester{}
	}

	store := kb.NewStore(afero.NewOsFs())
	contentFetcher := buildContentFetcher(cfg)
	orchestrator := llm.NewOpenAIOrchestrator(cfg.LLM.APIKey, cfg.LLM.APIURL, cfg.LLM.Model, contentFetcher)
	var committer igit.GitCommitter
	if cfg.GitDisabled {
		committer = &igit.NoopGitCommitter{}
	} else {
		committer = igit.NewExecGitCommitter(cfg.DataPath)
	}

	translator := translation.NewLLMTranslator(orchestrator)

	return ingestion.NewPipelineIngester(store, orchestrator, contentFetcher, committer, cfg.DataPath, cfg.AutoTranslate, translator, orchestrator)
}

func buildContentFetcher(cfg *config.Config) fetcher.ContentFetcher {
	jinaFetcher := fetcher.NewJinaFetcher(cfg.JinaAPIKey, nil)
	readabilityFetcher := fetcher.NewReadabilityFetcher(30 * time.Second)

	return fetcher.NewChainFetcher(jinaFetcher, readabilityFetcher)
}

func validateConfig(cfg *config.Config) error {
	if cfg.Telegram.Token != "" && cfg.Telegram.OwnerID == 0 {
		return errors.Errorf("TELEGRAM_OWNER_ID not set — all users can send messages to the bot")
	}

	return nil
}
