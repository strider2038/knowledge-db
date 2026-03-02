package ingestion

import (
	"context"

	"github.com/strider2038/knowledge-db/internal/kb"
)

// Ingester — интерфейс pipeline добавления записей в базу.
type Ingester interface {
	IngestText(ctx context.Context, text string) (*kb.Node, error)
	IngestURL(ctx context.Context, url string) (*kb.Node, error)
}
