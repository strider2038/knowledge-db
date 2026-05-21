package index

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/kb"
)

type noopStore struct{}

func (noopStore) Close() error { return nil }
func (noopStore) DataPath() string { return "" }
func (noopStore) InsertEmbedding(context.Context, []float32, string) (int64, error) {
	return 0, nil
}
func (noopStore) DeleteEmbedding(context.Context, int64) error { return nil }
func (noopStore) GetAllEmbeddings(context.Context) ([]EmbeddingRecord, error) {
	return nil, nil
}
func (noopStore) UpsertNode(context.Context, string, string, string, string, int64) error {
	return nil
}
func (noopStore) UpsertNodeSourceURL(context.Context, string, string) error { return nil }
func (noopStore) GetNodeByPath(context.Context, string) (*IndexedNode, error) {
	return nil, sql.ErrNoRows
}
func (noopStore) GetNodeByID(context.Context, string) (*IndexedNode, error) {
	return nil, sql.ErrNoRows
}
func (noopStore) UpdateNodePath(context.Context, string, string) error { return nil }
func (noopStore) FindBySourceURL(context.Context, string) (*NodeSourceMatch, error) {
	return nil, sql.ErrNoRows
}
func (noopStore) DeleteNode(context.Context, string) error   { return nil }
func (noopStore) DeleteNodeByID(context.Context, string) error { return nil }
func (noopStore) ListAllIndexed(context.Context) ([]IndexedNode, error) { return nil, nil }
func (noopStore) UpsertNodeSearch(context.Context, NodeSearchDocument) error {
	return nil
}
func (noopStore) DeleteNodeSearch(context.Context, string) error { return nil }
func (noopStore) SearchNodeByKeywords(context.Context, []string, int) ([]KeywordNodeHit, error) {
	return nil, nil
}
func (noopStore) UpsertChunks(context.Context, string, string, []Chunk) error { return nil }
func (noopStore) UpsertChunkSearch(context.Context, ChunkSearchDocument) error { return nil }
func (noopStore) ListChunksByNode(context.Context, string) ([]Chunk, error) { return nil, nil }
func (noopStore) DeleteChunks(context.Context, string, string) error { return nil }
func (noopStore) GetAllChunkEmbeddings(context.Context) ([]ChunkEmbedding, error) {
	return nil, nil
}
func (noopStore) GetAllNodeEmbeddings(context.Context) ([]NodeEmbedding, error) {
	return nil, nil
}
func (noopStore) SearchChunkByKeywords(context.Context, []string, int) ([]KeywordChunkHit, error) {
	return nil, nil
}
func (noopStore) GetStatus(context.Context, string) (*IndexStatus, error) {
	return nil, nil //nolint:nilnil
}
func (noopStore) ClearAll(context.Context) error                          { return nil }
func (noopStore) SearchVocabulary(context.Context, SearchVocabularyOptions) ([]string, error) {
	return nil, nil
}
func (noopStore) KeywordIndexMode() string { return "fts5" }
func (noopStore) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil //nolint:nilnil
}
func (noopStore) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func TestComputeContentHash_WhenSameInput_ExpectSameHash(t *testing.T) {
	t.Parallel()

	node := testNode("title", "annotation", []string{"k1"}, "article", "body")
	h1 := computeContentHash(node)
	h2 := computeContentHash(node)
	assert.Equal(t, h1, h2)
}

func TestComputeContentHash_WhenDifferentTitle_ExpectDifferentHash(t *testing.T) {
	t.Parallel()

	n1 := testNode("title1", "ann", nil, "article", "body")
	n2 := testNode("title2", "ann", nil, "article", "body")
	assert.NotEqual(t, computeContentHash(n1), computeContentHash(n2))
}

func TestComputeContentHash_WhenDifferentType_ExpectDifferentHash(t *testing.T) {
	t.Parallel()

	n1 := testNode("title", "ann", nil, "article", "body")
	n2 := testNode("title", "ann", nil, "link", "body")
	assert.NotEqual(t, computeContentHash(n1), computeContentHash(n2))
}

func TestComputeContentHash_WhenDifferentProfile_ExpectDifferentHash(t *testing.T) {
	t.Parallel()

	n1 := testNode("title", "ann", nil, "link", "body")
	n1.Metadata["source_kind"] = "repository"
	n1.Metadata["content_profile"] = string(kb.ContentProfileRepository)
	n2 := testNode("title", "ann", nil, "link", "body")
	n2.Metadata["source_kind"] = "documentation"
	n2.Metadata["content_profile"] = "documentation_profile"

	assert.NotEqual(t, computeContentHash(n1), computeContentHash(n2))
}

func TestComputeBodyHash_WhenSameBody_ExpectSameHash(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, computeBodyHash("hello"))
}

func TestComputeBodyHash_WhenDifferentBody_ExpectDifferentHash(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, computeBodyHash("hello"), computeBodyHash("world"))
}

func TestBuildNodeEmbeddingText_WhenNote_ExpectBodyIncluded(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", []string{"kw"}, "note", "Body content")
	text := buildNodeEmbeddingText(node, "note")
	assert.Contains(t, text, "Title")
	assert.Contains(t, text, "Annotation")
	assert.Contains(t, text, "kw")
	assert.Contains(t, text, "Body content")
}

func TestBuildNodeEmbeddingText_WhenArticle_ExpectBodyExcluded(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", nil, "article", "Body content")
	text := buildNodeEmbeddingText(node, "article")
	assert.Contains(t, text, "Title")
	assert.Contains(t, text, "Annotation")
	assert.NotContains(t, text, "Body content")
}

