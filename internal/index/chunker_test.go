package index

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkText_WhenArticleWithHeadings_ExpectChunksBySection(t *testing.T) {
	t.Parallel()

	introContent := strings.Repeat("intro content ", 80)
	detailsContent := strings.Repeat("detailed info ", 80)
	conclusionContent := strings.Repeat("final thoughts ", 80)
	body := "## Intro\n" + introContent + "\n\n## Details\n" + detailsContent + "\n\n## Conclusion\n" + conclusionContent
	chunks := ChunkText(body)

	require.Len(t, chunks, 3)
	assert.Equal(t, "Intro", chunks[0].Heading)
	assert.Contains(t, chunks[0].Content, "intro content")
	assert.Equal(t, "Details", chunks[1].Heading)
	assert.Contains(t, chunks[1].Content, "detailed info")
	assert.Equal(t, "Conclusion", chunks[2].Heading)
	assert.Contains(t, chunks[2].Content, "final thoughts")
}

func TestChunkText_WhenArticleNoHeadings_ExpectSingleChunk(t *testing.T) {
	t.Parallel()

	body := "Some content without any headings.\nJust plain text."
	chunks := ChunkText(body)

	require.Len(t, chunks, 1)
	assert.Empty(t, chunks[0].Heading)
	assert.Contains(t, chunks[0].Content, "Some content")
}

func TestChunkText_WhenEmptyBody_ExpectNil(t *testing.T) {
	t.Parallel()

	chunks := ChunkText("")
	assert.Nil(t, chunks)
}

func TestChunkText_WhenWhitespaceOnly_ExpectNil(t *testing.T) {
	t.Parallel()

	chunks := ChunkText("   \n  \n  ")
	assert.Nil(t, chunks)
}

func TestChunkText_WhenLongSection_ExpectSplitByParagraphs(t *testing.T) {
	t.Parallel()

	paragraph := strings.Repeat("word ", 200)
	body := "## Long Section\n" + paragraph + "\n\n" + paragraph + "\n\n" + paragraph
	chunks := ChunkText(body)

	assert.Greater(t, len(chunks), 1, "expected multiple chunks for long section")
	for _, c := range chunks {
		assert.Equal(t, "Long Section", c.Heading)
	}
}

func TestChunkText_WhenShortSection_ExpectMergedWithNext(t *testing.T) {
	t.Parallel()

	shortSection := "## Short\n" + strings.Repeat("word ", 20)
	longSection := "## Long Enough\n" + strings.Repeat("content ", 150)
	body := shortSection + "\n\n" + longSection
	chunks := ChunkText(body)

	require.Len(t, chunks, 1)
	assert.Contains(t, chunks[0].Content, strings.Repeat("word ", 20))
	assert.Contains(t, chunks[0].Content, strings.Repeat("content ", 150))
}

func TestChunkText_WhenMultipleShortSections_ExpectMergedCumulatively(t *testing.T) {
	t.Parallel()

	body := "## A\n" + strings.Repeat("a ", 30) + "\n\n" +
		"## B\n" + strings.Repeat("b ", 30) + "\n\n" +
		"## C\n" + strings.Repeat("c ", 30) + "\n\n" +
		"## D\n" + strings.Repeat("d ", 30)
	chunks := ChunkText(body)

	assert.Less(t, len(chunks), 4, "expected some merging of short sections")
}

func TestChunkText_WhenSingleParagraphExceedsTokenBudget_ExpectSeveralChunks(t *testing.T) {
	t.Parallel()

	body := "## Section\n" + strings.Repeat("word ", 2000)
	chunks := ChunkText(body)

	require.Greater(t, len(chunks), 1)
	for _, c := range chunks {
		assert.Equal(t, "Section", c.Heading)
		assert.LessOrEqual(t, len(c.Content), maxChunkContentBytes)
		assert.LessOrEqual(t, estimateTokens(c.Content), 500)
	}
}

func TestChunkText_WhenUnbrokenTextExceedsLimit_ExpectByteSplit(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("я", 20000)
	chunks := ChunkText(body)

	require.Greater(t, len(chunks), 1)
	var joined strings.Builder
	for _, c := range chunks {
		assert.LessOrEqual(t, len(c.Content), maxChunkContentBytes)
		assert.True(t, utf8.ValidString(c.Content))
		joined.WriteString(c.Content)
	}
	assert.Equal(t, body, joined.String())
}

func TestEstimateTokens(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, estimateTokens(""))
	assert.Equal(t, 1, estimateTokens("hello"))
	assert.Greater(t, estimateTokens(strings.Repeat("word ", 100)), 100)
}
