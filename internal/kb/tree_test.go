package kb_test

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestParseNodeFile_WhenValid_ExpectContent(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"node1/node1.md": `---
keywords: [a, b]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Short desc"
---

# Title

Body text`,
	})

	node, err := store.GetNode(context.Background(), base, "node1")
	require.NoError(t, err)

	assert.Equal(t, "Short desc", node.Annotation)
	assert.Equal(t, "# Title\n\nBody text", node.Content)
	assert.NotNil(t, node.Metadata["keywords"])
}

func TestParseNodeFile_WhenFileMissing_ExpectError(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/node1", 0o755)
	store := kb.NewStore(fs)
	base := "/"

	_, err := store.GetNode(context.Background(), base, "node1")
	require.Error(t, err)
}

func TestIsNodeDir_WhenHasMainFile_ExpectTrue(t *testing.T) {
	t.Parallel()

	store, _ := seedMemFS(map[string]string{
		"node1/node1.md": "---\nkeywords: []\ncreated: \"\"\nupdated: \"\"\n---\n",
	})

	assert.True(t, store.IsNodeDir("/node1"))
}

func TestIsNodeDir_WhenNoMainFile_ExpectFalse(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/node1", 0o755)
	store := kb.NewStore(fs)

	assert.False(t, store.IsNodeDir("/node1"))
}

func TestGetNode_WhenValid_ExpectNode(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"topic/node1/node1.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

Content`,
	})

	node, err := store.GetNode(context.Background(), base, "topic/node1")
	require.NoError(t, err)

	assert.Equal(t, "topic/node1", node.Path)
	assert.Equal(t, "Content", node.Content)
}

func TestGetNode_WhenNotFound_ExpectError(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{})

	_, err := store.GetNode(context.Background(), base, "missing/path")
	require.Error(t, err)
	assert.ErrorIs(t, err, kb.ErrNodeNotFound)
}
