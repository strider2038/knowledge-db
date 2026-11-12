package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
)

type mockResponsesClient struct {
	calls    []responses.ResponseNewParams
	response *responses.Response
	err      error
}

func (m *mockResponsesClient) New(_ context.Context, params responses.ResponseNewParams, _ ...option.RequestOption) (*responses.Response, error) {
	m.calls = append(m.calls, params)

	return m.response, m.err
}

type mockContentFetcher struct {
	result *fetcher.FetchResult
	err    error
	calls  int
}

func (m *mockContentFetcher) Fetch(_ context.Context, _ string) (*fetcher.FetchResult, error) {
	m.calls++

	return m.result, m.err
}

func buildFunctionCallResponse(tb testing.TB, id, callID, name, arguments string) *responses.Response {
	tb.Helper()
	data := `{"id":"` + id + `","created_at":0,"error":{},"incomplete_details":{},"instructions":"","metadata":{},"model":"gpt-4o","object":"response","parallel_tool_calls":false,"temperature":1,"output":[{"type":"function_call","id":"` + callID + `","call_id":"` + callID + `","name":"` + name + `","arguments":` + jsonString(tb, arguments) + `,"status":"completed"}],"usage":{},"status":"completed","tool_choice":"auto"}`
	var resp responses.Response
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		tb.Fatalf("unmarshal: %v", err)
	}

	return &resp
}

func buildCreateNodeResponse(tb testing.TB, id, callID string, args map[string]any) *responses.Response {
	tb.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		tb.Fatalf("marshal args: %v", err)
	}

	return buildFunctionCallResponse(tb, id, callID, "create_node", string(argsJSON))
}

func jsonString(tb testing.TB, s string) string {
	tb.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		tb.Fatalf("marshal string: %v", err)
	}

	return string(b)
}

func TestOpenAIOrchestrator_WhenNoteInput_ExpectDirectCreateNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	createNodeArgs := map[string]any{
		"keywords":   []any{"go", "patterns"},
		"annotation": "A note about Go patterns",
		"theme_path": "go/patterns",
		"slug":       "go-patterns-note",
		"type":       "note",
		"content":    "Some notes about Go patterns.",
		"title":      "Go Patterns Note",
	}
	mockClient := &mockResponsesClient{
		response: buildCreateNodeResponse(t, "resp1", "call1", createNodeArgs),
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", nil)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text: "Some notes about Go patterns.",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"go", "patterns"}, result.Keywords)
	assert.Equal(t, "go/patterns", result.ThemePath)
	assert.Equal(t, "note", result.Type)
	assert.Equal(t, "go-patterns-note", result.Slug)
	assert.Len(t, mockClient.calls, 1)
}

func TestOpenAIOrchestrator_WhenArticleInput_ExpectFetchThenCreateNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fetchArgs := `{"url":"https://habr.com/article/123"}`
	firstResp := buildFunctionCallResponse(t, "resp1", "call1", "fetch_url_content", fetchArgs)

	createNodeArgs := map[string]any{
		"keywords":   []any{"goroutines", "memory"},
		"annotation": "Article about goroutine leaks",
		"theme_path": "go/concurrency",
		"slug":       "goroutine-leak",
		"type":       "article",
		"content":    "# Goroutine Leaks\n\nContent here.",
		"title":      "Goroutine Leaks",
		"source_url": "https://habr.com/article/123",
		// source_author не передан LLM — должен подставиться из FetchResult
	}
	secondResp := buildCreateNodeResponse(t, "resp2", "call2", createNodeArgs)

	callCount := 0
	mockFetcher := &mockContentFetcher{
		result: &fetcher.FetchResult{
			Title:   "Goroutine Leaks",
			Content: "# Goroutine Leaks\n\nContent here.",
			Author:  "Иван Петров",
		},
	}

	var seqClient sequenceMockClient
	seqClient.responses = []*responses.Response{firstResp, secondResp}
	seqClient.idx = &callCount

	orch := llm.NewTestOrchestrator(&seqClient, "gpt-4o", mockFetcher)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text: "https://habr.com/article/123",
	})

	require.NoError(t, err)
	assert.Equal(t, "article", result.Type)
	assert.Equal(t, "goroutine-leak", result.Slug)
	assert.Equal(t, "https://habr.com/article/123", result.SourceURL)
	assert.Equal(t, "Иван Петров", result.SourceAuthor, "SourceAuthor должен подставиться из FetchResult")
	assert.Equal(t, 2, callCount)
}

