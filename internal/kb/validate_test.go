package kb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_WhenValidNode_ExpectNoViolations(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"topic/node1.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

# Content`,
	})

	violations, err := store.Validate(context.Background(), base)
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestValidate_WhenInvalidFrontmatter_ExpectViolation(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"topic/node1.md": `---
keywords: [a]
---

# Content`,
	})

	violations, err := store.Validate(context.Background(), base)
	require.NoError(t, err)
	require.NotEmpty(t, violations, "expected violations for invalid frontmatter")
}

func TestValidate_WhenThemeDepthExceeded_ExpectViolation(t *testing.T) {
	t.Parallel()

	store, base := seedMemFS(map[string]string{
		"a/b/c/d/node1.md": `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
---

# Content`,
	})

	violations, err := store.Validate(context.Background(), base)
	require.NoError(t, err)
	require.NotEmpty(t, violations, "expected violations for theme depth")

	messages := make([]string, len(violations))
	for i, v := range violations {
		messages[i] = v.Message
	}
	assert.Contains(t, messages, "theme depth exceeds 2-3 levels")
}

func TestValidate_WhenAttachmentDir_ExpectNoViolations(t *testing.T) {
	t.Parallel()

	// Директория long-slug/ рядом с long-slug.md — это вложения, не нарушение.
	store, base := seedMemFS(map[string]string{
		"topic/long-slug.md":             "---\nkeywords: [a]\ncreated: \"2024-01-01T00:00:00Z\"\nupdated: \"2024-01-01T00:00:00Z\"\n---\n",
		"topic/long-slug/attachment.png": "",
	})

	violations, err := store.Validate(context.Background(), base)
	require.NoError(t, err)
	assert.Empty(t, violations)
}
