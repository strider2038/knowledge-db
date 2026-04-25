package kb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestValidateFrontmatter_WhenValid_ExpectEmpty(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"keywords": []any{"a", "b"},
		"created":  "2024-01-01T00:00:00Z",
		"updated":  "2024-01-01T00:00:00Z",
	}

	result := kb.ValidateFrontmatter(matter)

	assert.Empty(t, result)
}

func TestValidateFrontmatter_WhenNil_ExpectError(t *testing.T) {
	t.Parallel()

	result := kb.ValidateFrontmatter(nil)

	assert.Equal(t, "frontmatter required", result)
}

func TestValidateFrontmatter_WhenMissingKeywords_ExpectError(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"created": "2024-01-01T00:00:00Z",
		"updated": "2024-01-01T00:00:00Z",
	}

	result := kb.ValidateFrontmatter(matter)

	assert.Equal(t, "frontmatter: keywords required", result)
}

func TestValidateFrontmatter_WhenMissingCreated_ExpectError(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"keywords": []any{"a"},
		"updated":  "2024-01-01T00:00:00Z",
	}

	result := kb.ValidateFrontmatter(matter)

	assert.Equal(t, "frontmatter: created required", result)
}

func TestValidateFrontmatter_WhenMissingUpdated_ExpectError(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"keywords": []any{"a"},
		"created":  "2024-01-01T00:00:00Z",
	}

	result := kb.ValidateFrontmatter(matter)

	assert.Equal(t, "frontmatter: updated required", result)
}

func TestValidateFrontmatter_WhenManualProcessedNotBool_ExpectError(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"keywords":          []any{"a"},
		"created":           "2024-01-01T00:00:00Z",
		"updated":           "2024-01-01T00:00:00Z",
		"manual_processed":  "yes",
	}

	result := kb.ValidateFrontmatter(matter)

	assert.Equal(t, "frontmatter: manual_processed must be a boolean", result)
}
