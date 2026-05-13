package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/chat"
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
		{Path: "notes/local", Title: "Local Note", Keywords: []string{"local", "workflow"}, Annotation: "note annotation"},
	})

	assert.Contains(t, contextText, "notes/local")
	assert.Contains(t, contextText, "Local Note")
	assert.Contains(t, contextText, "local, workflow")
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

func TestDetectChatMode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, chatModeMemory, detectChatMode("Сделай краткое резюме чата"))
	assert.Equal(t, chatModeRAG, detectChatMode("Что есть в базе про RAG?"))
	assert.Equal(t, chatModeHybrid, detectChatMode("Расскажи подробнее"))
}

func TestPostChat_WhenMemoryMode_ExpectNoSourcesAndLLMAnswer(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	h := NewHandler(tmp, nil)
	chatStore, err := chat.NewStore(filepath.Join(tmp, "chat.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = chatStore.Close() })
	session, err := chatStore.CreateSession(context.Background(), "s1", "Chat")
	require.NoError(t, err)
	require.NoError(t, chatStore.AddMessage(context.Background(), session.ID, "user", "обсудили sqlite", false))
	h.SetChatStore(chatStore)

	store, err := index.NewIndexStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	h.SetIndexComponents(store, nil, &chatTestEmbeddingProvider{}, config.Embedding{
		Enabled:   true,
		APIKey:    "key",
		APIURL:    "http://localhost",
		Model:     "text-embedding-3-small",
		ChatModel: "gpt-4o",
	})
	h.chatClient = &rewriteMockClient{chatTokens: []string{"Краткое резюме"}}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/chat", strings.NewReader(`{"session_id":"s1","message":"Сделай краткое резюме чата"}`))
	rec := httptest.NewRecorder()

	h.PostChat(rec, req)

	assert.Equal(t, 200, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `"sources": []`)
	assert.Contains(t, body, "Краткое резюме")
	assert.Contains(t, body, "data: [DONE]")
}

func TestCompactKnowledgeBaseQuery_WhenKnowledgeBaseQuestion_ExpectSearchTerms(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "RAG", compactKnowledgeBaseQuery("Что есть про RAG в базе знаний?"))
	assert.Equal(t, "RAG", compactKnowledgeBaseQuery("Какие статьи есть про RAG?"))
	assert.Equal(t, "dependency injection Go", compactKnowledgeBaseQuery("Найди в базе dependency injection Go"))
}

func TestFilterRelevantResults(t *testing.T) {
	t.Parallel()

	results := []index.HybridSearchResult{
		{Path: "docs/rag", Title: "RAG", Annotation: "retrieval augmented generation", Score: 0.82},
		{Path: "docs/other", Title: "Other", Annotation: "random topic", Score: 0.79},
		{Path: "docs/low", Title: "Low", Annotation: "rag maybe", Score: 0.04},
	}
	filtered := filterRelevantResults("что есть про rag", results)
	if assert.Len(t, filtered, 1) {
		assert.Equal(t, "docs/rag", filtered[0].Path)
	}
}

func TestFilterRelevantResults_WhenTopicInventory_ExpectKeepsLexicalSources(t *testing.T) {
	t.Parallel()

	results := []index.HybridSearchResult{
		{Path: "ai/wiki", Title: "Wiki", Annotation: "RAG and knowledge base", Score: 0.74, SourceKinds: []string{"keyword_chunk"}},
		{Path: "ai/blockify", Title: "Blockify: Агентная оптимизация данных для RAG", Score: 0.23, SourceKinds: []string{"keyword"}},
		{Path: "ai/rag-budget", Title: "RAG бюджет", Score: 0.21, SourceKinds: []string{"keyword"}},
		{Path: "ai/rlm", Title: "Рекурсивная модель и RAG", Score: 0.09, SourceKinds: []string{"keyword"}},
		{Path: "programming/antfly", Title: "Antfly", Annotation: "BM25 vectors RAG", Score: 0.08, SourceKinds: []string{"keyword"}},
		{Path: "docs/random", Title: "Random", Annotation: "unrelated", Score: 0.95, SourceKinds: []string{"vector_node"}},
	}

	filtered := filterRelevantResults("Что есть про RAG в базе знаний?", results)

	require.Len(t, filtered, 5)
	assert.Equal(t, "ai/wiki", filtered[0].Path)
	assert.Equal(t, "programming/antfly", filtered[4].Path)
}

type rewriteMockClient struct {
	response   *responses.Response
	err        error
	chatTokens []string
}

func (m *rewriteMockClient) New(_ context.Context, _ responses.ResponseNewParams, _ ...option.RequestOption) (*responses.Response, error) {
	return m.response, m.err
}

func (m *rewriteMockClient) NewStreaming(context.Context, responses.ResponseNewParams, ...option.RequestOption) chatStream {
	return nil
}

func (m *rewriteMockClient) NewChatStreaming(context.Context, openai.ChatCompletionNewParams, ...option.RequestOption) chatCompletionStream {
	return &mockChatCompletionStream{tokens: m.chatTokens}
}

type chatTestEmbeddingProvider struct{}

func (p *chatTestEmbeddingProvider) Embed(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{1, 0, 0}
	}

	return result, nil
}

type mockChatCompletionStream struct {
	tokens []string
	index  int
}

func (s *mockChatCompletionStream) Next() bool {
	if s.index >= len(s.tokens) {
		return false
	}
	s.index++

	return true
}

func (s *mockChatCompletionStream) Current() openai.ChatCompletionChunk {
	token := s.tokens[s.index-1]

	return openai.ChatCompletionChunk{
		Choices: []openai.ChatCompletionChunkChoice{
			{Delta: openai.ChatCompletionChunkChoiceDelta{Content: token}},
		},
	}
}

func (s *mockChatCompletionStream) Err() error {
	return nil
}

func (s *mockChatCompletionStream) Close() error {
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