func TestOpenAIOrchestrator_WhenDigestNoteAfterFetch_ExpectGeneratedContentPreserved(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fetchArgs := `{"url":"https://example.com/blog/designing-go-libraries"}`
	firstResp := buildFunctionCallResponse(t, "resp1", "call1", "fetch_url_content", fetchArgs)
	createNodeArgs := map[string]any{
		"keywords":        []any{"go"},
		"annotation":      "Conceptual digest",
		"theme_path":      "go/design",
		"slug":            "designing-go-libraries",
		"type":            "note",
		"source_kind":     "article",
		"content_profile": "conceptual_digest",
		"content":         "## Главная идея\n\nСжатая выжимка.",
		"title":           "Designing Go Libraries",
		"source_url":      "https://example.com/blog/designing-go-libraries",
	}
	secondResp := buildCreateNodeResponse(t, "resp2", "call2", createNodeArgs)

	callCount := 0
	mockFetcher := &mockContentFetcher{
		result: &fetcher.FetchResult{
			Title:   "Designing Go Libraries",
			Content: "# Full Article\n\nFull source content that must not be stored for digest.",
		},
	}
	var seqClient sequenceMockClient
	seqClient.responses = []*responses.Response{firstResp, secondResp}
	seqClient.idx = &callCount
	orch := llm.NewTestOrchestrator(&seqClient, "gpt-4o", mockFetcher)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text:            "https://example.com/blog/designing-go-libraries",
		SourceURL:       "https://example.com/blog/designing-go-libraries",
		SourceKind:      "article",
		ContentProfile:  "conceptual_digest",
		RecommendedType: "note",
	})

	require.NoError(t, err)
	assert.Equal(t, "note", result.Type)
	assert.Equal(t, "article", result.SourceKind)
	assert.Equal(t, "conceptual_digest", result.ContentProfile)
	assert.Equal(t, "## Главная идея\n\nСжатая выжимка.", result.Content)
	assert.NotContains(t, result.Content, "Full source content")
}

func TestOpenAIOrchestrator_WhenArticleAndWrongLLMSourceURL_ExpectCanonicalFromInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	canonical := "https://example.com/article/123"
	fetchArgs := `{"url":"` + canonical + `"}`
	firstResp := buildFunctionCallResponse(t, "resp1", "call1", "fetch_url_content", fetchArgs)

	createNodeArgs := map[string]any{
		"keywords":   []any{"goroutines", "memory"},
		"annotation": "Article about goroutine leaks",
		"theme_path": "go/concurrency",
		"slug":       "goroutine-leak",
		"type":       "article",
		"content":    "# Goroutine Leaks\n\nContent here.",
		"title":      "Goroutine Leaks",
		"source_url": "http://claude.md/",
	}
	secondResp := buildCreateNodeResponse(t, "resp2", "call2", createNodeArgs)

	callCount := 0
	mockFetcher := &mockContentFetcher{
		result: &fetcher.FetchResult{
			Title:   "Goroutine Leaks",
			Content: "# Goroutine Leaks\n\nContent here.",
			Author:  "Иван Петров",
		},
	}

	var seqClient sequenceMockClient
	seqClient.responses = []*responses.Response{firstResp, secondResp}
	seqClient.idx = &callCount

	orch := llm.NewTestOrchestrator(&seqClient, "gpt-4o", mockFetcher)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text:      "URL: " + canonical + "\nTitle: Goroutine Leaks\n\n# ...",
		SourceURL: canonical,
	})

	require.NoError(t, err)
	assert.Equal(t, "article", result.Type)
	assert.NotEqual(t, "http://claude.md/", result.SourceURL, "не ссылка из тела статьи")
	assert.Equal(t, canonical, result.SourceURL, "должен подставиться URL импорта после нормализации")
	assert.Equal(t, 2, callCount)
}

