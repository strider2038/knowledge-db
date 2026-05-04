package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
const searchRewriteTimeout = 2 * time.Second

var searchRewriteVocabularyOptions = index.SearchVocabularyOptions{
	Limit:                     150,
	MaxDocumentFrequencyRatio: 0.3,
	MinTermRunes:              3,
	MaxTermRunes:              64,
	MaxWords:                  5,
}

// ChatRequest — запрос к чатботу.
type ChatRequest struct {
	Message     string   `json:"message"`
	SourcePaths []string `json:"source_paths"` //nolint:tagliatelle // REST API snake_case
}

// ChatSource — источник ответа чатбота.
type ChatSource struct {
	Path      string           `json:"path"`
	Title     string           `json:"title"`
	Type      string           `json:"type,omitempty"`
	Fragments []SearchFragment `json:"fragments,omitempty"`
}

// SearchRequest — запрос гибридного поиска.
type SearchRequest struct {
	Query           string   `json:"query"`
	Types           []string `json:"type"`
	Path            string   `json:"path"`
	Recursive       bool     `json:"recursive"`
	ManualProcessed *bool    `json:"manual_processed"` //nolint:tagliatelle // REST API snake_case
	Limit           int      `json:"limit"`
	Offset          int      `json:"offset"`
	Mode            string   `json:"mode"`
	SourcePaths     []string `json:"source_paths"` //nolint:tagliatelle // REST API snake_case
}

// SearchResponse — ответ гибридного поиска.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Query   string         `json:"query"`
	Mode    string         `json:"mode"`
	Meta    SearchMeta     `json:"meta"`
}

// SearchMeta — metadata retrieval ответа.
type SearchMeta struct {
	KeywordIndex string `json:"keyword_index"`           //nolint:tagliatelle // REST API snake_case
	QueryRewrite string `json:"query_rewrite,omitempty"` //nolint:tagliatelle // REST API snake_case
}

// SearchResult — карточка результата гибридного поиска.
type SearchResult struct {
	Path         string           `json:"path"`
	Title        string           `json:"title"`
	Type         string           `json:"type"`
	Annotation   string           `json:"annotation"`
	Keywords     []string         `json:"keywords"`
	SourceURL    string           `json:"source_url,omitempty"` //nolint:tagliatelle // REST API snake_case
	Score        float64          `json:"score"`
	Rank         int              `json:"rank"`
	MatchReasons []string         `json:"match_reasons"` //nolint:tagliatelle // REST API snake_case
	SourceKinds  []string         `json:"source_kinds"`  //nolint:tagliatelle // REST API snake_case
	Fragments    []SearchFragment `json:"fragments,omitempty"`
}

// SearchFragment — найденный фрагмент статьи/заметки.
type SearchFragment struct {
	Heading   string  `json:"heading,omitempty"`
	Snippet   string  `json:"snippet,omitempty"`
	Content   string  `json:"content,omitempty"`
	Score     float64 `json:"score"`
	MatchType string  `json:"match_type"` //nolint:tagliatelle // REST API snake_case
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
	New(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) (*responses.Response, error)
	NewStreaming(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) chatStream
}

// openaiChatClient оборачивает responses.ResponseService для соответствия chatResponsesClient.
type openaiChatClient struct {
	service *responses.ResponseService
}

func (c *openaiChatClient) NewStreaming(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) chatStream {
	return c.service.NewStreaming(ctx, body, opts...)
}

