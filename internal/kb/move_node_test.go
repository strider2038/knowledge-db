package kb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestMoveNode_WhenDifferentTheme_ExpectMoved(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"old-topic/my-node.md": `---
keywords: [test]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

Content`,
		"new-topic/.gitkeep": "",
	})
	ctx := context.Background()

	node, err := store.MoveNode(ctx, base, "old-topic/my-node", "new-topic/my-node")

	require.NoError(t, err)
	assert.Equal(t, "new-topic/my-node", node.Path)
	assert.Equal(t, "Content", node.Content)

	_, oldErr := store.GetNode(ctx, base, "old-topic/my-node")
	assert.ErrorIs(t, oldErr, kb.ErrNodeNotFound)
}

func TestMoveNode_WhenSlugChanged_ExpectRenamed(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/old-name.md": `---
keywords: [test]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

Content`,
	})
	ctx := context.Background()

	node, err := store.MoveNode(ctx, base, "topic/old-name", "topic/new-name")

	require.NoError(t, err)
	assert.Equal(t, "topic/new-name", node.Path)
}

func TestMoveNode_WhenTargetConflict_ExpectErrConflict(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"source/node.md":   "---\nkeywords: []\n---\n",
		"target/node.md":   "---\nkeywords: []\n---\n",
	})
	ctx := context.Background()

	_, err := store.MoveNode(ctx, base, "source/node", "target/node")

	assert.ErrorIs(t, err, kb.ErrConflict)
}

func TestMoveNode_WhenNotFound_ExpectErrNodeNotFound(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()

	_, err := store.MoveNode(ctx, base, "nonexistent/node", "target/node")

	assert.ErrorIs(t, err, kb.ErrNodeNotFound)
}

func TestMoveNode_WhenNewIntermediateDirs_ExpectCreated(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [test]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

Content`,
	})
	ctx := context.Background()

	node, err := store.MoveNode(ctx, base, "topic/node", "new/deep/path/node")

	require.NoError(t, err)
	assert.Equal(t, "new/deep/path/node", node.Path)
}

func TestMoveNode_WhenPathTraversal_ExpectErrInvalidPath(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": "---\nkeywords: []\n---\n",
	})
	ctx := context.Background()

	_, err := store.MoveNode(ctx, base, "topic/node", "../etc/passwd")

	assert.ErrorIs(t, err, kb.ErrInvalidPath)
}

func TestMoveNode_WhenEmptyTargetPath_ExpectErrInvalidPath(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"topic/node.md": "---\nkeywords: []\n---\n",
	})
	ctx := context.Background()

	_, err := store.MoveNode(ctx, base, "topic/node", "")

	assert.ErrorIs(t, err, kb.ErrInvalidPath)
}
