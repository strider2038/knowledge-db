package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"

	"github.com/strider2038/knowledge-db/internal/index"
	"github.com/strider2038/knowledge-db/internal/kb"
)

const ragContextTokenBudget = 4000

// ChatRequest — запрос к чатботу.
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatSource — источник ответа чатбота.
type ChatSource struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

// chatStream — интерфейс для чтения SSE-потока от Responses API.
type chatStream interface {
	Next() bool
	Current() responses.ResponseStreamEventUnion
	Err() error
	Close() error
}

// chatResponsesClient — интерфейс для OpenAI Responses API с streaming.
type chatResponsesClient interface {
	NewStreaming(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) chatStream
}

// openaiChatClient оборачивает responses.ResponseService для соответствия chatResponsesClient.
type openaiChatClient struct {
	service *responses.ResponseService
}

func (c *openaiChatClient) NewStreaming(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) chatStream {
	return c.service.NewStreaming(ctx, body, opts...)
}

// newOpenAIChatClient создаёт клиент для чата через OpenAI Responses API.
func newOpenAIChatClient(apiURL, apiKey string) *openaiChatClient {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if apiURL != "" {
		opts = append(opts, option.WithBaseURL(apiURL))
	}
	client := openai.NewClient(opts...)

	return &openaiChatClient{service: &client.Responses}
}

// PostChat обрабатывает POST /api/chat — RAG-чатбот с SSE streaming.
func (h *Handler) PostChat(w http.ResponseWriter, r *http.Request) {
	if h.indexStore == nil || h.embeddingProvider == nil {
		writeError(w, http.StatusServiceUnavailable, "embedding service unavailable")

		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "message is required")

		return
	}

	ctx := r.Context()

	nodeResults, chunkResults, err := h.searchContext(ctx, req.Message)
	if err != nil {
		clog.Errorf(ctx, "chat: search context: %w", err)
		writeError(w, http.StatusInternalServerError, "search failed")

		return
	}

	sources := h.buildSources(nodeResults, chunkResults)
	contextText := h.buildContextText(nodeResults, chunkResults)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, canFlush := w.(http.Flusher)

	sourcesJSON, _ := json.Marshal(sources)
	fmt.Fprintf(w, "data: {\"sources\": %s}\n\n", sourcesJSON)
	if canFlush {
		flusher.Flush()
	}

	if err := h.streamLLMResponse(ctx, w, req.Message, contextText, canFlush, flusher); err != nil {
		clog.Errorf(ctx, "chat: stream LLM response: %w", err)

		return
	}

	fmt.Fprint(w, "data: [DONE]\n\n")
	if canFlush {
		flusher.Flush()
	}
}

// PostIndexRebuild обрабатывает POST /api/index/rebuild — запуск перестройки индекса.
func (h *Handler) PostIndexRebuild(w http.ResponseWriter, r *http.Request) {
	if h.indexStore == nil || h.syncWorker == nil {
		writeError(w, http.StatusServiceUnavailable, "embedding service unavailable")

		return
	}

	h.syncWorker.Send(index.ManualRebuildEvent{})
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{"status": "rebuild started"})
}

// GetIndexStatus обрабатывает GET /api/index/status — состояние индекса.
func (h *Handler) GetIndexStatus(w http.ResponseWriter, r *http.Request) {
	if h.indexStore == nil {
		writeError(w, http.StatusServiceUnavailable, "embedding service unavailable")

		return
	}

	status, err := h.indexStore.GetStatus(r.Context(), h.embeddingConfig.Model)
	if err != nil {
		clog.Errorf(r.Context(), "index status: %w", err)
		writeError(w, http.StatusInternalServerError, "failed to get index status")

		return
	}

	writeJSON(w, map[string]any{
		"total_nodes":     status.TotalNodes,
		"total_chunks":    status.TotalChunks,
		"embedding_model": status.EmbeddingModel,
		"last_indexed_at": status.LastIndexedAt,
		"status":          status.Status,
	})
}

func (h *Handler) searchContext(ctx context.Context, query string) ([]index.SearchResult, []index.ChunkSearchResult, error) {
	nodeResults, err := index.VectorSearch(ctx, h.indexStore, h.embeddingProvider, query, 5)
	if err != nil {
		return nil, nil, errors.Errorf("vector search: %w", err)
	}

	chunkResults, err := index.ChunkSearch(ctx, h.indexStore, h.embeddingProvider, query, 5)
	if err != nil {
		return nil, nil, errors.Errorf("chunk search: %w", err)
	}

	return nodeResults, chunkResults, nil
}

