package ingestion

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPlacementQuery_WhenInputContainsDiagnosticsAndLongText_ExpectCompactSubjectTerms(t *testing.T) {
	t.Parallel()

	// Arrange
	words := make([]string, 30)
	for i := range words {
		words[i] = fmt.Sprintf("subject%02d", i)
	}
	input := PlacementBuildInput{
		Text:           "Сохрани материал из repository " + strings.Join(words, " ") + strings.Repeat(" important", 2),
		SourceKind:     "repository",
		ContentProfile: "repository_profile",
		Type:           "link",
	}

	// Act
	query := buildPlacementQuery(input)
	terms := strings.Fields(query)

	// Assert
	require.NotEmpty(t, terms)
	assert.LessOrEqual(t, len(terms), placementTermLimit)
	assert.Equal(t, "important", terms[0])
	assert.NotContains(t, terms, "repository")
	assert.NotContains(t, terms, "сохрани")
	assert.NotContains(t, terms, "материал")
	assert.NotContains(t, terms, "profile")
	assert.NotContains(t, terms, "link")
}

func TestPlacementTextContainsTerm_WhenOnlySubstringMatches_ExpectFalse(t *testing.T) {
	t.Parallel()

	assert.False(t, placementTextContainsTerm("agentic coding", "agent"))
	assert.True(t, placementTextContainsTerm("agent tools", "agent"))
	assert.True(t, placementTextContainsTerm("goroutines", "goroutine"))
}
