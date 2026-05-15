package index

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestCosineSimilarity_WhenIdentical_Expect1(t *testing.T) {
	t.Parallel()

	v := []float32{1, 0, 0}
	assert.InDelta(t, float32(1.0), cosineSimilarity(v, v), 0.001)
}

func TestCosineSimilarity_WhenOrthogonal_Expect0(t *testing.T) {
	t.Parallel()

	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	assert.InDelta(t, float32(0.0), cosineSimilarity(a, b), 0.001)
}

func TestCosineSimilarity_WhenOpposite_ExpectMinus1(t *testing.T) {
	t.Parallel()

	a := []float32{1, 0, 0}
	b := []float32{-1, 0, 0}
	assert.InDelta(t, float32(-1.0), cosineSimilarity(a, b), 0.001)
}

func TestCosineSimilarity_WhenDifferentLengths_Expect0(t *testing.T) {
	t.Parallel()

	a := []float32{1, 0}
	b := []float32{1, 0, 0}
	assert.InDelta(t, float32(0), cosineSimilarity(a, b), 1e-6)
}

func TestCosineSimilarity_WhenZeroVector_Expect0(t *testing.T) {
	t.Parallel()

	a := []float32{0, 0, 0}
	b := []float32{1, 0, 0}
	assert.InDelta(t, float32(0), cosineSimilarity(a, b), 1e-6)
}

func TestKeywordQueryTokens_WhenQuestionWithStopWords_ExpectImportantTerms(t *testing.T) {
	t.Parallel()

	tokens := keywordQueryTokens("какие альтернативы rag есть в базе?")

	assert.Equal(t, []string{"альтернативы", "rag"}, tokens)
}

func TestKeywordQueryTokens_WhenOnlyStopWords_ExpectFallback(t *testing.T) {
	t.Parallel()

	tokens := keywordQueryTokens("что где")

	assert.Equal(t, []string{"что", "где"}, tokens)
}

func TestKeywordQueryTokens_WhenDomainStopWords_ExpectRemoved(t *testing.T) {
	t.Parallel()

	tokens := keywordQueryTokens("как эффективно управлять контекстом ии")

	assert.Equal(t, []string{"эффективно", "управлять", "контекстом"}, tokens)
}

func TestExactTokenBoost_WhenRussianSuffixDiffers_ExpectTitleBoost(t *testing.T) {
	t.Parallel()

	boost := exactTokenBoost([]string{"контекстом"}, KeywordNodeHit{
		Title: "Context Mode: использование токенов контекста",
	})

	assert.InDelta(t, 6.0, boost, 1e-9)
}

func TestBuildSnippet_WhenCyrillicBoundary_ExpectValidUTF8(t *testing.T) {
	t.Parallel()

	content := strings.Repeat("человеческого текста ", 20) + "rag " + strings.Repeat("полезного контекста ", 20)

	snippet := buildSnippet(content, []string{"rag"})

	assert.True(t, utf8.ValidString(snippet))
	assert.NotContains(t, snippet, "�")
	assert.Contains(t, snippet, "rag")
}

func TestTruncateSnippet_WhenCyrillicBoundary_ExpectValidUTF8(t *testing.T) {
	t.Parallel()

	content := strings.Repeat("человеческого текста ", 40)

	snippet := truncateSnippet(content)

	assert.True(t, utf8.ValidString(snippet))
	assert.NotContains(t, snippet, "�")
	assert.True(t, strings.HasSuffix(snippet, "..."))
}
