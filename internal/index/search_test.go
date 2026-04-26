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
