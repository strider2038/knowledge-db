package translation //nolint:testpackage // testing unexported chunking functions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractCodeBlocks_WhenNoBlocks_ExpectSameText(t *testing.T) {
	t.Parallel()

	text := "Plain text without code."
	result, blocks := extractCodeBlocks(text)

	assert.Equal(t, text, result)
	assert.Empty(t, blocks)
}

func TestExtractCodeBlocks_WhenOneBlock_ExpectPlaceholder(t *testing.T) {
	t.Parallel()

	text := "Before\n```go\nfmt.Println(1)\n```\nAfter"
	result, blocks := extractCodeBlocks(text)

	assert.Contains(t, result, "___KB_CODE_0___")
	assert.NotContains(t, result, "fmt.Println")
	assert.Len(t, blocks, 1)
	assert.Equal(t, "```go\nfmt.Println(1)\n```", blocks[0].content)
}

func TestReinsertCodeBlocks_WhenPlaceholders_ExpectOriginal(t *testing.T) {
	t.Parallel()

	text := "Before ___KB_CODE_0___ After"
	blocks := []codeBlockPlaceholder{
		{placeholder: "___KB_CODE_0___", content: "```go\ncode\n```"},
	}

	result := reinsertCodeBlocks(text, blocks)

	assert.Equal(t, "Before ```go\ncode\n``` After", result)
}

func TestSplitIntoChunks_WhenShort_ExpectSingleChunk(t *testing.T) {
	t.Parallel()

	text := "Short text"
	chunks := splitIntoChunks(text)

	require.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0])
}

func TestSplitIntoChunks_WhenLong_ExpectMultipleChunks(t *testing.T) {
	t.Parallel()

	paragraphs := make([]string, 20)
	for i := range paragraphs {
		paragraphs[i] = strings.Repeat("word ", 100) // ~500 chars per paragraph
	}
	text := strings.Join(paragraphs, "\n\n")

	chunks := splitIntoChunks(text)

	require.Greater(t, len(chunks), 1)
	for _, c := range chunks {
		assert.LessOrEqual(t, len(c), chunkThreshold+500) // allow some margin
	}
}

func TestMergeChunks_WhenNoOverlap_ExpectConcatenation(t *testing.T) {
	t.Parallel()

	chunks := []string{"First part.", "Second part."}
	result := mergeChunks(chunks)

	assert.Equal(t, "First part.\n\nSecond part.", result)
}

func TestMergeChunks_WhenOverlap_ExpectTrimmed(t *testing.T) {
	t.Parallel()

	chunks := []string{
		"The end of the first chunk is here.",
		"is here. The start of the second chunk.",
	}
	result := mergeChunks(chunks)

	assert.Contains(t, result, "The end of the first chunk is here.")
	assert.Contains(t, result, "The start of the second chunk.")
	assert.NotContains(t, result, "is here. is here.")
}
