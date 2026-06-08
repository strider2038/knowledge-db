package ingestion

import (
	"context"

	"github.com/strider2038/knowledge-db/internal/kb"
)

// IngestRequest — входные данные для IngestText.
type IngestRequest struct {
	Text         string // обязателен
	SourceURL    string // опционально — метаданные источника
	SourceAuthor string // опционально
	TypeHint     string // опционально: auto, article, link, note
	ContentMode  string // опционально: auto, verbatim, full_fetch, digest, link_bookmark
	NodeID       string // опционально — обновить существующий узел по id
}

// IngestResult — результат ingest с resolved content mode.
type IngestResult struct {
	Node        *kb.Node
	ContentMode ContentMode
}

// Ingester — интерфейс pipeline добавления записей в базу.
type Ingester interface {
	IngestText(ctx context.Context, req IngestRequest) (*IngestResult, error)
	IngestURL(ctx context.Context, url string) (*IngestResult, error)
}

type DescriptionRefresher interface {
	RefreshDescription(ctx context.Context, path string) (*kb.Node, error)
}