func (c *openaiChatClient) New(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) (*responses.Response, error) {
	return c.service.New(ctx, body, opts...)
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

	service := index.NewRetrievalService(h.indexStore, h.embeddingProvider)
	results, err := service.Retrieve(ctx, index.RetrievalOptions{
		Query:       req.Message,
		Mode:        index.RetrievalModeChat,
		Limit:       5,
		TopK:        10,
		SourcePaths: req.SourcePaths,
	})
	if err != nil {
		clog.Errorf(ctx, "chat: search context: %w", err)
		writeError(w, http.StatusInternalServerError, "search failed")

		return
	}

	sources := buildChatSources(results)
	contextText := h.buildHybridContextText(results)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, canFlush := w.(http.Flusher)

	sourcesJSON, _ := json.Marshal(sources)
	fmt.Fprintf(w, "data: {\"sources\": %s}\n\n", sourcesJSON)
	if canFlush {
		flusher.Flush()
	}

	if strings.TrimSpace(contextText) == "" {
		h.writeChatToken(w, "Недостаточно данных в базе знаний для ответа.", canFlush, flusher)
	} else if err := h.streamLLMResponse(ctx, w, req.Message, contextText, canFlush, flusher); err != nil {
		clog.Errorf(ctx, "chat: stream LLM response: %w", err)

		return
	}

	fmt.Fprint(w, "data: [DONE]\n\n")
	if canFlush {
		flusher.Flush()
	}
}

// PostSearch обрабатывает POST /api/search — гибридный поиск по индексу.
func (h *Handler) PostSearch(w http.ResponseWriter, r *http.Request) {
	if h.indexStore == nil || h.embeddingProvider == nil {
		writeError(w, http.StatusServiceUnavailable, "embedding service unavailable")

		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		writeError(w, http.StatusBadRequest, "query is required")

		return
	}

	mode := index.RetrievalModeSearch
	if req.Mode == string(index.RetrievalModeChat) {
		mode = index.RetrievalModeChat
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	retrievalQuery := h.rewriteSearchQuery(r.Context(), query)
	service := index.NewRetrievalService(h.indexStore, h.embeddingProvider)
	results, err := service.Retrieve(r.Context(), index.RetrievalOptions{
		Query:           retrievalQuery,
		Mode:            mode,
		Types:           req.Types,
		Path:            req.Path,
		Recursive:       req.Recursive,
		ManualProcessed: req.ManualProcessed,
		Limit:           limit + max(req.Offset, 0),
		TopK:            max(limit+max(req.Offset, 0), 10),
		SourcePaths:     req.SourcePaths,
	})
	if err != nil {
		clog.Errorf(r.Context(), "search: retrieve: %w", err)
		writeError(w, http.StatusInternalServerError, "search failed")

		return
	}

	total := len(results)
	offset := min(max(req.Offset, 0), total)
	end := min(offset+limit, total)
	writeJSON(w, SearchResponse{
		Results: mapSearchResults(results[offset:end]),
		Total:   total,
		Query:   query,
		Mode:    string(mode),
		Meta:    SearchMeta{KeywordIndex: h.indexStore.KeywordIndexMode(), QueryRewrite: queryRewriteMeta(query, retrievalQuery)},
	})
}

func (h *Handler) rewriteSearchQuery(ctx context.Context, query string) string {
	if !h.embeddingConfig.SearchRewriteEnabled || h.chatClient == nil {
		return query
	}

	chatModel := h.embeddingConfig.ChatModel
	if chatModel == "" {
		chatModel = "gpt-4o"
	}
	var vocabulary []string
	if h.indexStore != nil {
		var err error
		vocabulary, err = h.indexStore.SearchVocabulary(ctx, searchRewriteVocabularyOptions)
		if err != nil {
			clog.Debug(ctx, "search: vocabulary failed", "error", err.Error())
		}
	}

	rewriteCtx, cancel := context.WithTimeout(ctx, searchRewriteTimeout)
	defer cancel()

	params := responses.ResponseNewParams{
		Model: shared.ResponsesModel(chatModel),
		Instructions: openai.String(
			"Rewrite the user's knowledge-base search question into a compact search query. " +
				"Keep important exact terms, product names, acronyms, and domain terms. " +
				"Add common synonyms in Russian and English when useful. " +
				"Remove filler/question words. Return only the rewritten query as plain text. " +
				"Prefer vocabulary terms only when they are relevant to the user's intent.\n\n" +
				"Knowledge base vocabulary:\n" + strings.Join(vocabulary, ", "),
		),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(query, responses.EasyInputMessageRoleUser),
			},
		},
	}

	resp, err := h.chatClient.New(rewriteCtx, params)
	if err != nil {
		clog.Debug(ctx, "search: query rewrite failed", "error", err.Error())

		return query
	}
	rewritten := sanitizeSearchRewrite(resp.OutputText())
	if rewritten == "" {
		return query
	}

	return rewritten
}

