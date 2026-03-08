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
}

// Ingester — интерфейс pipeline добавления записей в базу.
type Ingester interface {
	IngestText(ctx context.Context, req IngestRequest) (*kb.Node, error)
	IngestURL(ctx context.Context, url string) (*kb.Node, error)
}
