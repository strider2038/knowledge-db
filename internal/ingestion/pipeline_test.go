package ingestion_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
	"github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/kb"
)

// mockOrchestrator — мок LLMOrchestrator.
type mockOrchestrator struct {
	result *llm.ProcessResult
	err    error
	input  llm.ProcessInput
}

func (m *mockOrchestrator) Process(_ context.Context, input llm.ProcessInput) (*llm.ProcessResult, error) {
	m.input = input

	return m.result, m.err
}

// mockTitleGenerator — мок TitleGenerator.
type mockTitleGenerator struct {
	title string
	err   error
}

func (m *mockTitleGenerator) GenerateTitle(_ context.Context, _ string) (string, error) {
	return m.title, m.err
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
func (m *mockCommitter) Status(_ context.Context) (*git.GitStatus, error) {
	return &git.GitStatus{}, nil
}
func (m *mockCommitter) CommitAll(_ context.Context, _ string) error { return nil }
func (m *mockCommitter) DiffStat(_ context.Context) (string, error)  { return "", nil }

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
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

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
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, testBasePath, false, false, nil, nil, nil)

	_, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "text"})

	require.Error(t, err)
}

func TestPipelineIngester_IngestText_WhenTitleHasMarkdown_ExpectStripped(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"микросервисы", "exactly-once"},
			Annotation: "Заметка о exactly-once",
			ThemePath:  "microservices/messaging",
			Slug:       "gde-mozhet-poteryatsya-exactly-once",
			Type:       "note",
			Content:    "Где может потеряться \"exactly-once\"\n\nПредставим классическую схему...",
			Title:      "**Где может потеряться \"exactly-once\"**", // LLM вернул с markdown
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

	node, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "Заметка про exactly-once."})

	require.NoError(t, err)
	assert.Equal(t, "Где может потеряться \"exactly-once\"", node.Metadata["title"])
	assert.Equal(t, []string{"Где может потеряться \"exactly-once\""}, node.Metadata["aliases"])
}

func TestPipelineIngester_IngestText_WhenTitleEmpty_ExpectTitleFromContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"микросервисы", "exactly-once"},
			Annotation: "Заметка о exactly-once",
			ThemePath:  "ai",
			Slug:       "gde-mozhet-poteryatsya-exactly-once",
			Type:       "note",
			Content:    "Где может потеряться \"exactly-once\"\n\nПредставим классическую схему...",
			Title:      "", // LLM не вернул title
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

	node, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "Заметка про exactly-once."})

	require.NoError(t, err)
	assert.Equal(t, "Где может потеряться \"exactly-once\"", node.Metadata["title"])
	assert.Equal(t, []string{"Где может потеряться \"exactly-once\""}, node.Metadata["aliases"])
}

func TestPipelineIngester_IngestText_WhenTitleAndContentEmpty_ExpectTitleFromSlug(t *testing.T) {
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
			Content:    "", // Пустой контент — нечего извлекать
			Title:      "", // LLM не вернул title
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

	node, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "Notes about Claude Cycles."})

	require.NoError(t, err)
	assert.Equal(t, "Professor Donald Knuth Clause Cycles", node.Metadata["title"])
	assert.Equal(t, []string{"Professor Donald Knuth Clause Cycles"}, node.Metadata["aliases"])
}

func TestPipelineIngester_IngestText_WhenTitleEmptyAndContentEmpty_ExpectTitleFromGenerator(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"knuth", "claude"},
			Annotation: "Профессор Кнут о цикле Клода",
			ThemePath:  "ai",
			Slug:       "professor-donald-knuth-clause-cycles",
			Type:       "note",
			Content:    "", // Пустой контент
			Title:      "", // LLM не вернул title
		},
	}
	gen := &mockTitleGenerator{title: "Профессор Кнут: цикл Клода"}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, gen, nil)

	node, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "Заметка про Claude Cycles."})

	require.NoError(t, err)
	assert.Equal(t, "Профессор Кнут: цикл Клода", node.Metadata["title"])
}

func TestPipelineIngester_IngestURL_WhenFetchSuccess_ExpectNodeWithSourceURL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	// Local server for NormalizeURL HEAD — no redirects
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	articleURL := srv.URL + "/article/123"

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
			SourceURL:    articleURL,
			SourceAuthor: "Иван Петров",
			SourceDate:   &date,
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, f, &mockCommitter{}, base, false, false, nil, nil, nil)

	node, err := pipeline.IngestURL(ctx, articleURL)

	require.NoError(t, err)
	assert.Equal(t, "go/concurrency/goroutine-leak", node.Path)
	assert.Equal(t, "article", node.Metadata["type"])
	assert.Equal(t, articleURL, node.Metadata["source_url"])
	assert.Equal(t, "Иван Петров", node.Metadata["source_author"])
}

type mockTranslator struct {
	translateFunc func(ctx context.Context, content string) (string, error)
}