type sequenceMockClient struct {
	responses []*responses.Response
	idx       *int
}

var errNoMoreResponses = errors.New("no more mock responses")

func (s *sequenceMockClient) New(_ context.Context, _ responses.ResponseNewParams, _ ...option.RequestOption) (*responses.Response, error) {
	i := *s.idx
	*s.idx++
	if i >= len(s.responses) {
		return nil, errNoMoreResponses
	}

	return s.responses[i], nil
}

func TestOpenAIOrchestrator_WhenSearchPlacementCandidatesToolCall_ExpectLoopContinuesToCreateNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	searchArgs := `{"query":"goroutine leaks","source_kind":"article","content_profile":"conceptual_digest","type":"note"}`
	firstResp := buildFunctionCallResponse(t, "resp1", "call1", "search_placement_candidates", searchArgs)
	createNodeArgs := map[string]any{
		"keywords":   []any{"goroutines"},
		"annotation": "A note about goroutine leaks",
		"theme_path": "go/concurrency",
		"slug":       "goroutine-leaks",
		"type":       "note",
		"content":    "Goroutine leaks note",
		"title":      "Goroutine Leaks",
	}
	secondResp := buildCreateNodeResponse(t, "resp2", "call2", createNodeArgs)

	callCount := 0
	seqClient := &sequenceMockClient{
		responses: []*responses.Response{firstResp, secondResp},
		idx:       &callCount,
	}
	toolCalled := false
	orch := llm.NewTestOrchestrator(seqClient, "gpt-4o", nil)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text: "goroutine leaks",
		PlacementSearch: func(_ context.Context, req llm.SearchPlacementCandidatesRequest) (*llm.PlacementContext, error) {
			toolCalled = true
			assert.Equal(t, "goroutine leaks", req.Query)
			assert.Equal(t, "article", req.SourceKind)
			assert.Equal(t, "conceptual_digest", req.ContentProfile)
			assert.Equal(t, "note", req.Type)

			return &llm.PlacementContext{
				Source: "fallback",
				CandidateThemes: []llm.ThemeCandidate{
					{Path: "go/concurrency", Score: 10},
				},
				CandidateKeywords: []llm.KeywordCandidate{
					{Keyword: "goroutines", Score: 8},
				},
			}, nil
		},
	})

	require.NoError(t, err)
	assert.True(t, toolCalled)
	assert.Equal(t, "go/concurrency", result.ThemePath)
	assert.Equal(t, "goroutine-leaks", result.Slug)
	assert.Equal(t, 2, callCount)
}

func TestOpenAIOrchestrator_WhenTypeHint_ExpectInInstructions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	createNodeArgs := map[string]any{
		"keywords":   []any{"go"},
		"annotation": "A note",
		"theme_path": "go",
		"slug":       "test",
		"type":       "article",
		"content":    "",
		"title":      "Test",
	}
	mockClient := &mockResponsesClient{
		response: buildCreateNodeResponse(t, "resp1", "call1", createNodeArgs),
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", nil)

	_, err := orch.Process(ctx, llm.ProcessInput{
		Text:     "https://example.com/article",
		TypeHint: "article",
	})

	require.NoError(t, err)
	require.Len(t, mockClient.calls, 1)
	instructions := mockClient.calls[0].Instructions.Or("")
	assert.Contains(t, instructions, "Пользователь указал тип: article")
}

