package ingestion_test

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/kb"
)

// mockOrchestrator — мок LLMOrchestrator.
type mockOrchestrator struct {
	result *llm.ProcessResult
	err    error
}

func (m *mockOrchestrator) Process(_ context.Context, _ llm.ProcessInput) (*llm.ProcessResult, error) {
	return m.result, m.err
}

// mockFetcher — мок ContentFetcher.
type mockFetcher struct {
	result *fetcher.FetchResult
	err    error
}

func (m *mockFetcher) Fetch(_ context.Context, _ string) (*fetcher.FetchResult, error) {
	return m.result, m.err
}

// mockCommitter — мок GitCommitter.
type mockCommitter struct {
	commitErr error
	syncErr   error
}

func (m *mockCommitter) CommitNode(_ context.Context, _, _ string) error { return m.commitErr }
func (m *mockCommitter) Sync(_ context.Context) error                    { return m.syncErr }

const testBasePath = "/data"

func TestPipelineIngester_IngestText_WhenSuccess_ExpectNodeCreated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"go", "concurrency"},
			Annotation: "Notes about Go concurrency",
			ThemePath:  "go/concurrency",
			Slug:       "goroutine-basics",
			Type:       "note",
			Content:    "# Goroutines\n\nBasic notes.",
			Title:      "Goroutine Basics",
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base)

	node, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "Notes about goroutines."})

	require.NoError(t, err)
	assert.Equal(t, "go/concurrency/goroutine-basics", node.Path)
	assert.Equal(t, "Notes about Go concurrency", node.Annotation)
	assert.Equal(t, "note", node.Metadata["type"])
}

func TestPipelineIngester_IngestText_WhenOrchestratorFails_ExpectError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll(testBasePath, 0o755)
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{err: ingestion.ErrNotImplemented}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, testBasePath)

	_, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "text"})

	require.Error(t, err)
}

func TestPipelineIngester_IngestText_WhenTitleEmpty_ExpectTitleFromSlug(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"knuth", "claude"},
			Annotation: "Он назвал статью Claude Cycles",
			ThemePath:  "ai",
			Slug:       "professor-donald-knuth-clause-cycles",
			Type:       "note",
			Content:    "Content.",
			Title:      "", // LLM не вернул title
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base)

	node, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "Notes about Claude Cycles."})

	require.NoError(t, err)
	assert.Equal(t, "Professor Donald Knuth Clause Cycles", node.Metadata["title"])
	assert.Equal(t, []string{"Professor Donald Knuth Clause Cycles"}, node.Metadata["aliases"])
}

func TestPipelineIngester_IngestURL_WhenFetchSuccess_ExpectNodeWithSourceURL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	f := &mockFetcher{
		result: &fetcher.FetchResult{
			Title:      "Goroutine Leaks",
			Content:    "# Goroutine Leaks\n\nContent.",
			SourceDate: &date,
			Author:     "Иван Петров",
		},
	}
	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:     []string{"goroutines", "leak"},
			Annotation:   "Article about goroutine leaks",
			ThemePath:    "go/concurrency",
			Slug:         "goroutine-leak",
			Type:         "article",
			Content:      "# Goroutine Leaks\n\nContent.",
			Title:        "Goroutine Leaks",
			SourceURL:    "https://habr.com/article/123",
			SourceAuthor: "Иван Петров",
			SourceDate:   &date,
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, f, &mockCommitter{}, base)

	node, err := pipeline.IngestURL(ctx, "https://habr.com/article/123")

	require.NoError(t, err)
	assert.Equal(t, "go/concurrency/goroutine-leak", node.Path)
	assert.Equal(t, "article", node.Metadata["type"])
	assert.Equal(t, "https://habr.com/article/123", node.Metadata["source_url"])
	assert.Equal(t, "Иван Петров", node.Metadata["source_author"])
}
