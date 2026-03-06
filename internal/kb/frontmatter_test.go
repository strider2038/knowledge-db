package kb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateFrontmatter_WhenValid_ExpectEmpty(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"keywords": []any{"a", "b"},
		"created":  "2024-01-01T00:00:00Z",
		"updated":  "2024-01-01T00:00:00Z",
	}

	result := ValidateFrontmatter(matter)

	assert.Empty(t, result)
}

func TestValidateFrontmatter_WhenNil_ExpectError(t *testing.T) {
	t.Parallel()

	result := ValidateFrontmatter(nil)

	assert.Equal(t, "frontmatter required", result)
}

func TestValidateFrontmatter_WhenMissingKeywords_ExpectError(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"created": "2024-01-01T00:00:00Z",
		"updated": "2024-01-01T00:00:00Z",
	}

	result := ValidateFrontmatter(matter)

	assert.Equal(t, "frontmatter: keywords required", result)
}

func TestValidateFrontmatter_WhenMissingCreated_ExpectError(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"keywords": []any{"a"},
		"updated":  "2024-01-01T00:00:00Z",
	}

	result := ValidateFrontmatter(matter)

	assert.Equal(t, "frontmatter: created required", result)
}

func TestValidateFrontmatter_WhenMissingUpdated_ExpectError(t *testing.T) {
	t.Parallel()

	matter := map[string]any{
		"keywords": []any{"a"},
		"created":  "2024-01-01T00:00:00Z",
	}

	result := ValidateFrontmatter(matter)

	assert.Equal(t, "frontmatter: updated required", result)
}
