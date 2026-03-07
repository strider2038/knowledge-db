package kb_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestCreateNode_WhenNewTheme_ExpectNodeCreated(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()

	node, err := store.CreateNode(ctx, base, kb.CreateNodeParams{
		ThemePath: "go/concurrency",
		Slug:      "goroutine-leak",
		Frontmatter: map[string]any{
			"keywords":   []string{"goroutines", "leak"},
			"created":    time.Now().UTC().Format(time.RFC3339),
			"updated":    time.Now().UTC().Format(time.RFC3339),
			"annotation": "Article about goroutine leaks",
			"type":       "article",
		},
		Content: "## Goroutine Leaks\n\nContent here.",
	})

	require.NoError(t, err)
	assert.Equal(t, "go/concurrency/goroutine-leak", node.Path)
	assert.Equal(t, "Article about goroutine leaks", node.Annotation)
	assert.Equal(t, "## Goroutine Leaks\n\nContent here.", node.Content)
}

func TestCreateNode_WhenExistingTheme_ExpectNodeCreated(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"go/concurrency/channels.md": "---\nkeywords: [channels]\ncreated: \"2024-01-01T00:00:00Z\"\nupdated: \"2024-01-01T00:00:00Z\"\n---\n",
	})
	ctx := context.Background()

	node, err := store.CreateNode(ctx, base, kb.CreateNodeParams{
		ThemePath: "go/concurrency",
		Slug:      "select-pattern",
		Frontmatter: map[string]any{
			"keywords": []string{"select", "channels"},
			"created":  time.Now().UTC().Format(time.RFC3339),
			"updated":  time.Now().UTC().Format(time.RFC3339),
		},
		Content: "Select pattern content.",
	})

	require.NoError(t, err)
	assert.Equal(t, "go/concurrency/select-pattern", node.Path)
}

func TestCreateNode_WhenSlugCollision_ExpectSuffixAdded(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(map[string]string{
		"notes/my-note.md": "---\nkeywords: []\ncreated: \"2024-01-01T00:00:00Z\"\nupdated: \"2024-01-01T00:00:00Z\"\n---\n",
	})
	ctx := context.Background()

	node, err := store.CreateNode(ctx, base, kb.CreateNodeParams{
		ThemePath: "notes",
		Slug:      "my-note",
		Frontmatter: map[string]any{
			"keywords": []string{"note"},
			"created":  time.Now().UTC().Format(time.RFC3339),
			"updated":  time.Now().UTC().Format(time.RFC3339),
		},
		Content: "Second note.",
	})

	require.NoError(t, err)
	assert.Equal(t, "notes/my-note-2", node.Path)
}

func TestCreateNode_WhenOptionalFields_ExpectStoredInFrontmatter(t *testing.T) {
	t.Parallel()
	store, base := seedMemFS(nil)
	ctx := context.Background()
	sourceURL := "https://habr.com/article/123"

	node, err := store.CreateNode(ctx, base, kb.CreateNodeParams{
		ThemePath: "links",
		Slug:      "habr-article",
		Frontmatter: map[string]any{
			"keywords":   []string{"go"},
			"created":    time.Now().UTC().Format(time.RFC3339),
			"updated":    time.Now().UTC().Format(time.RFC3339),
			"type":       "article",
			"source_url": sourceURL,
		},
		Content: "",
	})

	require.NoError(t, err)
	assert.Equal(t, "article", node.Metadata["type"])
	assert.Equal(t, sourceURL, node.Metadata["source_url"])
}
