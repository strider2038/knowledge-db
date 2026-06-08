package ingestion_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestPipelineIngester_IngestText_WhenPasteArticleWithYouTubeURL_ExpectVerbatimBody(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	require.NoError(t, fs.MkdirAll(base, 0o755))
	store := kb.NewStore(fs)

	pasteBody := strings.Repeat("Hermes desktop talk transcript paragraph. ", 40) + "https://www.youtube.com/watch?v=abc"
	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"hermes"},
			Annotation: "Talk transcript",
			ThemePath:  "ai",
			Slug:       "hermes-desktop",
			Type:       "article",
			Content:    "rewritten digest should not win",
			Title:      "Hermes Desktop",
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{
		result: &fetcher.FetchResult{
			Title:   "YouTube",
			Content: "scraped youtube description",
		},
	}, &mockCommitter{}, base, false, false, nil, nil, nil)

	result, err := pipeline.IngestText(ctx, ingestion.IngestRequest{
		Text:     pasteBody,
		TypeHint: "article",
	})
	require.NoError(t, err)
	node := result.Node
	require.NotNil(t, node)
	assert.Equal(t, ingestion.ContentModeVerbatim, result.ContentMode)
	assert.Equal(t, pasteBody, node.Content)
}
