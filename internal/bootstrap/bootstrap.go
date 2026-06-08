package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/muonsoft/errors"
	"github.com/pior/runnable"
	"github.com/spf13/afero"

	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/auth"
	"github.com/strider2038/knowledge-db/internal/auth/session"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/chat"
	sqlitechat "github.com/strider2038/knowledge-db/internal/chat/sqlite"
	"github.com/strider2038/knowledge-db/internal/debugdata"
	"github.com/strider2038/knowledge-db/internal/index"
	sqliteindex "github.com/strider2038/knowledge-db/internal/index/sqlite"
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
	if err := cfg.Auth.ValidateWebPublicBaseForOAuth(cfg.WebPublicBaseURL); err != nil {
		return errors.Errorf("invalid auth configuration: %w", err)
	}
	if err := cfg.Embedding.Validate(); err != nil {
		return errors.Errorf("invalid embedding configuration: %w", err)
	}
	if err := config.ValidateLogLevel(cfg.LogLevel); err != nil {
		return err
	}

	opts := &slog.HandlerOptions{
		Level: config.ParseLogLevel(cfg.LogLevel),
	}
	logHandler := slog.NewJSONHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(logHandler))

	slog.Info("kb: starting server", "addr", cfg.HTTP.Addr, "data_path", cfg.DataPath)

	committer := buildCommitter(cfg)
	translationQueue := buildTranslationQueue(cfg)

	var indexStore index.Store
	var syncWorker *index.SyncWorker
	var embeddingProvider index.EmbeddingProvider
	indexStore, syncWorker, embeddingProvider = buildIndexComponents(context.Background(), cfg)
	ingester, translationWorker := buildIngester(cfg, committer, translationQueue, indexStore)
	wireIndexNodeNotifications(syncWorker, ingester, translationWorker)

	apiHandler := api.NewHandlerWithUploads(cfg.DataPath, cfg.UploadsDir, ingester, translationQueue)
	apiHandler.SetNodeNormalizer(api.NewCursorNodeNormalizer())
	apiHandler.SetNodeAgentEditor(api.NewCursorNodeAgentEditor())
	debugStore := debugdata.NewStore(cfg.DataPath)
	apiHandler.SetDebugIssueStore(debugStore)
	chatStore := buildChatStore(cfg)
	if chatStore != nil {
		defer func() { _ = chatStore.Close() }()
		apiHandler.SetChatStore(chatStore)
	}

	commitMsgGen := igit.NewCommitMessageGenerator(cfg.LLM.APIKey, cfg.LLM.APIURL, cfg.LLM.Model)
	if !cfg.LLM.IsConfigured() {
		commitMsgGen = nil
	}
	apiHandler.SetGitCommitter(committer, commitMsgGen, cfg.GitDisabled)
	apiHandler.SetIndexComponents(indexStore, syncWorker, embeddingProvider, cfg.Embedding)

	sessionStore := session.NewStore()
	authHandler := api.NewAuthHandler(sessionStore, cfg)

	mux, err := api.NewMux(apiHandler, authHandler)
	if err != nil {
		return errors.Errorf("new mux: %w", err)
	}

	if cfg.MCPEnabled() {
		mcpHandler := mcp.NewHandler(cfg.MCPAPIKey, indexStore, embeddingProvider)
		mux.Handle("GET /api/mcp", mcpHandler)
		mux.Handle("POST /api/mcp", mcpHandler)
	} else {
		slog.Info("mcp endpoint disabled: KB_MCP_API_KEY is empty")
	}
	if cfg.MCPDebugEnabled() {
		debugMCPHandler := mcp.NewDebugHandler(cfg.MCPDebugAPIKey, debugStore)
		mux.Handle("GET /api/mcp/debug", debugMCPHandler)
		mux.Handle("POST /api/mcp/debug", debugMCPHandler)
	} else {
		slog.Info("debug mcp endpoint disabled: KB_MCP_DEBUG_API_KEY is empty")
	}

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
		bot := telegram.NewBot(
			cfg.Telegram.Token,
			cfg.Telegram.OwnerID,
			ingester,
			cfg.WebPublicBaseURL,
			debugStore,
			cfg.TelegramRawLogEnabled,
		)
		m.Register(bot)
	}

	if !cfg.GitDisabled && cfg.LLM.IsConfigured() {
		var onGitSynced func(context.Context)
		if syncWorker != nil {
			onGitSynced = func(context.Context) {
				syncWorker.Send(index.GitSyncDiffEvent{})
			}
		}
		syncRunner := igit.NewGitSyncRunner(committer, cfg.GitSyncInterval, onGitSynced)
		m.Register(syncRunner)
	}
	if translationWorker != nil {
		m.Register(translationWorker)
	}
	if syncWorker != nil {
		m.Register(syncWorker)
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

func buildIngester(
	cfg *config.Config,
	committer igit.GitCommitter,
	translationQueue *translationqueue.Queue,
	indexStore index.Store,
) (ingestion.Ingester, *translationworker.Worker) {
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
	pipeline.SetPlacementIndexStore(indexStore)

	var worker *translationworker.Worker
	if translationQueue != nil {
		worker = translationworker.New(translationQueue, store, translator, committer, cfg.DataPath)
	}

	return pipeline, worker
}

func wireIndexNodeNotifications(syncWorker *index.SyncWorker, ingester ingestion.Ingester, translationWorker *translationworker.Worker) {
	if syncWorker == nil {
		return
	}
	pipeline, ok := ingester.(*ingestion.PipelineIngester)
	if !ok {
		return
	}
	pipeline.SetNodesChangedNotifier(func(_ context.Context, paths ...string) {
		for _, path := range paths {
			syncWorker.Send(index.SingleNodeEvent{Path: path})
		}
	})
	if translationWorker == nil {
		return
	}
	translationWorker.SetOnNodesChanged(func(originalPath, translationPath string) {
		syncWorker.Send(index.SingleNodeEvent{Path: originalPath})
		syncWorker.Send(index.SingleNodeEvent{Path: translationPath})
	})
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

func buildIndexComponents(ctx context.Context, cfg *config.Config) (index.Store, *index.SyncWorker, index.EmbeddingProvider) {
	kbDir := filepath.Join(cfg.DataPath, ".kb")
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		slog.Error("failed to create .kb directory", "error", err)

		return nil, nil, nil
	}

	dbPath := filepath.Join(kbDir, "index.db")
	store, err := sqliteindex.NewStore(ctx, dbPath)
	if err != nil {
		slog.Error("failed to open index database", "error", err)

		return nil, nil, nil
	}

	if !cfg.Embedding.IsConfigured() {
		return store, nil, nil
	}

	provider := index.NewAPIProvider(cfg.Embedding.APIURL, cfg.Embedding.APIKey, cfg.Embedding.Model)
	worker := index.NewSyncWorker(store, provider, cfg.DataPath, cfg.Embedding.Model, cfg.Embedding.RateLimit)

	return store, worker, provider
}

func buildChatStore(cfg *config.Config) chat.Store {
	kbDir := filepath.Join(cfg.DataPath, ".kb")
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		slog.Error("failed to create .kb directory for chat store", "error", err)

		return nil
	}
	dbPath := filepath.Join(kbDir, "chat.db")
	store, err := sqlitechat.NewStore(dbPath)
	if err != nil {
		slog.Error("failed to open chat database", "error", err)

		return nil
	}

	return store
}
