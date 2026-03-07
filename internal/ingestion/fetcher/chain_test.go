package fetcher_test

import (
	"context"
	"testing"

	"github.com/muonsoft/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
)

var (
	errJinaUnavailable   = errors.New("jina unavailable")
	errReadabilityFailed = errors.New("readability failed")
)

type mockFetcher struct {
	result *fetcher.FetchResult
	err    error
}

func (m *mockFetcher) Fetch(_ context.Context, _ string) (*fetcher.FetchResult, error) {
	return m.result, m.err
}

func TestChainFetcher_WhenPrimarySucceeds_ExpectPrimaryResult(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	primary := &mockFetcher{result: &fetcher.FetchResult{Title: "Primary", Content: "primary content"}}
	fallback := &mockFetcher{result: &fetcher.FetchResult{Title: "Fallback", Content: "fallback content"}}
	chain := fetcher.NewChainFetcher(primary, fallback)

	result, err := chain.Fetch(ctx, "https://example.com")

	require.NoError(t, err)
	assert.Equal(t, "Primary", result.Title)
}

func TestChainFetcher_WhenPrimaryFails_ExpectFallbackResult(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	primary := &mockFetcher{err: errJinaUnavailable}
	fallback := &mockFetcher{result: &fetcher.FetchResult{Title: "Fallback", Content: "fallback content"}}
	chain := fetcher.NewChainFetcher(primary, fallback)

	result, err := chain.Fetch(ctx, "https://example.com")

	require.NoError(t, err)
	assert.Equal(t, "Fallback", result.Title)
}

func TestChainFetcher_WhenBothFail_ExpectError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	primary := &mockFetcher{err: errJinaUnavailable}
	fallback := &mockFetcher{err: errReadabilityFailed}
	chain := fetcher.NewChainFetcher(primary, fallback)

	_, err := chain.Fetch(ctx, "https://example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all fetchers failed")
}
