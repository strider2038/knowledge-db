package index

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	vectors [][]float32
}

func (m *mockProvider) Embed(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		if i < len(m.vectors) {
			result[i] = m.vectors[i]
		} else {
			result[i] = []float32{0.1, 0.2, 0.3}
		}
	}

	return result, nil
}

func TestCosineSimilarity_WhenIdentical_Expect1(t *testing.T) {
	t.Parallel()

	v := []float32{1, 0, 0}
	assert.InDelta(t, float32(1.0), cosineSimilarity(v, v), 0.001)
}

func TestCosineSimilarity_WhenOrthogonal_Expect0(t *testing.T) {
	t.Parallel()

	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	assert.InDelta(t, float32(0.0), cosineSimilarity(a, b), 0.001)
}

func TestCosineSimilarity_WhenOpposite_ExpectMinus1(t *testing.T) {
	t.Parallel()

	a := []float32{1, 0, 0}
	b := []float32{-1, 0, 0}
	assert.InDelta(t, float32(-1.0), cosineSimilarity(a, b), 0.001)
}

func TestCosineSimilarity_WhenDifferentLengths_Expect0(t *testing.T) {
	t.Parallel()

	a := []float32{1, 0}
	b := []float32{1, 0, 0}
	assert.Equal(t, float32(0), cosineSimilarity(a, b))
}

func TestCosineSimilarity_WhenZeroVector_Expect0(t *testing.T) {
	t.Parallel()

	a := []float32{0, 0, 0}
	b := []float32{1, 0, 0}
	assert.Equal(t, float32(0), cosineSimilarity(a, b))
}

func TestVectorSearch_WhenMatch_ExpectSorted(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	vec1 := []float32{1, 0, 0}
	vec2 := []float32{0.8, 0.6, 0}

	embID1, _ := store.InsertEmbedding(ctx, vec1, "model")
	embID2, _ := store.InsertEmbedding(ctx, vec2, "model")
	require.NoError(t, store.UpsertNode(ctx, "a/b", "h1", "bh1", embID1))
	require.NoError(t, store.UpsertNode(ctx, "c/d", "h2", "bh2", embID2))

	provider := &mockProvider{vectors: [][]float32{{1, 0, 0}}}

	results, err := VectorSearch(ctx, store, provider, "query", 5)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "a/b", results[0].Path)
	assert.Equal(t, "a/b", results[0].Title)
	assert.Equal(t, "", results[0].Annotation)
	assert.True(t, results[0].Score > results[1].Score)
}

func TestVectorSearch_WhenEmpty_ExpectNil(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	provider := &mockProvider{vectors: [][]float32{{0.1, 0.2}}}

	results, err := VectorSearch(context.Background(), store, provider, "query", 5)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestVectorSearch_WhenTopK_ExpectLimit(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{1, 0, 0}, "model")
	require.NoError(t, store.UpsertNode(ctx, "a/b", "h1", "bh1", embID))
	require.NoError(t, store.UpsertNode(ctx, "c/d", "h2", "bh2", embID))
	require.NoError(t, store.UpsertNode(ctx, "e/f", "h3", "bh3", embID))

	provider := &mockProvider{vectors: [][]float32{{1, 0, 0}}}

	results, err := VectorSearch(ctx, store, provider, "query", 2)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestChunkSearch_WhenMatch_ExpectSorted(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	vec1 := []float32{1, 0, 0}
	vec2 := []float32{0, 1, 0}

	nodeEmbID, _ := store.InsertEmbedding(ctx, vec1, "model")
	require.NoError(t, store.UpsertNode(ctx, "a/b", "h1", "bh1", nodeEmbID))

	chunkEmbID1, _ := store.InsertEmbedding(ctx, vec1, "model")
	chunkEmbID2, _ := store.InsertEmbedding(ctx, vec2, "model")
	require.NoError(t, store.UpsertChunks(ctx, "a/b", []Chunk{
		{NodePath: "a/b", ChunkIndex: 0, Heading: "Section 1", Content: "content 1", EmbeddingID: chunkEmbID1},
		{NodePath: "a/b", ChunkIndex: 1, Heading: "Section 2", Content: "content 2", EmbeddingID: chunkEmbID2},
	}))

	provider := &mockProvider{vectors: [][]float32{{1, 0, 0}}}

	results, err := ChunkSearch(ctx, store, provider, "query", 5)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "Section 1", results[0].Heading)
	assert.True(t, results[0].Score > results[1].Score)
}