func (m *mockTranslator) Translate(ctx context.Context, content string) (string, error) {
	if m.translateFunc != nil {
		return m.translateFunc(ctx, content)
	}

	return "translated: " + content, nil
}

func TestPipelineIngester_IngestText_WhenArticleAndEnglish_ExpectTranslationCreated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	englishContent := `This is a long article written entirely in English. It contains multiple paragraphs
with various topics and discussions. The content is substantial enough to trigger
the translation heuristic. We need at least two hundred characters of text.`

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"go", "article"},
			Annotation: "English article",
			ThemePath:  "go",
			Slug:       "english-article",
			Type:       "article",
			Content:    englishContent,
			Title:      "English Article",
		},
	}
	translator := &mockTranslator{
		translateFunc: func(_ context.Context, content string) (string, error) {
			return "Переведённый контент: " + content[:50] + "...", nil
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, true, false, translator, nil, nil)

	node, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "English article content."})

	require.NoError(t, err)
	assert.Equal(t, "go/english-article", node.Path)

	translationPath := filepath.Join(base, "go", "english-article.ru.md")
	_, err = fs.Stat(translationPath)
	require.NoError(t, err, "translation file should exist")

	// Verify original has translations field
	origNode, err := store.GetNode(ctx, base, "go/english-article")
	require.NoError(t, err)
	translations, ok := origNode.Metadata["translations"]
	require.True(t, ok)
	assert.Contains(t, translations, "english-article.ru")
}

func TestPipelineIngester_IngestText_WhenArticleAndRussian_ExpectNoTranslation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	russianContent := `Это длинная статья на русском языке. Она содержит несколько абзацев
с различными темами и обсуждениями. Контент достаточно объёмный.
Кириллицы здесь более чем достаточно для порога.`

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"go"},
			Annotation: "Russian article",
			ThemePath:  "go",
			Slug:       "russian-article",
			Type:       "article",
			Content:    russianContent,
			Title:      "Russian Article",
		},
	}
	translateCalled := false
	translator := &mockTranslator{
		translateFunc: func(_ context.Context, _ string) (string, error) {
			translateCalled = true

			return "", nil
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, true, false, translator, nil, nil)

	_, err := pipeline.IngestText(ctx, ingestion.IngestRequest{Text: "Russian content."})

	require.NoError(t, err)
	assert.False(t, translateCalled, "translator should not be called for Russian content")

	translationPath := filepath.Join(base, "go", "russian-article.ru.md")
	_, err = fs.Stat(translationPath)
	assert.Error(t, err, "translation file should not exist")
}

func TestPipelineIngester_IngestURL_WhenRepositoryProfile_ExpectProfilePersistedAndDigestContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"go", "runnable"},
			Annotation: "Repository profile",
			ThemePath:  "go/packages",
			Slug:       "runnable",
			Type:       "link",
			Content:    "## Назначение\n\nПрофиль репозитория.",
			Title:      "Runnable",
		},
	}
	fetch := &mockFetcher{result: &fetcher.FetchResult{
		Title:   "GitHub - pior/runnable",
		Content: "# Runnable\n\nLibrary README with architecture.",
	}}
	pipeline := ingestion.NewPipelineIngester(store, orc, fetch, &mockCommitter{}, base, false, false, nil, nil, nil)

	node, err := pipeline.IngestURL(ctx, "https://github.com/pior/runnable")

	require.NoError(t, err)
	assert.Equal(t, "link", node.Metadata["type"])
	assert.Equal(t, "repository", node.Metadata["source_kind"])
	assert.Equal(t, "repository_profile", node.Metadata["content_profile"])
	assert.Equal(t, "## Назначение\n\nПрофиль репозитория.", node.Content)
	assert.Equal(t, "repository", orc.input.SourceKind)
	assert.Equal(t, "repository_profile", orc.input.ContentProfile)
	assert.Equal(t, "link", orc.input.RecommendedType)
}

func TestPipelineIngester_IngestText_WhenArticleURLDigest_ExpectConceptualNote(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	_ = fs.MkdirAll(base, 0o755)
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:   []string{"go"},
			Annotation: "Conceptual digest",
			ThemePath:  "go/design",
			Slug:       "designing-go-libraries",
			Content:    "## Главная идея\n\nКонцептуальная выжимка.",
			Title:      "Designing Go Libraries",
		},
	}
	pipeline := ingestion.NewPipelineIngester(store, orc, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

	node, err := pipeline.IngestText(ctx, ingestion.IngestRequest{
		Text:      "https://example.com/blog/designing-go-libraries сохрани концептуальное описание",
		SourceURL: "https://example.com/blog/designing-go-libraries",
	})

	require.NoError(t, err)
	assert.Equal(t, "note", node.Metadata["type"])
	assert.Equal(t, "article", node.Metadata["source_kind"])
	assert.Equal(t, "conceptual_digest", node.Metadata["content_profile"])
	assert.Equal(t, "## Главная идея\n\nКонцептуальная выжимка.", node.Content)
}

