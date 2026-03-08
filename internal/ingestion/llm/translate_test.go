package llm_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
)

func buildMessageResponse(tb testing.TB, translatedText string) *responses.Response {
	tb.Helper()
	content := []responses.ResponseOutputMessageContentUnion{
		{Type: "output_text", Text: translatedText},
	}
	outputItem := responses.ResponseOutputItemUnion{
		Type:    "message",
		Content: content,
	}
	data := map[string]any{
		"id":                 "resp-translate",
		"created_at":         float64(0),
		"error":              map[string]any{},
		"incomplete_details": map[string]any{},
		"instructions":       "",
		"metadata":           map[string]any{},
		"model":              "gpt-4o",
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

func TestTranslateToRussian_WhenSuccess_ExpectTranslatedText(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	translated := "Это переведённый текст."
	mockClient := &mockResponsesClient{
		response: buildMessageResponse(t, translated),
	}
	orch := llm.NewTestOrchestrator(mockClient, "gpt-4o", nil)

	result, err := orch.TranslateToRussian(ctx, "This is translated text.")

	require.NoError(t, err)
	assert.Equal(t, translated, result)
	assert.Len(t, mockClient.calls, 1)
	params := mockClient.calls[0]
	assert.NotNil(t, params.Instructions)
	assert.Empty(t, params.Tools)
}
