package fetcher

import (
	"context"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
)

// ChainFetcher последовательно пробует primary, при ошибке — fallback.
// Primary: JinaFetcher; fallback: ReadabilityFetcher.
type ChainFetcher struct {
	primary  ContentFetcher
	fallback ContentFetcher
}

// NewChainFetcher создаёт ChainFetcher.
func NewChainFetcher(primary, fallback ContentFetcher) *ChainFetcher {
	return &ChainFetcher{primary: primary, fallback: fallback}
}

// Fetch сначала вызывает primary; при ошибке логирует warning и вызывает fallback.
func (c *ChainFetcher) Fetch(ctx context.Context, url string) (*FetchResult, error) {
	result, err := c.primary.Fetch(ctx, url)
	if err == nil {
		return result, nil
	}

	clog.Warn(ctx, "primary fetcher failed, trying fallback", "error", err, "url", url)

	result, err = c.fallback.Fetch(ctx, url)
	if err != nil {
		return nil, errors.Errorf("chain fetch: all fetchers failed: %w", err)
	}

	return result, nil
}
