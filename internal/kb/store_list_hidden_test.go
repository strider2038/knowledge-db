package kb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/kb"
)

const minimalNodeMD = `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

body
`

func TestListAllNodes_ExcludesHiddenPathSegments(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"vis/ok.md":             minimalNodeMD,
		".cursor/secret.md":     minimalNodeMD,
		"pub/.hidden/nested.md": minimalNodeMD,
	})

	nodes, err := store.ListAllNodes(context.Background(), base)
	require.NoError(t, err)
	paths := make([]string, 0, len(nodes))
	for _, n := range nodes {
		paths = append(paths, n.Path)
	}
	assert.ElementsMatch(t, []string{"vis/ok"}, paths)
}

func TestListNodesWithOptions_Recursive_ExcludesHiddenPathSegments(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"vis/ok.md":             minimalNodeMD,
		".cursor/secret.md":     minimalNodeMD,
		"pub/.hidden/nested.md": minimalNodeMD,
	})

	items, total, err := store.ListNodesWithOptions(context.Background(), base, kb.ListNodesOptions{
		Recursive: true,
		Limit:     200,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, "vis/ok", items[0].Path)
}

func TestListNodes_WhenThemePathIsHidden_ExpectNotFound(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		".cursor/inside.md": minimalNodeMD,
	})

	_, err := store.ListNodes(context.Background(), base, ".cursor")
	require.Error(t, err)
	assert.ErrorIs(t, err, kb.ErrNodeNotFound)
}

func TestListNodes_WhenThemePathDescendsIntoHidden_ExpectNotFound(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"pub/.hidden/nested.md": minimalNodeMD,
	})

	_, err := store.ListNodes(context.Background(), base, "pub/.hidden")
	require.Error(t, err)
	assert.ErrorIs(t, err, kb.ErrNodeNotFound)
}