func TestPipelineIngester_RefreshDescription_WhenRepository_ExpectStableFieldsPreserved(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	require.NoError(t, fs.MkdirAll(filepath.Join(base, "go/packages"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(base, "go/packages/runnable.md"), []byte(`---
title: Old Runnable
aliases: [Old Runnable]
keywords: [old]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-02T00:00:00Z"
annotation: "Old annotation"
type: link
source_url: "https://github.com/pior/runnable"
manual_processed: true
source_author: "Existing Author"
---

Old body
`), 0o644))
	store := kb.NewStore(fs)

	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:       []string{"go", "runnable"},
			Annotation:     "Updated annotation",
			ThemePath:      "ignored",
			Slug:           "ignored",
			Type:           "link",
			SourceKind:     "repository",
			ContentProfile: "repository_profile",
			Content:        "## Назначение\n\nUpdated digest.",
			Title:          "Runnable",
		},
	}
	fetch := &mockFetcher{result: &fetcher.FetchResult{
		Title:   "GitHub - pior/runnable",
		Content: "# Runnable\n\nREADME.",
	}}
	pipeline := ingestion.NewPipelineIngester(store, orc, fetch, &mockCommitter{}, base, false, false, nil, nil, nil)

	node, err := pipeline.RefreshDescription(ctx, "go/packages/runnable")

	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00Z", node.Metadata["created"])
	assert.Equal(t, true, node.Metadata["manual_processed"])
	assert.Equal(t, "https://github.com/pior/runnable", node.Metadata["source_url"])
	assert.Equal(t, "Existing Author", node.Metadata["source_author"])
	assert.Equal(t, "Updated annotation", node.Annotation)
	assert.Equal(t, "repository", node.Metadata["source_kind"])
	assert.Equal(t, "repository_profile", node.Metadata["content_profile"])
	assert.Equal(t, "## Назначение\n\nUpdated digest.", node.Content)
}

func TestPipelineIngester_RefreshDescription_WhenMissingSourceURL_ExpectError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	require.NoError(t, fs.MkdirAll(filepath.Join(base, "notes"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(base, "notes/local.md"), []byte(`---
keywords: [local]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Local"
---
`), 0o644))
	store := kb.NewStore(fs)
	pipeline := ingestion.NewPipelineIngester(store, &mockOrchestrator{}, &mockFetcher{}, &mockCommitter{}, base, false, false, nil, nil, nil)

	_, err := pipeline.RefreshDescription(ctx, "notes/local")

	require.Error(t, err)
	assert.ErrorIs(t, err, ingestion.ErrSourceURLRequired)
}

func TestPipelineIngester_RefreshDescription_WhenFetchFails_ExpectFileNotMutated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	nodePath := filepath.Join(base, "news/release.md")
	require.NoError(t, fs.MkdirAll(filepath.Dir(nodePath), 0o755))
	original := `---
title: Old Release
keywords: [old]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Old"
type: link
source_url: "https://techcrunch.com/release"
---

Old body
`
	require.NoError(t, afero.WriteFile(fs, nodePath, []byte(original), 0o644))
	store := kb.NewStore(fs)
	pipeline := ingestion.NewPipelineIngester(store, &mockOrchestrator{}, &mockFetcher{err: ingestion.ErrNotImplemented}, &mockCommitter{}, base, false, false, nil, nil, nil)

	_, err := pipeline.RefreshDescription(ctx, "news/release")

	require.Error(t, err)
	data, readErr := afero.ReadFile(fs, nodePath)
	require.NoError(t, readErr)
	assert.Equal(t, original, string(data))
}

func TestPipelineIngester_RefreshDescription_WhenNewsLink_ExpectTypeCorrectedToBriefNote(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	base := testBasePath
	require.NoError(t, fs.MkdirAll(filepath.Join(base, "news"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(base, "news/release.md"), []byte(`---
title: Old Release
keywords: [old]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Old"
type: link
source_url: "https://techcrunch.com/release"
---

Old body
`), 0o644))
	store := kb.NewStore(fs)
	orc := &mockOrchestrator{
		result: &llm.ProcessResult{
			Keywords:       []string{"ai", "релиз"},
			Annotation:     "Brief release digest",
			Type:           "note",
			SourceKind:     "news",
			ContentProfile: "brief_digest",
			Content:        "## Суть новости\n\nКраткая выжимка.",
			Title:          "Release",
		},
	}
	fetch := &mockFetcher{result: &fetcher.FetchResult{Title: "Release", Content: "Product released today."}}
	pipeline := ingestion.NewPipelineIngester(store, orc, fetch, &mockCommitter{}, base, false, false, nil, nil, nil)

	node, err := pipeline.RefreshDescription(ctx, "news/release")

	require.NoError(t, err)
	assert.Equal(t, "note", node.Metadata["type"])
	assert.Equal(t, "news", node.Metadata["source_kind"])
	assert.Equal(t, "brief_digest", node.Metadata["content_profile"])
	assert.Equal(t, "## Суть новости\n\nКраткая выжимка.", node.Content)
}
