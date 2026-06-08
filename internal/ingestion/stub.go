package ingestion

import (
	"context"
	"errors"

	"github.com/strider2038/knowledge-db/internal/kb"
)

var (
	ErrNotImplemented       = errors.New("ingestion not implemented")
	ErrSourceURLRequired    = errors.New("source_url required")
	ErrDigestContentEmpty   = errors.New("digest content is empty for profiled link")
	ErrArticleContentEmpty  = errors.New("article content is empty after full fetch")
	ErrInvalidContentMode   = errors.New("invalid content_mode")
)

// StubIngester — заглушка, возвращает ErrNotImplemented.
type StubIngester struct{}

// IngestText возвращает ErrNotImplemented.
func (s *StubIngester) IngestText(ctx context.Context, req IngestRequest) (*IngestResult, error) {
	return nil, ErrNotImplemented
}

// IngestURL возвращает ErrNotImplemented.
func (s *StubIngester) IngestURL(ctx context.Context, url string) (*IngestResult, error) {
	return nil, ErrNotImplemented
}

func (s *StubIngester) RefreshDescription(ctx context.Context, path string) (*kb.Node, error) {
	return nil, ErrNotImplemented
}
