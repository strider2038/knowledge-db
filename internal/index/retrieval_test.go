package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrievalService_Retrieve_WhenKeywordAndVector_ExpectFusedResults(t *testing.T) {
	t.Parallel()

	store := setupRetrievalStore(t)
	provider := &mockProvider{vectors: [][]float32{{1, 0, 0}}}
	service := NewRetrievalService(store, provider)

	results, err := service.Retrieve(context.Background(), RetrievalOptions{Query: "sqlite", Limit: 5})
	require.NoError(t, err)
	require.NotEmpty(t, results)

	assert.Equal(t, "articles/sqlite", results[0].Path)
	assert.Contains(t, results[0].SourceKinds, "keyword")
	assert.Contains(t, results[0].SourceKinds, "vector_node")
	assert.Contains(t, results[0].MatchReasons, "keywords")
}

func TestRetrievalService_Retrieve_WhenExactKeyword_ExpectBoosted(t *testing.T) {
	t.Parallel()

	store := setupRetrievalStore(t)
	provider := &mockProvider{vectors: [][]float32{{0, 0, 1}}}
	service := NewRetrievalService(store, provider)

	results, err := service.Retrieve(context.Background(), RetrievalOptions{Query: "sqlite", Limit: 5})
	require.NoError(t, err)
	require.NotEmpty(t, results)

	assert.Equal(t, "articles/sqlite", results[0].Path)
	assert.Contains(t, results[0].SourceKinds, "exact")
}

func TestRetrievalService_Retrieve_WhenFilterType_ExpectOnlyMatchingType(t *testing.T) {
	t.Parallel()

	store := setupRetrievalStore(t)
	service := NewRetrievalService(store, nil)

	results, err := service.Retrieve(context.Background(), RetrievalOptions{Query: "local", Types: []string{"note"}, Limit: 5})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "notes/local", results[0].Path)
	assert.Equal(t, "note", results[0].Type)
}

func TestRetrievalService_Retrieve_WhenNodeSearchMissingAndFilterType_ExpectMetadataFromFS(t *testing.T) {
	t.Parallel()

	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, ".kb"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, "articles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, "articles", "sqlite.md"), []byte(`---
title: SQLite
type: article
aliases: []
annotation: Local database
keywords:
  - sqlite
created: 2026-01-01
updated: 2026-01-01
---

SQLite body.
`), 0o644))

	store, err := NewIndexStore(filepath.Join(dataPath, ".kb", "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	embID, err := store.InsertEmbedding(context.Background(), []float32{1, 0, 0}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(context.Background(), "articles/sqlite", "h1", "bh1", embID))

	provider := &mockProvider{vectors: [][]float32{{1, 0, 0}}}
	service := NewRetrievalService(store, provider)

	results, err := service.Retrieve(context.Background(), RetrievalOptions{
		Query: "sqlite",
		Types: []string{"article"},
		Limit: 5,
	})

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "articles/sqlite", results[0].Path)
	assert.Equal(t, "SQLite", results[0].Title)
	assert.Equal(t, "article", results[0].Type)
}

func TestRetrievalService_Retrieve_WhenSourcePaths_ExpectRestricted(t *testing.T) {
	t.Parallel()

	store := setupRetrievalStore(t)
	service := NewRetrievalService(store, nil)

	results, err := service.Retrieve(context.Background(), RetrievalOptions{
		Query:       "local",
		SourcePaths: []string{"notes/local"},
		Limit:       5,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "notes/local", results[0].Path)
}

func TestRetrievalService_Retrieve_WhenPathRecursive_ExpectSubtreeOnly(t *testing.T) {
	t.Parallel()

	store := setupRetrievalStore(t)
	service := NewRetrievalService(store, nil)

	results, err := service.Retrieve(context.Background(), RetrievalOptions{
		Query:     "local",
		Path:      "notes",
		Recursive: true,
		Limit:     5,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "notes/local", results[0].Path)
}

func TestRetrievalService_Retrieve_WhenManualProcessed_ExpectFiltered(t *testing.T) {
	t.Parallel()

	store := setupRetrievalStore(t)
	service := NewRetrievalService(store, nil)
	manualProcessed := true

	results, err := service.Retrieve(context.Background(), RetrievalOptions{
		Query:           "local",
		ManualProcessed: &manualProcessed,
		Limit:           5,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "notes/local", results[0].Path)
}

func TestRetrievalService_Retrieve_WhenChatWeakVectorOnly_ExpectCutoff(t *testing.T) {
	t.Parallel()

	store := setupRetrievalStore(t)
	provider := &mockProvider{vectors: [][]float32{{0, 0, 1}}}
	service := NewRetrievalService(store, provider)

	results, err := service.Retrieve(context.Background(), RetrievalOptions{
		Query: "unrelated",
		Mode:  RetrievalModeChat,
		Limit: 5,
	})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func setupRetrievalStore(t *testing.T) *IndexStore {
	t.Helper()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{1, 0, 0}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, "articles/sqlite", "h1", "bh1", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, NodeSearchDocument{
		Path:       "articles/sqlite",
		Title:      "SQLite",
		Type:       "article",
		Annotation: "database retrieval",
		Keywords:   []string{"sqlite"},
	}))
	chunkEmbID, err := store.InsertEmbedding(ctx, []float32{1, 0, 0}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertChunks(ctx, "articles/sqlite", []Chunk{
		{NodePath: "articles/sqlite", ChunkIndex: 0, Heading: "Search", Content: "sqlite local retrieval chunk", EmbeddingID: chunkEmbID},
	}))

	embID, err = store.InsertEmbedding(ctx, []float32{0, 1, 0}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, "notes/local", "h2", "bh2", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, NodeSearchDocument{
		Path:            "notes/local",
		Title:           "Local Note",
		Type:            "note",
		Annotation:      "local workflow",
		Keywords:        []string{"local"},
		ManualProcessed: true,
		Body:            "local workflow body",
	}))

	return store
}