func sanitizeSearchRewrite(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`\"' \n\t")
	value = strings.TrimPrefix(value, "query:")
	value = strings.TrimPrefix(value, "Query:")
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > 240 || strings.Contains(value, "\n\n") {
		return ""
	}

	return strings.Join(strings.Fields(value), " ")
}

func queryRewriteMeta(original, rewritten string) string {
	if rewritten == "" || rewritten == original {
		return ""
	}

	return rewritten
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
		"keyword_index":   status.KeywordIndex,
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

func (h *Handler) buildHybridContextText(results []index.HybridSearchResult) string {
	var buf strings.Builder
	usedTokens := 0
	for _, result := range results {
		for _, fragment := range result.Fragments {
			piece := formatHybridFragmentContextPiece(result, fragment)
			pieceTokens := estimateContextTokens(piece)
			if usedTokens+pieceTokens > ragContextTokenBudget {
				return buf.String()
			}
			buf.WriteString(piece)
			usedTokens += pieceTokens
		}
		if len(result.Fragments) > 0 {
			continue
		}
		piece := fmt.Sprintf("--- %s ---\n%s\n\n", result.Path, result.Annotation)
		pieceTokens := estimateContextTokens(piece)
		if usedTokens+pieceTokens > ragContextTokenBudget {
			return buf.String()
		}
		buf.WriteString(piece)
		usedTokens += pieceTokens
	}

	return buf.String()
}

func formatHybridFragmentContextPiece(result index.HybridSearchResult, fragment index.HybridFragment) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("--- Fragment from %s ---\n", result.Path))
	if fragment.Heading != "" {
		buf.WriteString(fmt.Sprintf("## %s\n", fragment.Heading))
	}
	if fragment.Snippet != "" {
		buf.WriteString("Snippet: ")
		buf.WriteString(fragment.Snippet)
		buf.WriteString("\n")
	}
	if fragment.Content != "" {
		buf.WriteString(fragment.Content)
	}
	buf.WriteString("\n\n")

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

func buildChatSources(results []index.HybridSearchResult) []ChatSource {
	sources := make([]ChatSource, 0, len(results))
	for _, result := range results {
		sources = append(sources, ChatSource{
			Path:      result.Path,
			Title:     result.Title,
			Type:      result.Type,
			Fragments: mapSearchFragments(result.Fragments),
		})
	}

	return sources
}

func mapSearchResults(results []index.HybridSearchResult) []SearchResult {
	mapped := make([]SearchResult, len(results))
	for i, result := range results {
		mapped[i] = SearchResult{
			Path:         result.Path,
			Title:        result.Title,
			Type:         result.Type,
			Annotation:   result.Annotation,
			Keywords:     result.Keywords,
			SourceURL:    result.SourceURL,
			Score:        result.Score,
			Rank:         result.Rank,
			MatchReasons: result.MatchReasons,
			SourceKinds:  result.SourceKinds,
			Fragments:    mapSearchFragments(result.Fragments),
		}
	}

	return mapped
}

func mapSearchFragments(fragments []index.HybridFragment) []SearchFragment {
	mapped := make([]SearchFragment, len(fragments))
	for i, fragment := range fragments {
		mapped[i] = SearchFragment{
			Heading:   fragment.Heading,
			Snippet:   fragment.Snippet,
			Content:   fragment.Content,
			Score:     fragment.Score,
			MatchType: fragment.MatchType,
		}
	}

	return mapped
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

func (h *Handler) writeChatToken(w http.ResponseWriter, token string, canFlush bool, flusher http.Flusher) {
	tokenJSON, _ := json.Marshal(map[string]string{"token": token})
	fmt.Fprintf(w, "data: %s\n\n", tokenJSON)
	if canFlush {
		flusher.Flush()
	}
}
