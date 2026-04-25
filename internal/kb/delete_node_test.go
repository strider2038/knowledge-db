package kb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestDeleteNode_WhenSuccessful_ExpectFileRemoved(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

Hello`,
	})
	ctx := context.Background()

	err := store.DeleteNode(ctx, base, "topic/node")

	require.NoError(t, err)
	_, getErr := store.GetNode(ctx, base, "topic/node")
	assert.ErrorIs(t, getErr, kb.ErrNodeNotFound)
}

func TestDeleteNode_WhenHasAttachments_ExpectDirRemoved(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md":       "---\nkeywords: [a]\n---\n",
		"topic/node/image.png": "fake image data",
	})
	ctx := context.Background()

	err := store.DeleteNode(ctx, base, "topic/node")

	require.NoError(t, err)
	_, getErr := store.GetNode(ctx, base, "topic/node")
	assert.ErrorIs(t, getErr, kb.ErrNodeNotFound)
}

func TestDeleteNode_WhenNotFound_ExpectErrNodeNotFound(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()

	err := store.DeleteNode(ctx, base, "nonexistent/path")

	assert.ErrorIs(t, err, kb.ErrNodeNotFound)
}

func TestDeleteNode_WhenEmptyPath_ExpectErrNodeNotFound(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()

	err := store.DeleteNode(ctx, base, "")

	assert.ErrorIs(t, err, kb.ErrNodeNotFound)
}
