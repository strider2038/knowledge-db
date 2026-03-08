package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
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
}

func (m *mockContentFetcher) Fetch(_ context.Context, _ string) (*fetcher.FetchResult, error) {
	return m.result, m.err
}

func buildFunctionCallResponse(id, callID, name, arguments string) *responses.Response {
	data := `{"id":"` + id + `","created_at":0,"error":{},"incomplete_details":{},"instructions":"","metadata":{},"model":"gpt-4o","object":"response","parallel_tool_calls":false,"temperature":1,"output":[{"type":"function_call","id":"` + callID + `","call_id":"` + callID + `","name":"` + name + `","arguments":` + jsonString(arguments) + `,"status":"completed"}],"usage":{},"status":"completed","tool_choice":"auto"}`
	var resp responses.Response
	_ = json.Unmarshal([]byte(data), &resp)

	return &resp
}

func buildCreateNodeResponse(id, callID string, args map[string]any) *responses.Response {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		panic("marshal args: " + err.Error())
	}

	return buildFunctionCallResponse(id, callID, "create_node", string(argsJSON))
}

func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic("marshal string: " + err.Error())
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
		response: buildCreateNodeResponse("resp1", "call1", createNodeArgs),
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", nil)

	result, err := orch.Process(ctx, llm.ProcessInput{
		Text:             "Some notes about Go patterns.",
		ExistingThemes:   []string{"go/patterns"},
		ExistingKeywords: []string{"go", "patterns"},
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
	firstResp := buildFunctionCallResponse("resp1", "call1", "fetch_url_content", fetchArgs)

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
	secondResp := buildCreateNodeResponse("resp2", "call2", createNodeArgs)

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

func TestOpenAIOrchestrator_WhenLinkInput_ExpectFetchMetaThenCreateNode(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fetchMetaArgs := `{"url":"https://pkg.go.dev/net/http"}`
	firstResp := buildFunctionCallResponse("resp1", "call1", "fetch_url_meta", fetchMetaArgs)

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
	secondResp := buildCreateNodeResponse("resp2", "call2", createNodeArgs)

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