func TestOpenAIOrchestrator_WhenTypeHintConflictsWithLLM_ExpectTypeHintWins(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	createNodeArgs := map[string]any{
		"keywords":   []any{"openai"},
		"annotation": "A note",
		"theme_path": "ai/tools",
		"slug":       "harness-engineering",
		"type":       "note",
		"content":    "Summary",
		"title":      "Harness Engineering",
		"source_url": "https://openai.com/ru-RU/index/harness-engineering/",
	}
	mockClient := &mockResponsesClient{
		response: buildCreateNodeResponse(t, "resp1", "call1", createNodeArgs),
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", nil)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text:     "https://openai.com/ru-RU/index/harness-engineering/",
		TypeHint: "article",
	})

	require.NoError(t, err)
	assert.Equal(t, "article", result.Type)
}

func TestOpenAIOrchestrator_WhenForcedArticleAndEmptyContent_ExpectFetchBySourceURL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	createNodeArgs := map[string]any{
		"keywords":   []any{"openai"},
		"annotation": "A note",
		"theme_path": "ai/tools",
		"slug":       "harness-engineering",
		"type":       "note",
		"content":    "",
		"title":      "Harness Engineering",
		"source_url": "https://openai.com/ru-RU/index/harness-engineering/",
	}
	mockClient := &mockResponsesClient{
		response: buildCreateNodeResponse(t, "resp1", "call1", createNodeArgs),
	}
	mockFetcher := &mockContentFetcher{
		result: &fetcher.FetchResult{
			Title:   "Harness Engineering",
			Content: "# Harness Engineering\n\nFull article text.",
			Author:  "OpenAI",
		},
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", mockFetcher)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text:        "https://openai.com/ru-RU/index/harness-engineering/",
		TypeHint:    "article",
		ContentMode: "full_fetch",
	})

	require.NoError(t, err)
	assert.Equal(t, "article", result.Type)
	assert.Equal(t, "# Harness Engineering\n\nFull article text.", result.Content)
	assert.Equal(t, "OpenAI", result.SourceAuthor)
	assert.Equal(t, 1, mockFetcher.calls)
}

func TestOpenAIOrchestrator_WhenForcedArticleAndTruncatedPreview_ExpectFetchBySourceURL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	createNodeArgs := map[string]any{
		"keywords":   []any{"openai"},
		"annotation": "A note",
		"theme_path": "ai/tools",
		"slug":       "harness-engineering",
		"type":       "note",
		"content":    "Преамбула\n\n[...контент усечён для анализа аннотации...]",
		"title":      "Harness Engineering",
		"source_url": "https://openai.com/ru-RU/index/harness-engineering/",
	}
	mockClient := &mockResponsesClient{
		response: buildCreateNodeResponse(t, "resp1", "call1", createNodeArgs),
	}
	mockFetcher := &mockContentFetcher{
		result: &fetcher.FetchResult{
			Title:   "Harness Engineering",
			Content: "# Harness Engineering\n\nFull article text.",
			Author:  "OpenAI",
		},
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", mockFetcher)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text:        "https://openai.com/ru-RU/index/harness-engineering/",
		TypeHint:    "article",
		ContentMode: "full_fetch",
	})

	require.NoError(t, err)
	assert.Equal(t, "article", result.Type)
	assert.Equal(t, "# Harness Engineering\n\nFull article text.", result.Content)
	assert.Equal(t, "OpenAI", result.SourceAuthor)
	assert.Equal(t, 1, mockFetcher.calls)
}

func TestOpenAIOrchestrator_WhenEmptySourceURLAndWrongLLMSourceURL_ExpectURLFromMessageText(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	createNodeArgs := map[string]any{
		"keywords":   []any{"SAST", "LLM"},
		"annotation": "Скилл для анализа.",
		"theme_path": "ai/security",
		"slug":       "llm-sast-scanner",
		"type":       "link",
		"content":    "",
		"title":      "llm-sast-scanner",
		"source_url": "https://docs.github.com",
	}
	mockClient := &mockResponsesClient{
		response: buildCreateNodeResponse(t, "resp1", "call1", createNodeArgs),
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", nil)

	// Как при обычном сообщении в Telegram-бот: SourceURL не передаётся (см. processIngest(..., "", "")).
	text := `llm-sast-scanner: универсальный SAST-скилл.

исходники (https://github.com/SunWeb3Sec/llm-sast-scanner)`

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text: text,
	})

	require.NoError(t, err)
	assert.Equal(t, "link", result.Type)
	assert.Equal(t, "https://github.com/SunWeb3Sec/llm-sast-scanner", result.SourceURL)
}