func TestBuildNodeEmbeddingText_WhenLink_ExpectBodyExcluded(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", nil, "link", "")
	text := buildNodeEmbeddingText(node, "link")
	assert.Contains(t, text, "Title")
	assert.Contains(t, text, "Annotation")
}

func TestBuildNodeEmbeddingText_WhenProfileLink_ExpectBodyIncluded(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", nil, "link", "Digest-only term")
	node.Metadata["content_profile"] = string(kb.ContentProfileRepository)

	text := buildNodeEmbeddingText(node, "link")

	assert.Contains(t, text, "Digest-only term")
}

func TestComputeContentHash_WhenOnlyLabelsChange_ExpectUnchanged(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", []string{"kw"}, "article", "Body")
	hashBefore := computeContentHash(node)
	node.Metadata["labels"] = []string{"favorite", "review"}
	hashAfter := computeContentHash(node)

	assert.Equal(t, hashBefore, hashAfter)
}

func TestBuildNodeEmbeddingText_WhenLabelsPresent_ExpectExcluded(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", []string{"kw"}, "note", "Body")
	node.Metadata["labels"] = []string{"favorite"}

	text := buildNodeEmbeddingText(node, "note")

	assert.NotContains(t, text, "favorite")
	assert.Contains(t, text, "kw")
}

func TestBuildNodeSearchDocument_WhenNote_ExpectSearchMetadata(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", []string{"kw"}, "note", "Body content")
	node.Metadata["aliases"] = []string{"alias"}
	node.Metadata["source_url"] = "https://example.com"
	node.Metadata["manual_processed"] = true

	doc := buildNodeSearchDocument(node, "note")

	assert.Equal(t, "test/path", doc.Path)
	assert.Equal(t, "Title", doc.Title)
	assert.Equal(t, "note", doc.Type)
	assert.Equal(t, []string{"alias"}, doc.Aliases)
	assert.Equal(t, []string{"kw"}, doc.Keywords)
	assert.Equal(t, "https://example.com", doc.SourceURL)
	assert.True(t, doc.ManualProcessed)
	assert.Equal(t, "Body content", doc.Body)
}

func TestBuildNodeSearchDocument_WhenArticle_ExpectBodyExcluded(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", nil, "article", "Body content")
	doc := buildNodeSearchDocument(node, "article")

	assert.Empty(t, doc.Body)
}

func TestBuildNodeSearchDocument_WhenProfileLink_ExpectBodyAndProfileIncluded(t *testing.T) {
	t.Parallel()

	node := testNode("Title", "Annotation", nil, "link", "Digest body")
	node.Metadata["source_kind"] = "repository"
	node.Metadata["content_profile"] = string(kb.ContentProfileRepository)

	doc := buildNodeSearchDocument(node, "link")

	assert.Equal(t, "repository", doc.SourceKind)
	assert.Equal(t, string(kb.ContentProfileRepository), doc.ContentProfile)
	assert.Equal(t, "Digest body", doc.Body)
}

func TestExtractKeywords_WhenStringSlice_ExpectReturn(t *testing.T) {
	t.Parallel()

	meta := map[string]any{"keywords": []string{"a", "b"}}
	kw := extractKeywords(meta)
	assert.Equal(t, []string{"a", "b"}, kw)
}

func TestExtractKeywords_WhenInterfaceSlice_ExpectReturn(t *testing.T) {
	t.Parallel()

	meta := map[string]any{"keywords": []any{"a", "b"}}
	kw := extractKeywords(meta)
	assert.Equal(t, []string{"a", "b"}, kw)
}

func TestExtractKeywords_WhenMissing_ExpectNil(t *testing.T) {
	t.Parallel()

	kw := extractKeywords(map[string]any{})
	assert.Nil(t, kw)
}

func TestSyncWorker_Send_ExpectNonBlocking(t *testing.T) {
	t.Parallel()

	provider := &mockProviderSync{}
	worker := NewSyncWorker(noopStore{}, provider, "/data", "model", time.Second)

	for range 200 {
		worker.Send(SingleNodeEvent{Path: "test/path"})
	}
}

func TestSyncWorker_Run_WhenCancelled_ExpectStop(t *testing.T) {
	t.Parallel()

	provider := &mockProviderSync{}
	worker := NewSyncWorker(noopStore{}, provider, "/data", "model", time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		_ = worker.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
}

func TestSyncWorker_Run_WhenPeriodicTick_ExpectFullReconcileTriggered(t *testing.T) {
	t.Parallel()

	worker := &SyncWorker{
		periodicInterval: 10 * time.Millisecond,
		events:           make(chan SyncEvent, 1),
	}
	var calls atomic.Int32
	worker.fullReconcileFn = func(context.Context) {
		calls.Add(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Millisecond)
	defer cancel()

	_ = worker.Run(ctx)

	assert.GreaterOrEqual(t, calls.Load(), int32(2))
}

func testNode(title, annotation string, keywords []string, nodeType, content string) *kb.Node {
	return &kb.Node{
		ID:      "018f0000-0000-7000-8000-000000000001",
		Path:    "test/path",
		Content: content,
		Metadata: map[string]any{
			"id":         "018f0000-0000-7000-8000-000000000001",
			"title":      title,
			"annotation": annotation,
			"keywords":   keywords,
			"type":       nodeType,
		},
	}
}

type mockProviderSync struct {
	vectors [][]float32
}

func (m *mockProviderSync) Embed(_ context.Context, texts []string) ([][]float32, error) {
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
