package translation_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/ingestion/translation"
)

func TestNeedsTranslation_WhenEnglishText_ExpectTrue(t *testing.T) {
	t.Parallel()

	content := `This is a long article written entirely in English. It contains multiple paragraphs
with various topics and discussions. The content is substantial enough to trigger
the translation heuristic. We need at least two hundred characters of text without
code blocks to pass the minimum length check.`

	assert.True(t, translation.NeedsTranslation(content))
}

func TestNeedsTranslation_WhenRussianText_ExpectFalse(t *testing.T) {
	t.Parallel()

	content := `Это длинная статья, написанная полностью на русском языке. Она содержит
несколько абзацев с различными темами и обсуждениями. Контент достаточно
объёмный, чтобы пройти проверку минимальной длины. Кириллицы здесь более
чем достаточно для порога в двадцать пять процентов.`

	assert.False(t, translation.NeedsTranslation(content))
}

func TestNeedsTranslation_WhenShortText_ExpectFalse(t *testing.T) {
	t.Parallel()

	content := "Short English text."
	assert.False(t, translation.NeedsTranslation(content))
}

func TestNeedsTranslation_WhenMixedWithCodeBlocks_ExpectTrue(t *testing.T) {
	t.Parallel()

	content := `This article has code blocks that should be excluded from the character count.

` + "```go\nfunc main() { fmt.Println(\"hello\") }\n```" + `

The actual text content is in English and long enough. We need sufficient
characters outside of code blocks. This paragraph adds more content to reach
the minimum threshold of two hundred characters for the heuristic to work.`

	assert.True(t, translation.NeedsTranslation(content))
}

func TestNeedsTranslation_WhenMostlyCodeBlocks_ExpectFalse(t *testing.T) {
	t.Parallel()

	content := `Short intro. ` + "```\n" + strings.Repeat("x", 500) + "\n```"
	assert.False(t, translation.NeedsTranslation(content))
}