func TestOpenAIOrchestrator_WhenNoteWithForwardedLink_ExpectTelegramSourceURL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	createNodeArgs := map[string]any{
		"keywords":   []any{"SAST", "LLM"},
		"annotation": "Скилл для анализа.",
		"theme_path": "ai/security",
		"slug":       "llm-sast-scanner-note",
		"type":       "note",
		"content":    "Текст заметки с ссылкой в entities",
		"title":      "llm-sast-scanner",
		"source_url": "https://docs.github.com",
	}
	mockClient := &mockResponsesClient{
		response: buildCreateNodeResponse(t, "resp1", "call1", createNodeArgs),
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", nil)

	text := `**llm-sast-scanner**: описание.

[исходники](https://github.com/SunWeb3Sec/llm-sast-scanner)`

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text:      text,
		SourceURL: "https://t.me/vibe_coding/157278",
	})

	require.NoError(t, err)
	assert.Equal(t, "note", result.Type)
	assert.Equal(t, "https://t.me/vibe_coding/157278", result.SourceURL)
}

func TestOpenAIOrchestrator_WhenTelegramDeliveryAndWrongLLMSourceURL_ExpectURLFromMessageText(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	createNodeArgs := map[string]any{
		"keywords":   []any{"SAST", "LLM"},
		"annotation": "Скилл для анализа.",
		"theme_path": "ai/security",
		"slug":       "llm-sast-scanner",
		"type":       "link",
		"content":    "",
		"title":      "llm-sast-scanner",
		"source_url": "https://docs.github.com",
	}
	mockClient := &mockResponsesClient{
		response: buildCreateNodeResponse(t, "resp1", "call1", createNodeArgs),
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", nil)

	body := `llm-sast-scanner: универсальный SAST-скилл.

исходники (https://github.com/SunWeb3Sec/llm-sast-scanner)`
	text := "Telegram-канал (откуда получен контент, НЕ является source_url ресурса): https://t.me/vibe_coding, автор: Вайб-кодинг\n\n" + body

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text:         text,
		SourceURL:    "https://t.me/vibe_coding",
		SourceAuthor: "Вайб-кодинг",
	})

	require.NoError(t, err)
	assert.Equal(t, "link", result.Type)
	assert.Equal(t, "https://github.com/SunWeb3Sec/llm-sast-scanner", result.SourceURL)
}

func TestOpenAIOrchestrator_WhenTextOnlyResponse_ExpectNudgeThenCreateNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	textResp := buildMessageResponse(t, "Сохраню как узел базы знаний.")
	createNodeArgs := map[string]any{
		"keywords":   []any{"go"},
		"annotation": "A note",
		"theme_path": "go",
		"slug":       "test-note",
		"type":       "note",
		"content":    "Some note content",
		"title":      "Test",
	}
	secondResp := buildCreateNodeResponse(t, "resp2", "call2", createNodeArgs)

	callCount := 0
	seqClient := &sequenceMockClient{
		responses: []*responses.Response{textResp, secondResp},
		idx:       &callCount,
	}
	orch := llm.NewTestOrchestrator(seqClient, "gpt-4o", nil)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text: "Some note content",
	})

	require.NoError(t, err)
	assert.Equal(t, "note", result.Type)
	assert.Equal(t, "test-note", result.Slug)
	assert.Equal(t, 2, callCount)
}

