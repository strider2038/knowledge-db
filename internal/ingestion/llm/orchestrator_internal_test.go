package llm

import (
	"context"
	"encoding/json"
	"strings"
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
			result: &fetcher.FetchResult{Title: "Should not be used", Content: "Extra preview content"},
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
	assert.Equal(t, "content_fallback", payload["source"])
	assert.NotEmpty(t, payload["content_preview"])
	assert.Contains(t, payload["content_preview"], "Extra preview content")
}

func TestExecuteFetchURLMeta_WhenJinaReturnsImageNoise_ExpectCleanedPreview(t *testing.T) {
	t.Parallel()
	orch := &OpenAIOrchestrator{
		contentFetcher: &contentFetcherStub{
			result: &fetcher.FetchResult{
				Title: "Granola",
				Content: `## Helping the world's best product teams focus more on their meetings

![Image 1](https://www.granola.ai/customerLogos/posthog.svg)
![Image 2](https://www.granola.ai/customerLogos/intercom.svg)

## How it works
## How it works

Granola transcribes your computer's audio directly, with no meeting bots joining your call
Granola transcribes your computer's audio directly, with no meeting bots joining your call`,
			},
		},
		metaFetcher: &metaFetcherStub{
			meta: &fetcher.URLMeta{
				Title:       "Granola — The AI Notepad for back-to-back meetings",
				Description: "Meeting notes product",
			},
		},
	}

	out := orch.executeFetchURLMeta(context.Background(), `{"url":"https://www.granola.ai"}`)

	var payload map[string]string
	err := json.Unmarshal([]byte(out), &payload)
	require.NoError(t, err)
	require.Equal(t, "content_fallback", payload["source"])
	assert.NotContains(t, payload["content_preview"], "![Image")
	assert.Contains(t, payload["content_preview"], "## How it works")
	assert.Equal(t, 1, strings.Count(payload["content_preview"], "## How it works"))
	assert.Equal(
		t,
		1,
		strings.Count(payload["content_preview"], "Granola transcribes your computer's audio directly, with no meeting bots joining your call"),
	)
}
