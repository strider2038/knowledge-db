package kb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestPatchNodeManualProcessed_WhenSetAndClear_ExpectRoundTrip(t *testing.T) {
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

	require.NoError(t, store.PatchNodeManualProcessed(ctx, base, "topic/node", true))
	node, err := store.GetNode(ctx, base, "topic/node")
	require.NoError(t, err)
	assert.True(t, kb.ManualProcessedEffective(node.Metadata))

	require.NoError(t, store.PatchNodeManualProcessed(ctx, base, "topic/node", false))
	node, err = store.GetNode(ctx, base, "topic/node")
	require.NoError(t, err)
	assert.False(t, kb.ManualProcessedEffective(node.Metadata))
}