func TestOpenAIOrchestrator_WhenLinkInput_ExpectFetchMetaThenCreateNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fetchMetaArgs := `{"url":"https://pkg.go.dev/net/http"}`
	firstResp := buildFunctionCallResponse(t, "resp1", "call1", "fetch_url_meta", fetchMetaArgs)

	createNodeArgs := map[string]any{
		"keywords":   []any{"go", "http", "stdlib"},
		"annotation": "Go standard library HTTP package",
		"theme_path": "go/stdlib",
		"slug":       "net-http",
		"type":       "link",
		"content":    "",
		"title":      "net/http - Go",
		"source_url": "https://pkg.go.dev/net/http",
	}
	secondResp := buildCreateNodeResponse(t, "resp2", "call2", createNodeArgs)

	callCount := 0
	seqClient := &sequenceMockClient{
		responses: []*responses.Response{firstResp, secondResp},
		idx:       &callCount,
	}

	orch := llm.NewTestOrchestrator(seqClient, "gpt-4o", nil)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text: "https://pkg.go.dev/net/http",
	})

	require.NoError(t, err)
	assert.Equal(t, "link", result.Type)
	assert.Equal(t, "net-http", result.Slug)
	assert.Equal(t, 2, callCount)
}

func TestOpenAIOrchestrator_WhenToolCallOmitsID_ExpectIDInFollowUp(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	first := unmarshalResponse(t, `{
		"id":"resp1","created_at":0,"error":{},"incomplete_details":{},"instructions":"","metadata":{},
		"model":"gpt-5","object":"response","status":"completed","tool_choice":"auto","usage":{},
		"output":[{
			"type":"function_call",
			"call_id":"call_missing",
			"name":"fetch_url_content",
			"arguments":"{\"url\":\"https://example.com/article\"}",
			"status":"completed"
		}]
	}`)
	second := buildCreateNodeResponse(t, "resp2", "call2", map[string]any{
		"keywords":   []any{"linux"},
		"annotation": "Article",
		"theme_path": "os/linux",
		"slug":       "linux-fuzzing",
		"type":       "note",
		"content":    "Digest",
		"title":      "Linux fuzzing",
	})
	client := &recordingResponsesClient{responses: []*responses.Response{first, second}}
	orch := llm.NewTestOrchestrator(client, "gpt-5", &mockContentFetcher{
		result: &fetcher.FetchResult{Title: "Linux", Content: "body"},
	})

	_, err := orch.Process(ctx, llm.ProcessInput{Text: "https://example.com/article"})

	require.NoError(t, err)
	require.Len(t, client.calls, 2)
	followUp := mustMarshalInput(t, client.calls[1])
	assert.Contains(t, followUp, `"id":"fc_call_missing"`)
	assert.Contains(t, followUp, `"call_id":"call_missing"`)
	assert.Contains(t, followUp, "function_call_output")
}

func TestOpenAIOrchestrator_WhenReasoningPrecedesToolCall_ExpectReasoningReplayed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	first := unmarshalResponse(t, `{
		"id":"resp1","created_at":0,"error":{},"incomplete_details":{},"instructions":"","metadata":{},
		"model":"gpt-5","object":"response","status":"completed","tool_choice":"auto","usage":{},
		"output":[
			{"type":"reasoning","id":"rs_keep","summary":[{"type":"summary_text","text":"need the page"}],"status":"completed"},
			{"type":"function_call","id":"fc_1","call_id":"call_1","name":"fetch_url_content","arguments":"{\"url\":\"https://example.com/article\"}","status":"completed"}
		]
	}`)
	second := buildCreateNodeResponse(t, "resp2", "call2", map[string]any{
		"keywords":   []any{"linux"},
		"annotation": "Article",
		"theme_path": "os/linux",
		"slug":       "linux-fuzzing",
		"type":       "note",
		"content":    "Digest",
		"title":      "Linux fuzzing",
	})
	client := &recordingResponsesClient{responses: []*responses.Response{first, second}}
	orch := llm.NewTestOrchestrator(client, "gpt-5", &mockContentFetcher{
		result: &fetcher.FetchResult{Title: "Linux", Content: "body"},
	})

	_, err := orch.Process(ctx, llm.ProcessInput{Text: "https://example.com/article"})

	require.NoError(t, err)
	require.Len(t, client.calls, 2)
	followUp := mustMarshalInput(t, client.calls[1])
	reasoningAt := strings.Index(followUp, `"id":"rs_keep"`)
	callAt := strings.Index(followUp, `"call_id":"call_1"`)
	outputAt := strings.Index(followUp, "function_call_output")
	require.NotEqual(t, -1, reasoningAt)
	require.NotEqual(t, -1, callAt)
	require.NotEqual(t, -1, outputAt)
	assert.Less(t, reasoningAt, callAt)
	assert.Less(t, callAt, outputAt)
}

