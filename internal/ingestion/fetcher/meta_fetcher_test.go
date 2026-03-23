//nolint:testpackage // tests need access to unexported test hooks.
package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMetaFetcher struct {
	meta *URLMeta
	err  error
}

func (s *stubMetaFetcher) FetchMeta(_ context.Context, _ string) (*URLMeta, error) {
	return s.meta, s.err
}

func TestHTMLMetaFetcher_WhenTwitterMetaPresent_ExpectExtractedTitleAndDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><html><head>
<meta name="twitter:title" content="  Antfly   ">
<meta name="twitter:description" content=" Distributed   search   engine with vectors ">
</head><body><h1>ignored</h1></body></html>`))
	}))
	defer server.Close()

	f := NewHTMLMetaFetcher(server.Client())

	meta, err := f.FetchMeta(context.Background(), server.URL)

	require.NoError(t, err)
	assert.Equal(t, "Antfly", meta.Title)
	assert.Equal(t, "Distributed search engine with vectors", meta.Description)
}

func TestChainURLMetaFetcher_WhenPrimaryUnsupported_ExpectFallbackResult(t *testing.T) {
	t.Parallel()
	chain := NewChainURLMetaFetcher(
		&stubMetaFetcher{err: ErrURLMetaNotSupported},
		&stubMetaFetcher{meta: &URLMeta{Title: "Fallback", Description: "From fallback"}},
	)

	meta, err := chain.FetchMeta(context.Background(), "https://example.com")

	require.NoError(t, err)
	assert.Equal(t, "Fallback", meta.Title)
}

func TestGitHubMetaFetcher_WhenRepoURL_ExpectAPIDataInDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/acme/antfly", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"full_name":"acme/antfly",
			"description":"Distributed search engine",
			"homepage":"https://antfly.dev",
			"language":"Go",
			"topics":["search","vector","rag"]
		}`))
	}))
	defer server.Close()

	f := NewGitHubMetaFetcher(server.Client())
	f.apiBaseURL = server.URL

	meta, err := f.FetchMeta(context.Background(), "https://github.com/acme/antfly")

	require.NoError(t, err)
	assert.Equal(t, "GitHub - acme/antfly", meta.Title)
	assert.Contains(t, meta.Description, "Distributed search engine")
	assert.Contains(t, meta.Description, "Основной язык: Go")
	assert.Contains(t, meta.Description, "Темы: search, vector, rag")
}
