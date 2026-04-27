package index

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/kb"
)

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

func TestComputeBodyHash_WhenSameBody_ExpectSameHash(t *testing.T) {
	t.Parallel()

	assert.Equal(t, computeBodyHash("hello"), computeBodyHash("hello"))
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

func TestExtractKeywords_WhenStringSlice_ExpectReturn(t *testing.T) {
	t.Parallel()

	meta := map[string]any{"keywords": []string{"a", "b"}}
	kw := extractKeywords(meta)
	assert.Equal(t, []string{"a", "b"}, kw)
}

func TestExtractKeywords_WhenInterfaceSlice_ExpectReturn(t *testing.T) {
	t.Parallel()

	meta := map[string]any{"keywords": []interface{}{"a", "b"}}
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

	store := setupTestStore(t)
	provider := &mockProvider{}
	worker := NewSyncWorker(store, provider, "/data", "model", time.Second)

	for i := 0; i < 200; i++ {
		worker.Send(SingleNodeEvent{Path: "test/path"})
	}
}

func TestSyncWorker_Run_WhenCancelled_ExpectStop(t *testing.T) {
	t.Parallel()

	store := setupTestStore(t)
	provider := &mockProvider{}
	worker := NewSyncWorker(store, provider, "/data", "model", time.Second)

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

func testNode(title, annotation string, keywords []string, nodeType, content string) *kb.Node {
	return &kb.Node{
		Path:    "test/path",
		Content: content,
		Metadata: map[string]any{
			"title":      title,
			"annotation": annotation,
			"keywords":   keywords,
			"type":       nodeType,
		},
	}
}
