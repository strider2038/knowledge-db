package index

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClipEmbeddingInput_WhenShort_ExpectUnchanged(t *testing.T) {
	t.Parallel()

	got, truncated := clipEmbeddingInput("hello")

	assert.Equal(t, "hello", got)
	assert.False(t, truncated)
}

func TestClipEmbeddingInput_WhenOverByteLimit_ExpectUTF8Prefix(t *testing.T) {
	t.Parallel()

	text := "a" + strings.Repeat("я", 6000)
	got, truncated := clipEmbeddingInput(text)

	require.True(t, truncated)
	assert.LessOrEqual(t, len(got), maxEmbeddingInputBytes)
	assert.True(t, utf8.ValidString(got))
	assert.True(t, strings.HasPrefix(text, got))
	assert.NotEqual(t, text, got)
}
