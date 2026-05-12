package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestComputeContentHash_WhenDifferentProfile_ExpectDifferentHash(t *testing.T) {
	t.Parallel()

	n1 := testNode("title", "ann", nil, "link", "body")
	n1.Metadata["source_kind"] = "repository"
	n1.Metadata["content_profile"] = "repository_profile"
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
	node.Metadata["content_profile"] = "repository_profile"

	text := buildNodeEmbeddingText(node, "link")

	assert.Contains(t, text, "Digest-only term")
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
	node.Metadata["content_profile"] = "repository_profile"

	doc := buildNodeSearchDocument(node, "link")

	assert.Equal(t, "repository", doc.SourceKind)
	assert.Equal(t, "repository_profile", doc.ContentProfile)
	assert.Equal(t, "Digest body", doc.Body)
}

func TestSyncWorker_ProcessSingleNode_WhenProfileLinkDigest_ExpectKeywordAndChunkRetrieval(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, "go/packages"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, "go/packages/runnable.md"), []byte(`---
title: Runnable
keywords: [go]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Repository profile"
type: link
source_url: "https://github.com/pior/runnable"
source_kind: repository
content_profile: repository_profile
---

## Назначение

`+strings.Repeat("digestonlyterm ", 160)+`
`), 0o644))

	store := setupTestStore(t)
	worker := NewSyncWorker(store, &mockProvider{vectors: [][]float32{{1, 0}, {1, 0}}}, dataPath, "model", 0)

	worker.processSingleNode(ctx, "go/packages/runnable")

	nodeHits, err := KeywordSearchNodes(ctx, store, "digestonlyterm", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, nodeHits)
	chunkHits, err := KeywordSearchChunks(ctx, store, "digestonlyterm", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, chunkHits)
	chunkResults, err := ChunkSearch(ctx, store, &mockProvider{vectors: [][]float32{{1, 0}}}, "digestonlyterm", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, chunkResults)
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

	store := setupTestStore(t)
	provider := &mockProvider{}
	worker := NewSyncWorker(store, provider, "/data", "model", time.Second)

	for range 200 {
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

func TestSyncWorker_Run_WhenPeriodicTick_ExpectFullReconcileTriggered(t *testing.T) {
	t.Parallel()

	worker := &SyncWorker{
		periodicInterval: 10 * time.Millisecond,
		events:           make(chan SyncEvent, 1),
	}
	var calls int32
	worker.fullReconcileFn = func(context.Context) {
		atomic.AddInt32(&calls, 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Millisecond)
	defer cancel()

	_ = worker.Run(ctx)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&calls), int32(2))
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
