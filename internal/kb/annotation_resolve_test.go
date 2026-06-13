package kb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownPlainText_WhenInlineMarkdown_ExpectStripped(t *testing.T) {
	t.Parallel()
	text := markdownPlainText("**bold** and `code`")
	assert.Equal(t, "bold and code", text)
}

func TestResolveTextQuote_WhenExactInPlainText_ExpectTrue(t *testing.T) {
	t.Parallel()
	content := "Intro\n\n**exactly-once** delivery works"
	anchor := &AnnotationAnchor{Exact: "exactly-once"}
	assert.True(t, resolveTextQuote(content, anchor))
}

func TestResolveTextQuote_WhenExactOnlyInMarkdownMarkup_ExpectFalse(t *testing.T) {
	t.Parallel()
	content := "Intro\n\n**missing** delivery works"
	anchor := &AnnotationAnchor{Exact: "**missing**"}
	assert.False(t, resolveTextQuote(content, anchor))
}

func TestResolveTextQuote_WhenPrefixSuffixDisambiguate_ExpectTrue(t *testing.T) {
	t.Parallel()
	content := "alpha beta gamma alpha beta end"
	anchor := &AnnotationAnchor{
		Exact:  "alpha beta",
		Prefix: "gamma ",
		Suffix: " end",
	}
	assert.True(t, resolveTextQuote(content, anchor))
}

func TestResolveTextQuote_WhenHeadingSection_ExpectScoped(t *testing.T) {
	t.Parallel()
	content := "# First\nshared text\n\n## Second\nshared text"
	anchor := &AnnotationAnchor{
		Exact:     "shared text",
		HeadingID: "second",
		Prefix:    "",
		Suffix:    "",
	}
	assert.True(t, resolveTextQuote(content, anchor))
}

func TestRemapAnnotationContentPath_WhenTranslation_ExpectRewritten(t *testing.T) {
	t.Parallel()
	updated, ok := remapAnnotationContentPath("old/node.ru", "old/node", "new/node")
	require.True(t, ok)
	assert.Equal(t, "new/node.ru", updated)
}
