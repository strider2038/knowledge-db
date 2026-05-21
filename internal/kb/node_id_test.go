package kb_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestCreateNode_WhenNoID_ExpectGeneratedUUIDv7(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()

	node, err := store.CreateNode(ctx, base, kb.CreateNodeParams{
		ThemePath: "topic",
		Slug:      "note",
		Frontmatter: map[string]any{
			"keywords": []string{"test"},
			"created":  time.Now().UTC().Format(time.RFC3339),
			"updated":  time.Now().UTC().Format(time.RFC3339),
			"type":     "note",
			"title":    "Note",
		},
		Content: "body",
	})
	require.NoError(t, err)
	require.NotEmpty(t, node.ID)
	assert.True(t, kb.ValidateNodeID(node.ID))
	assert.Equal(t, node.ID, kb.NodeIDFromMetadata(node.Metadata))
}

func TestGetNodeByID_WhenExists_ExpectNode(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()

	created, err := store.CreateNode(ctx, base, kb.CreateNodeParams{
		ThemePath: "topic",
		Slug:      "target",
		Frontmatter: map[string]any{
			"keywords": []string{"x"},
			"created":  time.Now().UTC().Format(time.RFC3339),
			"updated":  time.Now().UTC().Format(time.RFC3339),
			"type":     "note",
			"title":    "Target",
		},
	})
	require.NoError(t, err)

	got, err := store.GetNodeByID(ctx, base, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, created.Path, got.Path)
}

func TestGetNodeByID_WhenMissing_ExpectErrNodeNotFound(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()

	_, err := store.GetNodeByID(ctx, base, "00000000-0000-7000-8000-000000000001")
	require.Error(t, err)
	assert.ErrorIs(t, err, kb.ErrNodeNotFound)
}

func TestMoveNode_WhenMoved_ExpectStableID(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()

	created, err := store.CreateNode(ctx, base, kb.CreateNodeParams{
		ThemePath: "old",
		Slug:      "node",
		Frontmatter: map[string]any{
			"keywords": []string{"x"},
			"created":  time.Now().UTC().Format(time.RFC3339),
			"updated":  time.Now().UTC().Format(time.RFC3339),
			"type":     "note",
			"title":    "Node",
		},
	})
	require.NoError(t, err)

	moved, err := store.MoveNode(ctx, base, created.Path, "new/node")
	require.NoError(t, err)
	assert.Equal(t, created.ID, moved.ID)
}
