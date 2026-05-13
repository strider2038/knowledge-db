package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/index"
)

type testEmbeddingProvider struct {
	vectors [][]float32
}

func (p *testEmbeddingProvider) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return p.vectors, nil
}

func TestBearerToken_WhenValid_ExpectToken(t *testing.T) {
	t.Parallel()

	token, ok := bearerToken("Bearer abc")
	require.True(t, ok)
	require.Equal(t, "abc", token)
}

func TestBearerToken_WhenInvalidFormat_ExpectFalse(t *testing.T) {
	t.Parallel()

	_, ok := bearerToken("Basic abc")
	require.False(t, ok)
}

func TestHandler_WhenMissingBearer_Expect401(t *testing.T) {
	t.Parallel()

	h := NewHandler("test-key", nil, nil)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/mcp", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_WhenValidBearer_ExpectPassesAuth(t *testing.T) {
	t.Parallel()

	h := NewHandler("test-key", nil, nil)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1}`))
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	require.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestSearchServices_SemanticSearch_WhenProviderMissing_ExpectUnavailableError(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	store, err := index.NewIndexStore(filepath.Join(tmp, "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	services := &searchServices{indexStore: store}
	_, err = services.semanticSearch(t.Context(), semanticSearchInput{Query: "test", Limit: 5})
	require.Error(t, err)
	assert.ErrorIs(t, err, errSemanticSearchUnavailable)
}

func TestSearchServices_SearchNotes_WhenQueryEmpty_ExpectValidationError(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	store, err := index.NewIndexStore(filepath.Join(tmp, "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	services := &searchServices{indexStore: store}
	_, err = services.searchNotes(t.Context(), searchNotesInput{Query: "   "})
	require.EqualError(t, err, "query is required")
}

func TestSearchServices_SearchNotes_WhenValidQuery_ExpectRankedResults(t *testing.T) {
	t.Parallel()

	store := setupSearchStore(t)
	services := &searchServices{indexStore: store}

	got, err := services.searchNotes(t.Context(), searchNotesInput{Query: "sqlite", Limit: 5})
	require.NoError(t, err)
	require.NotEmpty(t, got.Results)
	assert.Equal(t, "articles/sqlite", got.Results[0].Path)
	assert.Equal(t, "SQLite", got.Results[0].Title)
	assert.Equal(t, "article", got.Results[0].Type)
}

func TestSearchServices_SemanticSearch_WhenProviderEnabled_ExpectResults(t *testing.T) {
	t.Parallel()

	store := setupSearchStore(t)
	services := &searchServices{
		indexStore: store,
		provider: &testEmbeddingProvider{
			vectors: [][]float32{{1, 0, 0}},
		},
	}

	got, err := services.semanticSearch(t.Context(), semanticSearchInput{Query: "vector-only-query", Limit: 5})
	require.NoError(t, err)
	require.NotEmpty(t, got.Results)
	assert.Equal(t, "articles/sqlite", got.Results[0].Path)
}

func TestSearchServices_GetNote_WhenExists_ExpectContent(t *testing.T) {
	t.Parallel()

	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, "topic"), 0o755))
	content := `---
title: Test Node
type: article
source_url: https://example.com
keywords:
  - one
  - two
---
Body text for reading.`
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, "topic", "node.md"), []byte(content), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, ".kb"), 0o755))

	store, err := index.NewIndexStore(filepath.Join(dataPath, ".kb", "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	services := &searchServices{indexStore: store}
	got, err := services.getNote(context.Background(), getNoteInput{Path: "topic/node"})
	require.NoError(t, err)
	assert.Equal(t, "topic/node", got.Path)
	assert.Equal(t, "Test Node", got.Title)
	assert.Equal(t, "article", got.Type)
	assert.Equal(t, "https://example.com", got.SourceURL)
	assert.Equal(t, []string{"one", "two"}, got.Keywords)
	assert.Contains(t, got.Content, "Body text for reading.")
	assert.False(t, got.Truncated)
}

func TestSearchServices_GetNote_WhenMaxChars_ExpectTruncated(t *testing.T) {
	t.Parallel()

	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, "topic"), 0o755))
	content := `---
title: Test Node
---
0123456789abcdef`
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, "topic", "node.md"), []byte(content), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, ".kb"), 0o755))

	store, err := index.NewIndexStore(filepath.Join(dataPath, ".kb", "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	services := &searchServices{indexStore: store}
	got, err := services.getNote(context.Background(), getNoteInput{Path: "topic/node", MaxChars: 5})
	require.NoError(t, err)
	assert.Equal(t, "01234", got.Content)
	assert.True(t, got.Truncated)
}

func TestSearchServices_GetNote_WhenIncludeContentFalse_ExpectMetadataOnly(t *testing.T) {
	t.Parallel()

	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, "topic"), 0o755))
	content := `---
title: Test Node
---
0123456789abcdef`
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, "topic", "node.md"), []byte(content), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, ".kb"), 0o755))

	store, err := index.NewIndexStore(filepath.Join(dataPath, ".kb", "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	services := &searchServices{indexStore: store}
	include := false
	got, err := services.getNote(context.Background(), getNoteInput{
		Path:           "topic/node",
		IncludeContent: &include,
		MaxChars:       5,
	})
	require.NoError(t, err)
	assert.Empty(t, got.Content)
	assert.False(t, got.Truncated)
	assert.Equal(t, "Test Node", got.Title)
}

func TestSearchServices_GetNote_WhenPathEmpty_ExpectValidationError(t *testing.T) {
	t.Parallel()

	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, ".kb"), 0o755))
	store, err := index.NewIndexStore(filepath.Join(dataPath, ".kb", "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	services := &searchServices{indexStore: store}
	_, err = services.getNote(context.Background(), getNoteInput{})
	require.EqualError(t, err, "path is required")
}

func TestSearchServices_GetNote_WhenNotFound_ExpectError(t *testing.T) {
	t.Parallel()

	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, ".kb"), 0o755))
	store, err := index.NewIndexStore(filepath.Join(dataPath, ".kb", "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	services := &searchServices{indexStore: store}
	_, err = services.getNote(context.Background(), getNoteInput{Path: "topic/missing"})
	require.EqualError(t, err, "node not found: topic/missing")
}

func setupSearchStore(t *testing.T) *index.IndexStore {
	t.Helper()

	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, ".kb"), 0o755))
	store, err := index.NewIndexStore(filepath.Join(dataPath, ".kb", "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	embID, err := store.InsertEmbedding(ctx, []float32{1, 0, 0}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, "articles/sqlite", "h1", "bh1", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, index.NodeSearchDocument{
		Path:     "articles/sqlite",
		Title:    "SQLite",
		Type:     "article",
		Keywords: []string{"sqlite"},
	}))

	return store
}
