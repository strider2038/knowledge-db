package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/index"
)

func TestBuildContextText_WhenLongChunks_ExpectRespectsBudgetAndChunkPriority(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	long := strings.Repeat("word ", 2500)
	nodeResults := []index.SearchResult{
		{Path: "topic/node1"},
	}
	chunkResults := []index.ChunkSearchResult{
		{NodePath: "topic/node1", Heading: "Part 1", Content: long},
		{NodePath: "topic/node1", Heading: "Part 2", Content: long},
	}

	contextText := h.buildContextText(nodeResults, chunkResults)

	assert.Contains(t, contextText, "Part 1")
	assert.NotContains(t, contextText, "Part 2")
	assert.NotContains(t, contextText, "--- topic/node1 ---")
	assert.LessOrEqual(t, estimateContextTokens(contextText), ragContextTokenBudget)
}

func TestBuildSources_WhenChunkOnly_ExpectTitleFallbackToPath(t *testing.T) {
	t.Parallel()

	h := &Handler{dataPath: "/non-existent"}
	sources := h.buildSources(nil, []index.ChunkSearchResult{
		{NodePath: "topic/node1", Heading: "Intro", Content: "sample"},
	})

	if assert.Len(t, sources, 1) {
		assert.Equal(t, "topic/node1", sources[0].Path)
		assert.Equal(t, "topic/node1", sources[0].Title)
	}
}
