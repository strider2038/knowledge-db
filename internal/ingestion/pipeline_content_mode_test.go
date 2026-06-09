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

func TestPipelineIngester_IngestText_WhenPasteArticleWithYouTubeURL_AutoMode_ExpectVerbatimBody(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	require.NoError(t, fs.MkdirAll(base, 0o755))
	store := kb.NewStore(fs)

	pasteBody := strings.Repeat("Hermes desktop talk transcript paragraph. ", 40) + "https://www.youtube.com/watch?v=EJm8Ka-gVOc"
	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"hermes"},
			Annotation: "Talk transcript",
			ThemePath:  "ai",
			Slug:       "hermes-desktop-obzory",
			Type:       "link",
			Content:    "short bookmark digest should not win",
			Title:      "Hermes Agent Desktop",
			SourceURL:  "https://www.youtube.com/watch?v=EJm8Ka-gVOc",
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{
		result: &fetcher.FetchResult{
			Title:   "YouTube",
			Content: "scraped youtube description",
		},
	}, &mockCommitter{}, base, false, false, nil, nil, nil)

	result, err := pipeline.IngestText(ctx, ingestion.IngestRequest{
		Text: pasteBody,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Node)
	assert.Equal(t, ingestion.ContentModeVerbatim, result.ContentMode)
	assert.Equal(t, pasteBody, result.Node.Content)
}

func TestPipelineIngester_IngestText_WhenTelegramLongForm_ExpectVerbatimBody(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	require.NoError(t, fs.MkdirAll(base, 0o755))
	store := kb.NewStore(fs)

	longBody := strings.Repeat("Gemma 4 local AI on 8GB VRAM benchmark paragraph. ", 30) +
		"https://github.com/example/gemma-4"
	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"gemma"},
			Annotation: "Digest rewrite",
			ThemePath:  "ai",
			Slug:       "gemma-4-local",
			Type:       "note",
			Content:    "conceptual digest should not win",
			Title:      "Gemma 4",
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

	result, err := pipeline.IngestText(ctx, ingestion.IngestRequest{
		Text:      longBody,
		SourceURL: "https://t.me/ai_channel/42",
	})
	require.NoError(t, err)
	require.NotNil(t, result.Node)
	assert.Equal(t, ingestion.ContentModeVerbatim, result.ContentMode)
	assert.Equal(t, longBody, result.Node.Content)
}

func TestPipelineIngester_IngestText_WhenForwardWithBody_ExpectNonEmptyContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	require.NoError(t, fs.MkdirAll(base, 0o755))
	store := kb.NewStore(fs)

	forwardBody := strings.Repeat("Security plugin for Claude discussion paragraph. ", 25) +
		"https://example.com/security-plugin"
	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"security"},
			Annotation: "Forwarded plugin note",
			ThemePath:  "security",
			Slug:       "claude-security-plugin",
			Type:       "note",
			Content:    "",
			Title:      "Security plugin",
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

	result, err := pipeline.IngestText(ctx, ingestion.IngestRequest{
		Text:      forwardBody,
		SourceURL: "https://t.me/security_channel/99",
	})
	require.NoError(t, err)
	require.NotNil(t, result.Node)
	assert.Equal(t, ingestion.ContentModeVerbatim, result.ContentMode)
	assert.NotEmpty(t, result.Node.Content)
	assert.Equal(t, forwardBody, result.Node.Content)
}

func TestPipelineIngester_IngestText_WhenURLOnlyBookmark_ExpectCompactBody(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	require.NoError(t, fs.MkdirAll(base, 0o755))
	store := kb.NewStore(fs)

	orc := &sequenceOrchestrator{
		results: []*llm.ProcessResult{
			{
				Keywords:   []string{"bookmark"},
				Annotation: "Unknown page",
				ThemePath:  "links",
				Slug:       "example-page",
				Type:       "link",
				Content:    "",
				Title:      "Example page",
				SourceURL:  "https://example.invalid/page",
			},
			{
				Keywords:   []string{"bookmark"},
				Annotation: "Unknown page",
				ThemePath:  "links",
				Slug:       "example-page",
				Type:       "link",
				Content:    "Compact bookmark body for example.invalid/page.",
				Title:      "Example page",
				SourceURL:  "https://example.invalid/page",
			},
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

	result, err := pipeline.IngestText(ctx, ingestion.IngestRequest{
		Text: "https://example.invalid/page",
	})
	require.NoError(t, err)
	require.NotNil(t, result.Node)
	assert.Equal(t, ingestion.ContentModeLinkBookmark, result.ContentMode)
	assert.Equal(t, "Compact bookmark body for example.invalid/page.", result.Node.Content)
}