func TestChunkSearch_WhenEmpty_ExpectNil(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	provider := &mockProvider{vectors: [][]float32{{0.1}}}

	results, err := ChunkSearch(context.Background(), store, provider, "query", 5)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestKeywordSearchNodes_WhenKeywordMatch_ExpectHit(t *testing.T) {
	t.Parallel()

	store := setupKeywordStore(t)
	ctx := context.Background()

	results, err := KeywordSearchNodes(ctx, store, "sqlite", 5)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "articles/sqlite", results[0].Path)
	assert.Contains(t, results[0].MatchFields, "keywords")
}

func TestKeywordSearchNodes_WhenExactTitleMatch_ExpectBoostedFirst(t *testing.T) {
	t.Parallel()

	store := setupKeywordStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.2}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, "articles/other", "h3", "bh3", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, NodeSearchDocument{
		Path:       "articles/other",
		Title:      "Other",
		Annotation: "SQLite Search appears in this annotation many times SQLite Search",
		Keywords:   []string{"search"},
	}))

	results, err := KeywordSearchNodes(ctx, store, "SQLite Search", 5)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "articles/sqlite", results[0].Path)
	assert.Greater(t, results[0].ExactBoost, 0.0)
}

func TestKeywordSearchNodes_WhenPathMatch_ExpectHit(t *testing.T) {
	t.Parallel()

	store := setupKeywordStore(t)
	ctx := context.Background()

	results, err := KeywordSearchNodes(ctx, store, "articles/sqlite", 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "articles/sqlite", results[0].Path)
	assert.Contains(t, results[0].MatchFields, "path")
}

func TestKeywordSearchChunks_WhenContentMatch_ExpectSnippet(t *testing.T) {
	t.Parallel()

	store := setupKeywordStore(t)
	ctx := context.Background()

	results, err := KeywordSearchChunks(ctx, store, "hybrid retrieval", 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "articles/sqlite", results[0].NodePath)
	assert.Equal(t, "Retrieval", results[0].Heading)
	assert.Contains(t, results[0].Snippet, "hybrid retrieval")
}

func TestKeywordSearch_WhenScanFallback_ExpectHits(t *testing.T) {
	t.Parallel()

	store := setupKeywordStore(t)
	store.keywordIndexMode = "scan"
	ctx := context.Background()

	nodes, chunks, err := KeywordSearch(ctx, store, "local index", 5)
	require.NoError(t, err)
	require.NotEmpty(t, nodes)
	require.NotEmpty(t, chunks)
	assert.Equal(t, "articles/sqlite", nodes[0].Path)
	assert.Equal(t, "articles/sqlite", chunks[0].NodePath)
}

func setupKeywordStore(t *testing.T) *IndexStore {
	t.Helper()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, "articles/sqlite", "h1", "bh1", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, NodeSearchDocument{
		Path:       "articles/sqlite",
		Title:      "SQLite Search",
		Type:       "article",
		Aliases:    []string{"localdb"},
		Annotation: "Local index notes",
		Keywords:   []string{"sqlite", "fts"},
		SourceURL:  "https://example.com/sqlite",
	}))

	chunkEmbID, err := store.InsertEmbedding(ctx, []float32{0.3}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertChunks(ctx, "articles/sqlite", []Chunk{
		{
			NodePath:    "articles/sqlite",
			ChunkIndex:  0,
			Heading:     "Retrieval",
			Content:     "This chunk explains hybrid retrieval with a local index and snippets.",
			EmbeddingID: chunkEmbID,
		},
	}))

	embID, err = store.InsertEmbedding(ctx, []float32{0.4}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, "notes/go", "h2", "bh2", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, NodeSearchDocument{
		Path:       "notes/go",
		Title:      "Go Notes",
		Type:       "note",
		Annotation: "goroutine scheduler",
		Keywords:   []string{"golang"},
		Body:       "notes about channels",
	}))

	return store
}
