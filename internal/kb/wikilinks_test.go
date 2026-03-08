package kb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/kb"
)

func TestParseWikilinks_WhenEmpty_ExpectEmpty(t *testing.T) {
	t.Parallel()

	result := kb.ParseWikilinks("")
	assert.Empty(t, result)
}

func TestParseWikilinks_WhenSimpleLink_ExpectTarget(t *testing.T) {
	t.Parallel()

	result := kb.ParseWikilinks("See [[target]] for more.")
	assert.Equal(t, []string{"target"}, result)
}

func TestParseWikilinks_WhenLabeledLink_ExpectTarget(t *testing.T) {
	t.Parallel()

	result := kb.ParseWikilinks("See [[target|Custom Label]] here.")
	assert.Equal(t, []string{"target"}, result)
}

func TestParseWikilinks_WhenMultiple_ExpectAllTargets(t *testing.T) {
	t.Parallel()

	result := kb.ParseWikilinks("[[a]] and [[b|c]] and [[a]] again.")
	assert.Equal(t, []string{"a", "b"}, result)
}
