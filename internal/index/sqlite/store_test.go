package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/index"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := NewStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	return store
}

func TestStore_Migrate_ExpectSchemaCreated(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)

	var count int
	err := store.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM embeddings").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = store.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM node_search").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = store.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM chunk_search").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, []string{"fts5", "scan"}, store.KeywordIndexMode())
}

func TestStore_Migrate_WhenExistingIndex_ExpectSearchTablesAdded(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, store.Close())

	store, err = NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	var count int
	err = store.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM node_search").Scan(&count)
	require.NoError(t, err)
	err = store.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM chunk_search").Scan(&count)
	require.NoError(t, err)
}

func TestStore_InsertEmbedding_ExpectRoundTrip(t *testing.T) {
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

func TestStore_DeleteEmbedding_ExpectRemoved(t *testing.T) {
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

func TestStore_UpdateNodePath_WhenMoved_ExpectPathUpdatedAndEmbeddingKept(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1, 0.2}, "model")
	require.NoError(t, err)
	nodeID := TestNodeID("old/path")
	require.NoError(t, store.UpsertNode(ctx, nodeID, "old/path", "hash1", "bh1", embID))

	require.NoError(t, store.UpdateNodePath(ctx, nodeID, "new/path"))

	byID, err := store.GetNodeByID(ctx, nodeID)
	require.NoError(t, err)
	assert.Equal(t, "new/path", byID.Path)
	assert.Equal(t, embID, byID.NodeEmbeddingID)

	_, err = store.GetNodeByPath(ctx, "old/path")
	require.Error(t, err)

	byPath, err := store.GetNodeByPath(ctx, "new/path")
	require.NoError(t, err)
	assert.Equal(t, nodeID, byPath.NodeID)
}

func TestStore_FindBySourceURL_WhenIndexed_ExpectMatch(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)
	nodeID := TestNodeID("articles/example")
	require.NoError(t, store.UpsertNode(ctx, nodeID, "articles/example", "h1", "bh1", embID))
	normalized := "https://example.com/article"
	require.NoError(t, store.UpsertNodeSourceURL(ctx, nodeID, normalized))

	match, err := store.FindBySourceURL(ctx, normalized)
	require.NoError(t, err)
	assert.Equal(t, nodeID, match.NodeID)
	assert.Equal(t, "articles/example", match.Path)

	_, err = store.FindBySourceURL(ctx, "https://other.example/x")
	require.Error(t, err)
}

func TestStore_UpsertNode_ExpectCreated(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)

	err = store.UpsertNode(ctx, TestNodeID("topic/node"), "topic/node", "hash1", "hash2", embID)
	require.NoError(t, err)

	node, err := store.GetNodeByPath(ctx, "topic/node")
	require.NoError(t, err)
	assert.Equal(t, "topic/node", node.Path)
	assert.Equal(t, "hash1", node.ContentHash)
	assert.Equal(t, "hash2", node.BodyHash)
	assert.Equal(t, embID, node.NodeEmbeddingID)
}

func TestStore_UpsertNodeSearch_ExpectSearchableTextStored(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, TestNodeID("topic/node"), "topic/node", "hash1", "hash2", embID))

	err = store.UpsertNodeSearch(ctx, index.NodeSearchDocument{
		NodeID: TestNodeID("topic/node"),
		Path:            "topic/node",
		Title:           "SQLite Search",
		Type:            "note",
		Aliases:         []string{"fts"},
		Annotation:      "local index",
		Keywords:        []string{"keyword"},
		SourceURL:       "https://example.com",
		ManualProcessed: true,
		Body:            "note body",
	})
	require.NoError(t, err)

	var title, searchable string
	var manualProcessed int
	err = store.db.QueryRowContext(ctx, `
		SELECT title, searchable_text, manual_processed FROM node_search WHERE path = ?`, "topic/node",
	).Scan(&title, &searchable, &manualProcessed)
	require.NoError(t, err)
	assert.Equal(t, "SQLite Search", title)
	assert.Contains(t, searchable, "keyword")
	assert.Contains(t, searchable, "note body")
	assert.Equal(t, 1, manualProcessed)
}

func TestStore_SearchVocabulary_ExpectCuratedTermsWithLimits(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	for _, doc := range []index.NodeSearchDocument{
		{
			Path:     "ai/context-mode",
			Title:    "Context Mode: Context Management",
			Aliases:  []string{"context management"},
			Keywords: []string{"context mode", "ai"},
		},
		{
			Path:     "ai/harness",
			Title:    "Harness architecture",
			Aliases:  []string{"agent harness"},
			Keywords: []string{"context mode", "infrastructure"},
		},
		{
			Path:     "ai/skills",
			Title:    "Agent Skills",
			Keywords: []string{"context mode", "skills"},
		},
	} {
		embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
		require.NoError(t, err)
		doc.NodeID = TestNodeID(doc.Path)
		require.NoError(t, store.UpsertNode(ctx, doc.NodeID, doc.Path, "hash", "body", embID))
		require.NoError(t, store.UpsertNodeSearch(ctx, doc))
	}

	terms, err := store.SearchVocabulary(ctx, index.SearchVocabularyOptions{
		Limit:                     10,
		MaxDocumentFrequencyRatio: 0.7,
		MinTermRunes:              3,
		MaxTermRunes:              32,
		MaxWords:                  3,
	})
	require.NoError(t, err)

	assert.LessOrEqual(t, len(terms), 10)
	assert.Contains(t, terms, "Context Management")
	assert.Contains(t, terms, "Context Mode")
	assert.NotContains(t, terms, "context mode")
	assert.NotContains(t, terms, "ai")
}

