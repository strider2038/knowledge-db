package fetcher

import (
	"context"
	stderrors "errors"

	"github.com/muonsoft/errors"
)

// ErrURLMetaNotSupported означает, что fetcher не поддерживает URL.
var ErrURLMetaNotSupported = stderrors.New("url meta not supported")

// URLMetaFetcher извлекает лёгкие метаданные страницы.
type URLMetaFetcher interface {
	FetchMeta(ctx context.Context, rawURL string) (*URLMeta, error)
}

// ChainURLMetaFetcher пробует meta-fetchers по порядку, пока не получит успешный результат.
type ChainURLMetaFetcher struct {
	fetchers []URLMetaFetcher
}

// NewChainURLMetaFetcher создаёт цепочку meta-fetchers.
func NewChainURLMetaFetcher(fetchers ...URLMetaFetcher) *ChainURLMetaFetcher {
	return &ChainURLMetaFetcher{fetchers: fetchers}
}

// FetchMeta возвращает результат первого успешного fetcher.
func (c *ChainURLMetaFetcher) FetchMeta(ctx context.Context, rawURL string) (*URLMeta, error) {
	var lastErr error
	for _, f := range c.fetchers {
		if f == nil {
			continue
		}

		meta, err := f.FetchMeta(ctx, rawURL)
		if err == nil {
			return meta, nil
		}
		if stderrors.Is(err, ErrURLMetaNotSupported) {
			continue
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, errors.Errorf("chain meta fetch: all fetchers failed: %w", lastErr)
	}

	return nil, errors.New("chain meta fetch: no fetchers configured")
}