func TestOpenAIOrchestrator_WhenReasoningWithoutToolCall_ExpectNudgeWithoutReasoning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	first := unmarshalResponse(t, `{
		"id":"resp1","created_at":0,"error":{},"incomplete_details":{},"instructions":"","metadata":{},
		"model":"gpt-5","object":"response","status":"completed","tool_choice":"auto","usage":{},
		"output":[
			{"type":"reasoning","id":"rs_orphan","summary":[{"type":"summary_text","text":"thinking"}],"status":"completed"},
			{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Сохраню как заметку."}]}
		]
	}`)
	second := buildCreateNodeResponse(t, "resp2", "call2", map[string]any{
		"keywords":   []any{"go"},
		"annotation": "A note",
		"theme_path": "go",
		"slug":       "note",
		"type":       "note",
		"content":    "text",
		"title":      "Note",
	})
	client := &recordingResponsesClient{responses: []*responses.Response{first, second}}
	orch := llm.NewTestOrchestrator(client, "gpt-5", nil)

	_, err := orch.Process(ctx, llm.ProcessInput{Text: "Короткая заметка"})

	require.NoError(t, err)
	require.Len(t, client.calls, 2)
	followUp := mustMarshalInput(t, client.calls[1])
	assert.NotContains(t, followUp, "rs_orphan")
	assert.Contains(t, followUp, "create_node")
}

func TestOpenAIOrchestrator_WhenAPIReturns400_ExpectProviderMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	client := &recordingResponsesClient{err: &openai.Error{
		StatusCode: 400,
		Code:       "invalid_request_error",
		Param:      "input",
		Message:    "Item 'fc_1' of type 'function_call' was provided without its required reasoning item.",
	}}
	orch := llm.NewTestOrchestrator(client, "gpt-5", nil)

	_, err := orch.Process(ctx, llm.ProcessInput{Text: "https://example.com/article"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "without its required reasoning item")
	assert.Contains(t, err.Error(), "invalid_request_error")
	assert.Contains(t, err.Error(), "param input")
	assert.Contains(t, err.Error(), "400")
}

type recordingResponsesClient struct {
	calls     []responses.ResponseNewParams
	responses []*responses.Response
	err       error
}

func (c *recordingResponsesClient) New(_ context.Context, params responses.ResponseNewParams, _ ...option.RequestOption) (*responses.Response, error) {
	c.calls = append(c.calls, params)
	if c.err != nil {
		return nil, c.err
	}
	i := len(c.calls) - 1
	if i >= len(c.responses) {
		return nil, errNoMoreResponses
	}

	return c.responses[i], nil
}

func unmarshalResponse(tb testing.TB, data string) *responses.Response {
	tb.Helper()
	var resp responses.Response
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		tb.Fatalf("unmarshal response: %v", err)
	}

	return &resp
}

func mustMarshalInput(tb testing.TB, params responses.ResponseNewParams) string {
	tb.Helper()
	body, err := json.Marshal(params.Input)
	if err != nil {
		tb.Fatalf("marshal input: %v", err)
	}

	return string(body)
}