func TestStore_UpsertNode_WhenExists_ExpectUpdated(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)

	err = store.UpsertNode(ctx, TestNodeID("topic/node"), "topic/node", "hash1", "bhash1", embID)
	require.NoError(t, err)

	embID2, err := store.InsertEmbedding(ctx, []float32{0.2}, "model")
	require.NoError(t, err)

	err = store.UpsertNode(ctx, TestNodeID("topic/node"), "topic/node", "hash2", "bhash2", embID2)
	require.NoError(t, err)

	node, err := store.GetNodeByPath(ctx, "topic/node")
	require.NoError(t, err)
	assert.Equal(t, "hash2", node.ContentHash)
	assert.Equal(t, "bhash2", node.BodyHash)
	assert.Equal(t, embID2, node.NodeEmbeddingID)
}

func TestStore_DeleteNode_ExpectRemoved(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, err := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, err)

	err = store.UpsertNode(ctx, TestNodeID("topic/node"), "topic/node", "hash1", "bhash1", embID)
	require.NoError(t, err)

	err = store.DeleteNode(ctx, "topic/node")
	require.NoError(t, err)

	_, err = store.GetNodeByPath(ctx, "topic/node")
	assert.Error(t, err)
}

func TestStore_ListAllIndexed_ExpectAllNodes(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, store.UpsertNode(ctx, TestNodeID("a/b"), "a/b", "h1", "bh1", embID))
	require.NoError(t, store.UpsertNode(ctx, TestNodeID("c/d"), "c/d", "h2", "bh2", embID))

	nodes, err := store.ListAllIndexed(ctx)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
}

func TestStore_Chunks_ExpectCRUD(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, store.UpsertNode(ctx, TestNodeID("topic/node"), "topic/node", "h1", "bh1", embID))

	chunkEmbID, _ := store.InsertEmbedding(ctx, []float32{0.5}, "model")
	chunks := []index.Chunk{
		{NodePath: "topic/node", ChunkIndex: 0, Heading: "Intro", Content: "intro text", EmbeddingID: chunkEmbID},
		{NodePath: "topic/node", ChunkIndex: 1, Heading: "Details", Content: "details text", EmbeddingID: chunkEmbID},
	}

	err := store.UpsertChunks(ctx, TestNodeID("topic/node"), "topic/node", chunks)
	require.NoError(t, err)

	result, err := store.ListChunksByNode(ctx, "topic/node")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Intro", result[0].Heading)
	assert.Equal(t, "Details", result[1].Heading)

	var count int
	err = store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunk_search WHERE node_path = ?`, "topic/node").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestStore_UpsertChunks_WhenReplaced_ExpectOldRemoved(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, store.UpsertNode(ctx, TestNodeID("topic/node"), "topic/node", "h1", "bh1", embID))

	chunkEmbID, _ := store.InsertEmbedding(ctx, []float32{0.5}, "model")
	require.NoError(t, store.UpsertChunks(ctx, TestNodeID("topic/node"), "topic/node", []index.Chunk{
		{NodePath: "topic/node", ChunkIndex: 0, Heading: "Old", Content: "old", EmbeddingID: chunkEmbID},
	}))

	require.NoError(t, store.UpsertChunks(ctx, TestNodeID("topic/node"), "topic/node", []index.Chunk{
		{NodePath: "topic/node", ChunkIndex: 0, Heading: "New", Content: "new", EmbeddingID: chunkEmbID},
	}))

	result, err := store.ListChunksByNode(ctx, "topic/node")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "New", result[0].Heading)
}

func TestStore_GetStatus_ExpectMetrics(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	status, err := store.GetStatus(ctx, "text-embedding-3-small")
	require.NoError(t, err)
	assert.Equal(t, 0, status.TotalNodes)
	assert.Equal(t, 0, status.TotalChunks)
	assert.Equal(t, "text-embedding-3-small", status.EmbeddingModel)
	assert.Contains(t, []string{"fts5", "scan"}, status.KeywordIndex)
	assert.Equal(t, "ready", status.Status)
}

func TestStore_ClearAll_ExpectEmpty(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	ctx := context.Background()

	embID, _ := store.InsertEmbedding(ctx, []float32{0.1}, "model")
	require.NoError(t, store.UpsertNode(ctx, TestNodeID("a/b"), "a/b", "h1", "bh1", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, index.NodeSearchDocument{
		NodeID: TestNodeID("a/b"),Path: "a/b", Title: "Title"}))
	require.NoError(t, store.UpsertChunks(ctx, TestNodeID("a/b"), "a/b", []index.Chunk{
		{NodePath: "a/b", ChunkIndex: 0, Heading: "Heading", Content: "content", EmbeddingID: embID},
	}))

	err := store.ClearAll(ctx)
	require.NoError(t, err)

	nodes, err := store.ListAllIndexed(ctx)
	require.NoError(t, err)
	assert.Empty(t, nodes)

	var count int
	err = store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM node_search`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	err = store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunk_search`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestEncodeDecodeVector_ExpectRoundTrip(t *testing.T) {
	t.Parallel()

	original := []float32{0.1, -0.2, 0.3, -0.4, 0.5}
	encoded := encodeVector(original)
	decoded := decodeVector(encoded)
	assert.Equal(t, original, decoded)
}
