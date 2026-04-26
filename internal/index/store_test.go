package index

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) *IndexStore {
	t.Helper()

	store, err := NewIndexStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	return store
}

func TestIndexStore_Migrate_ExpectSchemaCreated(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	var count int
	err := store.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM embeddings").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestIndexStore_InsertEmbedding_ExpectRoundTrip(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	vector := []float32{0.1, 0.2, 0.3}
	id, err := store.InsertEmbedding(ctx, vector, "text-embedding-3-small")
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)

	records, err := store.GetAllEmbeddings(ctx)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, vector, records[0].Vector)
	assert.Equal(t, "text-embedding-3-small", records[0].Model)
	assert.Equal(t, 3, records[0].Dimensions)
}

func TestIndexStore_DeleteEmbedding_ExpectRemoved(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	id, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)

	err = store.DeleteEmbedding(ctx, id)
	require.NoError(t, err)

	records, err := store.GetAllEmbeddings(ctx)
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestIndexStore_UpsertNode_ExpectCreated(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)

	err = store.UpsertNode(ctx, "topic/node", "hash1", "hash2", embID)
	require.NoError(t, err)

	node, err := store.GetNodeByPath(ctx, "topic/node")
	require.NoError(t, err)
	assert.Equal(t, "topic/node", node.Path)
	assert.Equal(t, "hash1", node.ContentHash)
	assert.Equal(t, "hash2", node.BodyHash)
	assert.Equal(t, embID, node.NodeEmbeddingID)
}

func TestIndexStore_UpsertNode_WhenExists_ExpectUpdated(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)

	err = store.UpsertNode(ctx, "topic/node", "hash1", "bhash1", embID)
	require.NoError(t, err)

	embID2, err := store.InsertEmbedding(ctx, []float32{0.2}, "model")
	require.NoError(t, err)

	err = store.UpsertNode(ctx, "topic/node", "hash2", "bhash2", embID2)
	require.NoError(t, err)

	node, err := store.GetNodeByPath(ctx, "topic/node")
	require.NoError(t, err)
	assert.Equal(t, "hash2", node.ContentHash)
	assert.Equal(t, "bhash2", node.BodyHash)
	assert.Equal(t, embID2, node.NodeEmbeddingID)
}

func TestIndexStore_DeleteNode_ExpectRemoved(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)

	err = store.UpsertNode(ctx, "topic/node", "hash1", "bhash1", embID)
	require.NoError(t, err)

	err = store.DeleteNode(ctx, "topic/node")
	require.NoError(t, err)

	_, err = store.GetNodeByPath(ctx, "topic/node")
	assert.Error(t, err)
}

func TestIndexStore_ListAllIndexed_ExpectAllNodes(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, store.UpsertNode(ctx, "a/b", "h1", "bh1", embID))
	require.NoError(t, store.UpsertNode(ctx, "c/d", "h2", "bh2", embID))

	nodes, err := store.ListAllIndexed(ctx)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestIndexStore_Chunks_ExpectCRUD(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, store.UpsertNode(ctx, "topic/node", "h1", "bh1", embID))

	chunkEmbID, _ := store.InsertEmbedding(ctx, []float32{0.5}, "model")
	chunks := []Chunk{
		{NodePath: "topic/node", ChunkIndex: 0, Heading: "Intro", Content: "intro text", EmbeddingID: chunkEmbID},
		{NodePath: "topic/node", ChunkIndex: 1, Heading: "Details", Content: "details text", EmbeddingID: chunkEmbID},
	}

	err := store.UpsertChunks(ctx, "topic/node", chunks)
	require.NoError(t, err)

	result, err := store.ListChunksByNode(ctx, "topic/node")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Intro", result[0].Heading)
	assert.Equal(t, "Details", result[1].Heading)
}

func TestIndexStore_UpsertChunks_WhenReplaced_ExpectOldRemoved(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, store.UpsertNode(ctx, "topic/node", "h1", "bh1", embID))

	chunkEmbID, _ := store.InsertEmbedding(ctx, []float32{0.5}, "model")
	require.NoError(t, store.UpsertChunks(ctx, "topic/node", []Chunk{
		{NodePath: "topic/node", ChunkIndex: 0, Heading: "Old", Content: "old", EmbeddingID: chunkEmbID},
	}))

	require.NoError(t, store.UpsertChunks(ctx, "topic/node", []Chunk{
		{NodePath: "topic/node", ChunkIndex: 0, Heading: "New", Content: "new", EmbeddingID: chunkEmbID},
	}))

	result, err := store.ListChunksByNode(ctx, "topic/node")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "New", result[0].Heading)
}

func TestIndexStore_GetStatus_ExpectMetrics(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	status, err := store.GetStatus(ctx, "text-embedding-3-small")
	require.NoError(t, err)
	assert.Equal(t, 0, status.TotalNodes)
	assert.Equal(t, 0, status.TotalChunks)
	assert.Equal(t, "text-embedding-3-small", status.EmbeddingModel)
	assert.Equal(t, "ready", status.Status)
}

func TestIndexStore_ClearAll_ExpectEmpty(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, store.UpsertNode(ctx, "a/b", "h1", "bh1", embID))

	err := store.ClearAll(ctx)
	require.NoError(t, err)

	nodes, err := store.ListAllIndexed(ctx)
	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestEncodeDecodeVector_ExpectRoundTrip(t *testing.T) {
	t.Parallel()

	original := []float32{0.1, -0.2, 0.3, -0.4, 0.5}
	encoded := encodeVector(original)
	decoded := decodeVector(encoded)
	assert.Equal(t, original, decoded)
}
