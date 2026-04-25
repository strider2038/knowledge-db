package bootstrap

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/muonsoft/errors"
	"github.com/pior/runnable"
	"github.com/spf13/afero"

	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/auth"
	"github.com/strider2038/knowledge-db/internal/auth/session"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
	igit "github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/ingestion/translation"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationqueue"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationworker"
	"github.com/strider2038/knowledge-db/internal/kb"
	"github.com/strider2038/knowledge-db/internal/mcp"
	"github.com/strider2038/knowledge-db/internal/pkg/tracing"
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
	if err := cfg.Auth.ValidateAuth(); err != nil {
		return errors.Errorf("invalid auth configuration: %w", err)
	}
	if err := cfg.Auth.ValidateWebPublicBaseForGoogle(cfg.WebPublicBaseURL); err != nil {
		return errors.Errorf("invalid auth configuration: %w", err)
	}

	slog.Info("kb-server: starting", "addr", cfg.HTTP.Addr, "data_path", cfg.DataPath)

	committer := buildCommitter(cfg)
	translationQueue := buildTranslationQueue(cfg)
	ingester, translationWorker := buildIngester(cfg, committer, translationQueue)
	handler := api.NewHandlerWithUploads(cfg.DataPath, cfg.UploadsDir, ingester, translationQueue)

	sessionStore := session.NewStore()
	authHandler := api.NewAuthHandler(sessionStore, cfg)

	mux, err := api.NewMux(handler, authHandler)
	if err != nil {
		return errors.Errorf("new mux: %w", err)
	}

	mux.Handle("GET /api/mcp", mcp.NewHandler(cfg.DataPath))
	mux.Handle("POST /api/mcp", mcp.NewHandler(cfg.DataPath))

	baseHandler := api.Gzip(api.CORS(mux, cfg.HTTP.AllowedCORSOrigin))
	if cfg.Auth.AuthEnabled() {
		baseHandler = auth.Middleware(baseHandler, sessionStore)
	}
	httpHandler := tracing.Middleware(
		api.LoggingMiddleware(
			api.RequestLoggingMiddleware(baseHandler),
		),
	)

	srv := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: httpHandler,
	}

	runnable.SetLogger(slog.Default())
	m := runnable.Manager().ShutdownTimeout(30 * time.Second)
	m.Register(runnable.HTTPServer(srv).ShutdownTimeout(30 * time.Second))

	if cfg.Telegram.Token != "" {
		bot := telegram.NewBot(cfg.Telegram.Token, cfg.Telegram.OwnerID, ingester, cfg.WebPublicBaseURL)
		m.Register(bot)
	}

	if !cfg.GitDisabled && cfg.LLM.IsConfigured() {
		syncRunner := igit.NewGitSyncRunner(committer, cfg.GitSyncInterval)
		m.Register(syncRunner)
	}
	if translationWorker != nil {
		m.Register(translationWorker)
	}

	runnable.Run(m)

	return nil
}

func buildCommitter(cfg *config.Config) igit.GitCommitter {
	if cfg.GitDisabled {
		return &igit.NoopGitCommitter{}
	}
	exec := igit.NewExecGitCommitter(cfg.DataPath)

	return igit.NewSerializedGitCommitter(exec)
}

func buildTranslationQueue(cfg *config.Config) *translationqueue.Queue {
	if !cfg.LLM.IsConfigured() {
		return nil
	}

	return translationqueue.New(100)
}

func buildIngester(cfg *config.Config, committer igit.GitCommitter, translationQueue *translationqueue.Queue) (ingestion.Ingester, *translationworker.Worker) {
	if !cfg.LLM.IsConfigured() {
		slog.Warn("LLM configuration not set, ingestion pipeline disabled (using stub)")

		return &ingestion.StubIngester{}, nil
	}

	store := kb.NewStore(afero.NewOsFs())
	contentFetcher := buildContentFetcher(cfg)
	metaFetcher := buildMetaFetcher()
	orchestrator := llm.NewOpenAIOrchestratorWithMetaFetcher(cfg.LLM.APIKey, cfg.LLM.APIURL, cfg.LLM.Model, contentFetcher, metaFetcher)
	translator := translation.NewLLMTranslator(orchestrator)

	pipeline := ingestion.NewPipelineIngester(store, orchestrator, contentFetcher, committer, cfg.DataPath, cfg.AutoTranslate, cfg.IngestExpandURLs, translator, orchestrator, translationQueue)

	var worker *translationworker.Worker
	if translationQueue != nil {
		worker = translationworker.New(translationQueue, store, translator, committer, cfg.DataPath)
	}

	return pipeline, worker
}

func buildContentFetcher(cfg *config.Config) fetcher.ContentFetcher {
	jinaFetcher := fetcher.NewJinaFetcher(cfg.JinaAPIKey, nil)
	readabilityFetcher := fetcher.NewReadabilityFetcher(30 * time.Second)

	return fetcher.NewChainFetcher(jinaFetcher, readabilityFetcher)
}

func buildMetaFetcher() fetcher.URLMetaFetcher {
	return fetcher.NewChainURLMetaFetcher(
		fetcher.NewGitHubMetaFetcher(nil),
		fetcher.NewHTMLMetaFetcher(nil),
	)
}

func validateConfig(cfg *config.Config) error {
	if cfg.Telegram.Token != "" && cfg.Telegram.OwnerID == 0 {
		return errors.Errorf("TELEGRAM_OWNER_ID not set — all users can send messages to the bot")
	}

	return nil
}