func (h *Handler) buildSources(nodeResults []index.SearchResult, chunkResults []index.ChunkSearchResult) []ChatSource {
	seen := make(map[string]bool)
	var sources []ChatSource

	for _, r := range nodeResults {
		if !seen[r.Path] {
			seen[r.Path] = true
			sources = append(sources, ChatSource{Path: r.Path, Title: r.Title})
		}
	}
	for _, r := range chunkResults {
		if !seen[r.NodePath] {
			seen[r.NodePath] = true
			title := r.NodePath
			node, err := h.getNodeForContext(r.NodePath)
			if err == nil {
				if metaTitle, ok := node.Metadata["title"].(string); ok && strings.TrimSpace(metaTitle) != "" {
					title = metaTitle
				}
			}
			sources = append(sources, ChatSource{Path: r.NodePath, Title: title})
		}
	}

	return sources
}

func (h *Handler) buildContextText(nodeResults []index.SearchResult, chunkResults []index.ChunkSearchResult) string {
	var buf strings.Builder
	usedTokens := 0

	coveredNodes := make(map[string]bool)
	for _, cr := range chunkResults {
		piece := formatChunkContextPiece(cr)
		pieceTokens := estimateContextTokens(piece)
		if usedTokens+pieceTokens > ragContextTokenBudget {
			break
		}

		coveredNodes[cr.NodePath] = true
		buf.WriteString(piece)
		usedTokens += pieceTokens
	}

	for _, nr := range nodeResults {
		if usedTokens >= ragContextTokenBudget {
			break
		}
		if coveredNodes[nr.Path] {
			continue
		}
		node, err := h.getNodeForContext(nr.Path)
		if err != nil {
			continue
		}
		annotation, _ := node.Metadata["annotation"].(string)
		piece := fmt.Sprintf("--- %s ---\n%s\n\n", nr.Path, annotation)
		pieceTokens := estimateContextTokens(piece)
		if usedTokens+pieceTokens > ragContextTokenBudget {
			break
		}
		buf.WriteString(piece)
		usedTokens += pieceTokens
	}

	return buf.String()
}

func formatChunkContextPiece(cr index.ChunkSearchResult) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("--- Fragment from %s ---\n", cr.NodePath))
	if cr.Heading != "" {
		buf.WriteString(fmt.Sprintf("## %s\n", cr.Heading))
	}
	buf.WriteString(cr.Content)
	buf.WriteString("\n\n")

	return buf.String()
}

func estimateContextTokens(text string) int {
	words := len(strings.Fields(text))

	return int(float64(words) * 1.3)
}

func (h *Handler) getNodeForContext(path string) (*kb.Node, error) {
	return kb.GetNode(context.Background(), h.dataPath, path)
}

func (h *Handler) streamLLMResponse(ctx context.Context, w http.ResponseWriter, message, contextText string, canFlush bool, flusher http.Flusher) error {
	chatModel := h.embeddingConfig.ChatModel
	if chatModel == "" {
		chatModel = "gpt-4o"
	}

	instructions := "You are a helpful assistant that answers questions based on the provided knowledge base context. " +
		"If the context is empty or doesn't contain relevant information, say that you couldn't find relevant information in the knowledge base. " +
		"Answer in the same language as the user's question."

	params := responses.ResponseNewParams{
		Model:        shared.ResponsesModel(chatModel),
		Instructions: openai.String(instructions),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(
					"Context:\n"+contextText+"\n\nQuestion: "+message,
					responses.EasyInputMessageRoleUser,
				),
			},
		},
	}

	stream := h.chatClient.NewStreaming(ctx, params)
	defer stream.Close()

	for stream.Next() {
		event := stream.Current()
		switch event.Type {
		case "response.output_text.delta":
			delta := event.AsResponseOutputTextDelta()
			if delta.Delta == "" {
				continue
			}
			tokenJSON, _ := json.Marshal(map[string]string{"token": delta.Delta})
			fmt.Fprintf(w, "data: %s\n\n", tokenJSON)
			if canFlush {
				flusher.Flush()
			}
		case "error":
			return errors.Errorf("stream error: %s", event.AsError().Message)
		}
	}

	return stream.Err()
}
