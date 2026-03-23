package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
)

type metaFetcherStub struct {
	meta *fetcher.URLMeta
	err  error
}

func (m *metaFetcherStub) FetchMeta(_ context.Context, _ string) (*fetcher.URLMeta, error) {
	return m.meta, m.err
}

type contentFetcherStub struct {
	result *fetcher.FetchResult
	err    error
}

func (c *contentFetcherStub) Fetch(_ context.Context, _ string) (*fetcher.FetchResult, error) {
	return c.result, c.err
}

func TestExecuteFetchURLMeta_WhenLowQualityMeta_ExpectContentFallbackPreview(t *testing.T) {
	t.Parallel()
	orch := &OpenAIOrchestrator{
		contentFetcher: &contentFetcherStub{
			result: &fetcher.FetchResult{
				Title:   "Antfly",
				Content: "Antfly is a distributed search engine with BM25, vectors and graph traversal.",
			},
		},
		metaFetcher: &metaFetcherStub{
			meta: &fetcher.URLMeta{
				Title:       "GitHub - acme/antfly",
				Description: "Project hosted on GitHub",
			},
		},
	}

	out := orch.executeFetchURLMeta(context.Background(), `{"url":"https://github.com/acme/antfly"}`)

	var payload map[string]string
	err := json.Unmarshal([]byte(out), &payload)
	require.NoError(t, err)
	assert.Equal(t, "content_fallback", payload["source"])
	assert.Equal(t, "GitHub - acme/antfly", payload["title"])
	assert.NotEmpty(t, payload["content_preview"])
	assert.Contains(t, payload["content_preview"], "distributed search engine")
}

func TestExecuteFetchURLMeta_WhenHighQualityMeta_ExpectMetaSource(t *testing.T) {
	t.Parallel()
	orch := &OpenAIOrchestrator{
		contentFetcher: &contentFetcherStub{
			result: &fetcher.FetchResult{Title: "Should not be used", Content: "ignored"},
		},
		metaFetcher: &metaFetcherStub{
			meta: &fetcher.URLMeta{
				Title:       "GitHub - acme/antfly",
				Description: "Distributed search engine. Основной язык: Go. Темы: search, vector",
			},
		},
	}

	out := orch.executeFetchURLMeta(context.Background(), `{"url":"https://github.com/acme/antfly"}`)

	var payload map[string]string
	err := json.Unmarshal([]byte(out), &payload)
	require.NoError(t, err)
	assert.Equal(t, "github_api", payload["source"])
	assert.Empty(t, payload["content_preview"])
}
