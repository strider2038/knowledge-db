package kb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestPatchNodeMetadata_WhenUpdatesLabels_ExpectRoundTrip(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"topic/node.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: Node
---

Hello`,
	})
	ctx := context.Background()
	labels := []string{"  favorite ", "Favorite", "review"}

	require.NoError(t, store.PatchNodeMetadata(ctx, base, "topic/node", kb.PatchNodeMetadataParams{
		Labels: &labels,
	}))

	node, err := store.GetNode(ctx, base, "topic/node")
	require.NoError(t, err)
	assert.Equal(t, []string{"favorite", "review"}, kb.LabelsEffective(node.Metadata))

	require.NoError(t, store.PatchNodeMetadata(ctx, base, "topic/node", kb.PatchNodeMetadataParams{
		Labels: &[]string{},
	}))
	node, err = store.GetNode(ctx, base, "topic/node")
	require.NoError(t, err)
	assert.Empty(t, kb.LabelsEffective(node.Metadata))
}

func TestListNodesWithOptions_WhenLabelsFilterAND_ExpectFiltered(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"topic/a.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: A
labels: [favorite, review]
---

A`,
		"topic/b.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: B
labels: [favorite]
---

B`,
	})
	ctx := context.Background()
	items, total, err := store.ListNodesWithOptions(ctx, base, kb.ListNodesOptions{
		Recursive: true,
		Labels:    []string{"favorite", "review"},
		Limit:     50,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, "topic/a", items[0].Path)
	assert.Equal(t, []string{"favorite", "review"}, items[0].Labels)
}

func TestListLabelSuggestions_WhenLabelsPresent_ExpectUniqueSorted(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"topic/a.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: A
labels: [zebra, Alpha]
---

A`,
		"topic/b.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
type: note
title: B
labels: [alpha]
---

B`,
	})
	ctx := context.Background()
	labels, err := store.ListLabelSuggestions(ctx, base, 500)
	require.NoError(t, err)
	assert.Equal(t, []string{"Alpha", "zebra"}, labels)
}
