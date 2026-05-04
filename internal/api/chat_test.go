package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/stretchr/testify/assert"

	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
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

func TestBuildChatSources_WhenFragments_ExpectMapped(t *testing.T) {
	t.Parallel()

	sources := buildChatSources([]index.HybridSearchResult{
		{
			Path:  "articles/sqlite",
			Title: "SQLite",
			Type:  "article",
			Fragments: []index.HybridFragment{
				{Heading: "Intro", Snippet: "sqlite snippet", MatchType: "keyword", Score: 1},
			},
		},
	})

	if assert.Len(t, sources, 1) {
		assert.Equal(t, "article", sources[0].Type)
		if assert.Len(t, sources[0].Fragments, 1) {
			assert.Equal(t, "Intro", sources[0].Fragments[0].Heading)
			assert.Equal(t, "keyword", sources[0].Fragments[0].MatchType)
		}
	}
}

func TestBuildHybridContextText_WhenFragments_ExpectFragmentContext(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	contextText := h.buildHybridContextText([]index.HybridSearchResult{
		{
			Path: "articles/sqlite",
			Fragments: []index.HybridFragment{
				{Heading: "Intro", Snippet: "sqlite snippet", Content: "full sqlite content"},
			},
		},
	})

	assert.Contains(t, contextText, "Fragment from articles/sqlite")
	assert.Contains(t, contextText, "sqlite snippet")
	assert.Contains(t, contextText, "full sqlite content")
}

func TestBuildHybridContextText_WhenEmpty_ExpectEmpty(t *testing.T) {
	t.Parallel()

	h := &Handler{}

	assert.Empty(t, h.buildHybridContextText(nil))
}

func TestBuildHybridContextText_WhenFragmentCoversNode_ExpectNoAnnotationDuplicate(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	contextText := h.buildHybridContextText([]index.HybridSearchResult{
		{
			Path:       "articles/sqlite",
			Annotation: "annotation fallback",
			Fragments: []index.HybridFragment{
				{Heading: "Intro", Content: "fragment content"},
			},
		},
	})

	assert.Contains(t, contextText, "fragment content")
	assert.NotContains(t, contextText, "annotation fallback")
}

func TestBuildHybridContextText_WhenNoFragments_ExpectAnnotationContext(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	contextText := h.buildHybridContextText([]index.HybridSearchResult{
		{Path: "notes/local", Annotation: "note annotation"},
	})

	assert.Contains(t, contextText, "notes/local")
	assert.Contains(t, contextText, "note annotation")
}

func TestRewriteSearchQuery_WhenEnabled_ExpectRewrittenQuery(t *testing.T) {
	t.Parallel()

	h := &Handler{
		chatClient: &rewriteMockClient{response: buildSearchRewriteResponse(t, "context mode контекст управление")},
		embeddingConfig: config.Embedding{
			SearchRewriteEnabled: true,
			ChatModel:            "gpt-4o-mini",
		},
	}

	query := h.rewriteSearchQuery(context.Background(), "как эффективно управлять контекстом ии")

	assert.Equal(t, "context mode контекст управление", query)
}

func TestRewriteSearchQuery_WhenDisabled_ExpectOriginalQuery(t *testing.T) {
	t.Parallel()

	h := &Handler{
		chatClient: &rewriteMockClient{response: buildSearchRewriteResponse(t, "rewritten")},
		embeddingConfig: config.Embedding{
			SearchRewriteEnabled: false,
		},
	}

	query := h.rewriteSearchQuery(context.Background(), "original")

	assert.Equal(t, "original", query)
}

func TestSanitizeSearchRewrite(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "context mode", sanitizeSearchRewrite("  `Query: context mode`  "))
	assert.Empty(t, sanitizeSearchRewrite(strings.Repeat("word ", 80)))
	assert.Empty(t, sanitizeSearchRewrite("one\n\ntwo"))
}

type rewriteMockClient struct {
	response *responses.Response
	err      error
}

func (m *rewriteMockClient) New(_ context.Context, _ responses.ResponseNewParams, _ ...option.RequestOption) (*responses.Response, error) {
	return m.response, m.err
}

func (m *rewriteMockClient) NewStreaming(context.Context, responses.ResponseNewParams, ...option.RequestOption) chatStream {
	return nil
}

func buildSearchRewriteResponse(tb testing.TB, text string) *responses.Response {
	tb.Helper()
	content := []responses.ResponseOutputMessageContentUnion{
		{Type: "output_text", Text: text},
	}
	outputItem := responses.ResponseOutputItemUnion{
		Type:    "message",
		Content: content,
	}
	data := map[string]any{
		"id":                 "resp-search-rewrite",
		"created_at":         float64(0),
		"error":              map[string]any{},
		"incomplete_details": map[string]any{},
		"instructions":       "",
		"metadata":           map[string]any{},
		"model":              "gpt-4o-mini",
		"object":             "response",
		"output":             []any{outputItem},
		"usage":              map[string]any{},
		"status":             "completed",
		"tool_choice":        "auto",
	}
	b, err := json.Marshal(data)
	if err != nil {
		tb.Fatalf("marshal: %v", err)
	}
	var resp responses.Response
	if err := json.Unmarshal(b, &resp); err != nil {
		tb.Fatalf("unmarshal: %v", err)
	}

	return &resp
}
