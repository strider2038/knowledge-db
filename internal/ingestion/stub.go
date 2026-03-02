package ingestion

import (
	"context"
	"errors"

	"github.com/strider2038/knowledge-db/internal/kb"
)

var ErrNotImplemented = errors.New("ingestion not implemented")

// StubIngester — заглушка, возвращает ErrNotImplemented.
type StubIngester struct{}

// IngestText возвращает ErrNotImplemented.
func (s *StubIngester) IngestText(ctx context.Context, text string) (*kb.Node, error) {
	return nil, ErrNotImplemented
}

// IngestURL возвращает ErrNotImplemented.
func (s *StubIngester) IngestURL(ctx context.Context, url string) (*kb.Node, error) {
	return nil, ErrNotImplemented
}
