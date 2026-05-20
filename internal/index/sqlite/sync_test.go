package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/index"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestSyncWorker_ProcessSingleNode_WhenProfileLinkDigest_ExpectKeywordAndChunkRetrieval(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, "go/packages"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, "go/packages/runnable.md"), []byte(`---
id: "018f0000-0000-7000-8000-000000000099"
title: Runnable
keywords: [go]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Repository profile"
type: link
source_url: "https://github.com/pior/runnable"
source_kind: repository
content_profile: `+string(kb.ContentProfileRepository)+`

---

## Назначение

`+strings.Repeat("digestonlyterm ", 160)+`
`), 0o644))

	store := setupTestStore(t)
	worker := index.NewSyncWorker(store, &mockProvider{vectors: [][]float32{{1, 0}, {1, 0}}}, dataPath, "model", 0)

	worker.ProcessSingleNodeForTest(ctx, "go/packages/runnable")

	nodeHits, err := index.KeywordSearchNodes(ctx, store, "digestonlyterm", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, nodeHits)
	chunkHits, err := index.KeywordSearchChunks(ctx, store, "digestonlyterm", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, chunkHits)
	chunkResults, err := index.ChunkSearch(ctx, store, &mockProvider{vectors: [][]float32{{1, 0}}}, "digestonlyterm", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, chunkResults)
}

func TestSyncWorker_ProcessSingleNode_WhenNoID_ExpectSkipped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dataPath, "topic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, "topic", "no-id.md"), []byte(`---
title: No ID
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
---

Body without id.
`), 0o644))

	store := setupTestStore(t)
	worker := index.NewSyncWorker(store, &mockProvider{vectors: [][]float32{{1, 0}}}, dataPath, "model", 0)
	worker.ProcessSingleNodeForTest(ctx, "topic/no-id")

	nodes, err := store.ListAllIndexed(ctx)
	require.NoError(t, err)
	assert.Empty(t, nodes)
}
