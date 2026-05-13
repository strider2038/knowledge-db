package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_WhenPlacementContextProvided_ExpectCompactContextWithoutGlobalKeywords(t *testing.T) {
	t.Parallel()

	// Arrange
	input := ProcessInput{
		PlacementContext: PlacementContext{
			Source: "fallback",
			ThemeMap: []ThemeSummary{
				{Path: "go/concurrency", NodeCount: 3, TopKeywords: []string{"goroutines"}},
			},
			CandidateThemes: []ThemeCandidate{
				{Path: "go/concurrency", Score: 12, NodeCount: 3, Reasons: []string{"similar_node"}},
			},
			CandidateKeywords: []KeywordCandidate{
				{Keyword: "goroutines", Score: 8, Frequency: 2, Themes: []string{"go/concurrency"}},
			},
			SimilarNodes: []SimilarNode{
				{Path: "go/concurrency/goroutines", Title: "Goroutines", Keywords: []string{"goroutines"}},
			},
		},
	}

	// Act
	prompt := buildSystemPrompt(input)

	// Assert
	assert.Contains(t, prompt, "## Placement context")
	assert.Contains(t, prompt, "### Candidate themes")
	assert.Contains(t, prompt, "go/concurrency")
	assert.Contains(t, prompt, "goroutines")
	assert.NotContains(t, prompt, "## Existing keywords")
	assert.NotContains(t, prompt, "## Existing themes")
}

func TestBuildSystemPrompt_WhenExplicitThemePathProvided_ExpectPriorityInstruction(t *testing.T) {
	t.Parallel()

	// Arrange
	input := ProcessInput{
		PlacementContext: PlacementContext{
			ExplicitThemePath: "go/concurrency",
			CandidateThemes: []ThemeCandidate{
				{Path: "go/concurrency", Reasons: []string{"explicit_user_instruction"}},
			},
		},
	}

	// Act
	prompt := buildSystemPrompt(input)

	// Assert
	assert.Contains(t, prompt, "эта инструкция важнее автоматического shortlist")
	assert.Contains(t, prompt, "Explicit user theme instruction detected: go/concurrency")
}
