package bootstrap

import (
	"context"
	"log/slog"
	"os"

	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/index"
)

// IndexRebuildResult — итог синхронной перестройки индекса.
type IndexRebuildResult struct {
	Status *index.IndexStatus
}

// RunIndexRebuild очищает index.db и заново индексирует все ноды из KB_DATA_PATH.
// Требует KB_EMBEDDING_ENABLED=true и настроенный embedding API (как для kb serve).
func RunIndexRebuild(ctx context.Context, dataPath string) (*IndexRebuildResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, errors.Errorf("load config: %w", err)
	}
	if dataPath != "" {
		cfg.DataPath = dataPath
	}
	if cfg.DataPath == "" {
		return nil, errors.New("KB_DATA_PATH is required (or pass --path)")
	}
	if err := cfg.Embedding.Validate(); err != nil {
		return nil, errors.Errorf("invalid embedding configuration: %w", err)
	}
	if !cfg.Embedding.IsConfigured() {
		return nil, errors.New("index rebuild requires KB_EMBEDDING_ENABLED=true with KB_EMBEDDING_API_URL and KB_EMBEDDING_API_KEY")
	}
	if err := config.ValidateLogLevel(cfg.LogLevel); err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{Level: config.ParseLogLevel(cfg.LogLevel)}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, opts)))

	store, worker, _ := buildIndexComponents(ctx, cfg)
	if store == nil {
		return nil, errors.New("index rebuild: failed to open index database")
	}
	defer func() { _ = store.Close() }()
	if worker == nil {
		return nil, errors.New("index rebuild: sync worker unavailable (check embedding config)")
	}

	if err := worker.ManualRebuild(ctx); err != nil {
		return nil, errors.Errorf("index rebuild: %w", err)
	}

	status, err := store.GetStatus(ctx, cfg.Embedding.Model)
	if err != nil {
		return nil, errors.Errorf("index status: %w", err)
	}

	return &IndexRebuildResult{Status: status}, nil
}
