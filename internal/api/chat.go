package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"

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
const (
	chatMinRelevantScore      = 0.08
	chatScoreGapCutoff        = 0.18
	chatMinSourcesBeforeGap   = 3
	chatMaxRelevantSourceRank = 5
)

type chatMode string

const (
	chatModeMemory chatMode = "chat_memory"
	chatModeRAG    chatMode = "rag_kb"
	chatModeHybrid chatMode = "hybrid"
)

var searchRewriteVocabularyOptions = index.SearchVocabularyOptions{
	Limit:                     150,
	MaxDocumentFrequencyRatio: 0.3,
	MinTermRunes:              3,
	MaxTermRunes:              64,
	MaxWords:                  5,
}

var chatRetrievalStopWords = map[string]struct{}{
	"about":      {},
	"base":       {},
	"documents":  {},
	"find":       {},
	"materials":  {},
	"knowledge":  {},
	"kb":         {},
	"show":       {},
	"source":     {},
	"sources":    {},
	"what":       {},
	"база":       {},
	"базе":       {},
	"базу":       {},
	"базы":       {},
	"в":          {},
	"во":         {},
	"документам": {},
	"документах": {},
	"документы":  {},
	"есть":       {},
	"знание":     {},
	"знания":     {},
	"знаний":     {},
	"из":         {},
	"источникам": {},
	"источниках": {},
	"какая":      {},
	"какие":      {},
	"какой":      {},
	"какое":      {},
	"материал":   {},
	"материалы":  {},
	"найди":      {},
	"на":         {},
	"по":         {},
	"покажи":     {},
	"про":        {},
	"расскажи":   {},
	"статей":     {},
	"статьи":     {},
	"статья":     {},
	"что":        {},
}

// ChatRequest — запрос к чатботу.
type ChatRequest struct {
	Message     string   `json:"message"`
	SessionID   string   `json:"session_id"`   //nolint:tagliatelle // REST API snake_case
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

type chatCompletionStream interface {
	Next() bool
	Current() openai.ChatCompletionChunk
	Err() error
	Close() error
}

// chatClient — интерфейс для rewrite через Responses API и ответа через Chat Completions streaming.
type chatClient interface {
	New(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) (*responses.Response, error)
	NewStreaming(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) chatStream
	NewChatStreaming(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) chatCompletionStream
}

// openaiChatClient оборачивает OpenAI-compatible APIs для rewrite и streaming-чата.
type openaiChatClient struct {
	responsesService       *responses.ResponseService
	chatCompletionsService *openai.ChatCompletionService
}

func (c *openaiChatClient) NewStreaming(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) chatStream {
	return c.responsesService.NewStreaming(ctx, body, opts...)
}

func (c *openaiChatClient) New(ctx context.Context, body responses.ResponseNewParams, opts ...option.RequestOption) (*responses.Response, error) {
	return c.responsesService.New(ctx, body, opts...)
}

func (c *openaiChatClient) NewChatStreaming(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) chatCompletionStream {
	return c.chatCompletionsService.NewStreaming(ctx, body, opts...)
}

// newOpenAIChatClient создаёт клиент для чата через OpenAI Responses API.
func newOpenAIChatClient(apiURL, apiKey string) *openaiChatClient {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if apiURL != "" {
		opts = append(opts, option.WithBaseURL(apiURL))
	}
	client := openai.NewClient(opts...)

	return &openaiChatClient{
		responsesService:       &client.Responses,
		chatCompletionsService: &client.Chat.Completions,
	}
}

// PostChat обрабатывает POST /api/chat — RAG-чатбот с SSE streaming.
func (h *Handler) PostChat(w http.ResponseWriter, r *http.Request) {
	if h.indexStore == nil || h.embeddingProvider == nil || h.chatStore == nil {
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
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")

		return
	}
	if err := h.chatStore.EnsureSession(ctx, sessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "chat not found")

			return
		}
		writeError(w, http.StatusInternalServerError, "chat store failed")

		return
	}
	if err := h.chatStore.AddMessage(ctx, sessionID, "user", req.Message, false); err != nil {
		clog.Errorf(ctx, "chat: save user message: %w", err)
		writeError(w, http.StatusInternalServerError, "chat store failed")

		return
	}
	_ = h.chatStore.SummarizeAndTrim(ctx, sessionID)
	mode := detectChatMode(req.Message)

	if mode == chatModeMemory {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache, no-transform")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, canFlush := w.(http.Flusher)
		fmt.Fprintf(w, "data: {\"sources\": []}\n\n")
		if canFlush {
			flusher.Flush()
		}

		assistantReply, err := h.streamLLMResponse(ctx, w, sessionID, req.Message, "", canFlush, flusher)
		if err != nil {
			clog.Errorf(ctx, "chat(memory): stream LLM response: %w", err)

			return
		}
		_ = h.chatStore.AddMessage(ctx, sessionID, "assistant", assistantReply, false)
		_ = h.chatStore.SummarizeAndTrim(ctx, sessionID)

		fmt.Fprint(w, "data: [DONE]\n\n")
		if canFlush {
			flusher.Flush()
		}

		return
	}

	service := index.NewRetrievalService(h.indexStore, h.embeddingProvider)
	retrievalQuery := h.chatRetrievalQuery(ctx, req.Message)
	results, err := service.Retrieve(ctx, index.RetrievalOptions{
		Query:       retrievalQuery,
		Mode:        index.RetrievalModeChat,
		Limit:       chatMaxRelevantSourceRank,
		TopK:        max(chatMaxRelevantSourceRank*2, 10),
		SourcePaths: req.SourcePaths,
	})
	if err != nil {
		clog.Errorf(ctx, "chat: search context: %w", err)
		writeError(w, http.StatusInternalServerError, "search failed")

		return
	}
	results = filterRelevantResults(retrievalQuery, results)

	sources := buildChatSources(results)
	contextText := h.buildHybridContextText(results)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, canFlush := w.(http.Flusher)

	sourcesJSON, _ := json.Marshal(sources)
	fmt.Fprintf(w, "data: {\"sources\": %s}\n\n", sourcesJSON)
	if canFlush {
		flusher.Flush()
	}

	if strings.TrimSpace(contextText) == "" && mode == chatModeRAG {
		h.writeChatToken(w, "В базе знаний не найдено релевантной информации по запросу.", canFlush, flusher)
		_ = h.chatStore.AddMessage(ctx, sessionID, "assistant", "В базе знаний не найдено релевантной информации по запросу.", false)
	} else if strings.TrimSpace(contextText) == "" {
		h.writeChatToken(w, "Недостаточно данных в базе знаний для ответа.", canFlush, flusher)
		_ = h.chatStore.AddMessage(ctx, sessionID, "assistant", "Недостаточно данных в базе знаний для ответа.", false)
	} else if assistantReply, err := h.streamLLMResponse(ctx, w, sessionID, req.Message, contextText, canFlush, flusher); err != nil {
		clog.Errorf(ctx, "chat: stream LLM response: %w", err)

		return
	} else {
		_ = h.chatStore.AddMessage(ctx, sessionID, "assistant", assistantReply, false)
		_ = h.chatStore.SummarizeAndTrim(ctx, sessionID)
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
		Model: shared.ResponsesModel(chatModel), //nolint:unconvert // shared.ResponsesModel is distinct from string
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
		piece := formatHybridNodeContextPiece(result)
		pieceTokens := estimateContextTokens(piece)
		if usedTokens+pieceTokens > ragContextTokenBudget {
			return buf.String()
		}
		buf.WriteString(piece)
		usedTokens += pieceTokens
	}

	return buf.String()
}

func formatHybridNodeContextPiece(result index.HybridSearchResult) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "--- %s ---\n", result.Path)
	if strings.TrimSpace(result.Title) != "" {
		buf.WriteString("Title: ")
		buf.WriteString(result.Title)
		buf.WriteString("\n")
	}
	if len(result.Keywords) > 0 {
		buf.WriteString("Keywords: ")
		buf.WriteString(strings.Join(result.Keywords, ", "))
		buf.WriteString("\n")
	}
	if strings.TrimSpace(result.Annotation) != "" {
		buf.WriteString("Annotation: ")
		buf.WriteString(result.Annotation)
		buf.WriteString("\n")
	}
	buf.WriteString("\n")

	return buf.String()
}

func formatHybridFragmentContextPiece(result index.HybridSearchResult, fragment index.HybridFragment) string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "--- Fragment from %s ---\n", result.Path)
	if fragment.Heading != "" {
		fmt.Fprintf(&buf, "## %s\n", fragment.Heading)
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
	fmt.Fprintf(&buf, "--- Fragment from %s ---\n", cr.NodePath)
	if cr.Heading != "" {
		fmt.Fprintf(&buf, "## %s\n", cr.Heading)
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

func (h *Handler) streamLLMResponse(ctx context.Context, w http.ResponseWriter, sessionID, message, contextText string, canFlush bool, flusher http.Flusher) (string, error) {
	chatModel := h.embeddingConfig.ChatModel
	if chatModel == "" {
		chatModel = "gpt-4o"
	}

	instructions := "You are a helpful assistant that answers questions based on the provided knowledge base context. " +
		"If the context is empty and user asks about previous conversation, answer from conversation history only. " +
		"If the context is empty or not relevant for a knowledge-base question, explicitly say that relevant information was not found in the knowledge base. " +
		"When context is present, provide the best possible answer from it and clearly note any uncertainty. " +
		"Answer in the same language as the user's question."

	promptMessages, err := h.chatStore.BuildPromptMessages(ctx, sessionID)
	if err != nil {
		return "", err
	}
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(promptMessages)+2)
	messages = append(messages, openai.SystemMessage(instructions))
	for _, m := range promptMessages {
		switch m["role"] {
		case "assistant":
			messages = append(messages, openai.AssistantMessage(m["content"]))
		case "system":
			messages = append(messages, openai.SystemMessage(m["content"]))
		default:
			messages = append(messages, openai.UserMessage(m["content"]))
		}
	}
	messages = append(messages, openai.UserMessage("Context:\n"+contextText+"\n\nQuestion: "+message))

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(chatModel), //nolint:unconvert // shared.ChatModel is distinct from string
		Messages: messages,
	}

	stream := h.chatClient.NewChatStreaming(ctx, params)
	defer stream.Close()

	var full strings.Builder
	for stream.Next() {
		chunk := stream.Current()
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			full.WriteString(choice.Delta.Content)
			tokenJSON, _ := json.Marshal(map[string]string{"token": choice.Delta.Content})
			fmt.Fprintf(w, "data: %s\n\n", tokenJSON)
			if canFlush {
				flusher.Flush()
			}
		}
	}

	if err := stream.Err(); err != nil {
		return "", err
	}

	return full.String(), nil
}

func (h *Handler) writeChatToken(w http.ResponseWriter, token string, canFlush bool, flusher http.Flusher) {
	tokenJSON, _ := json.Marshal(map[string]string{"token": token})
	fmt.Fprintf(w, "data: %s\n\n", tokenJSON)
	if canFlush {
		flusher.Flush()
	}
}

func (h *Handler) chatRetrievalQuery(ctx context.Context, message string) string {
	rewritten := h.rewriteSearchQuery(ctx, message)
	if compact := compactKnowledgeBaseQuery(rewritten); compact != "" {
		return compact
	}
	if compact := compactKnowledgeBaseQuery(message); compact != "" {
		return compact
	}

	return strings.TrimSpace(message)
}

func compactKnowledgeBaseQuery(text string) string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	terms := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		normalized := strings.ToLower(field)
		if normalized == "" {
			continue
		}
		if _, ok := chatRetrievalStopWords[normalized]; ok {
			continue
		}
		if len([]rune(normalized)) < 2 {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		terms = append(terms, field)
	}

	return strings.Join(terms, " ")
}

func detectChatMode(query string) chatMode {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return chatModeHybrid
	}
	memoryHints := []string{
		"резюме чата", "краткое резюме", "подведи итог", "подведи итоги", "что мы обсудили",
		"суммируй чат", "сделай резюме", "что ты уже сказал", "предыдущ",
	}
	for _, hint := range memoryHints {
		if strings.Contains(q, hint) {
			return chatModeMemory
		}
	}
	ragHints := []string{
		"в базе", "по базе", "из базы", "в kb", "по документам", "по источникам", "найди в базе",
	}
	for _, hint := range ragHints {
		if strings.Contains(q, hint) {
			return chatModeRAG
		}
	}

	return chatModeHybrid
}

func filterRelevantResults(query string, results []index.HybridSearchResult) []index.HybridSearchResult {
	if len(results) == 0 {
		return results
	}
	sorted := append([]index.HybridSearchResult(nil), results...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })
	queryTerms := significantTerms(query)
	filtered := make([]index.HybridSearchResult, 0, len(sorted))
	for _, r := range sorted {
		hasOverlap := len(queryTerms) == 0 || hasTermOverlap(queryTerms, r)
		if isLexicalResult(r) {
			if hasOverlap {
				filtered = append(filtered, r)
			}

			continue
		}
		if r.Score < chatMinRelevantScore || !hasOverlap {
			continue
		}
		filtered = append(filtered, r)
	}
	if len(filtered) <= 1 {
		return filtered
	}
	cutIdx := len(filtered)
	for i := chatMinSourcesBeforeGap; i < len(filtered); i++ {
		gap := filtered[i-1].Score - filtered[i].Score
		if gap >= chatScoreGapCutoff {
			cutIdx = i

			break
		}
	}

	return filtered[:cutIdx]
}

func significantTerms(text string) []string {
	compact := compactKnowledgeBaseQuery(text)
	if compact != "" {
		text = compact
	}
	text = strings.ToLower(text)
	parts := strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	terms := make([]string, 0, len(parts))
	for _, p := range parts {
		if len([]rune(p)) < 2 {
			continue
		}
		if _, ok := chatRetrievalStopWords[p]; ok {
			continue
		}
		terms = append(terms, p)
	}

	return terms
}

func isLexicalResult(result index.HybridSearchResult) bool {
	for _, kind := range result.SourceKinds {
		if kind == "exact" || kind == "keyword" || kind == "keyword_chunk" {
			return true
		}
	}
	for _, reason := range result.MatchReasons {
		if strings.Contains(reason, "exact") || strings.Contains(reason, "keyword") || strings.Contains(reason, "chunk:") {
			return true
		}
	}

	return false
}

func hasTermOverlap(queryTerms []string, result index.HybridSearchResult) bool {
	bag := strings.ToLower(result.Path + " " + result.Title + " " + result.Annotation + " " + strings.Join(result.Keywords, " "))
	var bagSb947 strings.Builder
	for _, f := range result.Fragments {
		bagSb947.WriteString(" " + strings.ToLower(f.Heading) + " " + strings.ToLower(f.Snippet) + " " + strings.ToLower(f.Content))
	}
	bag += bagSb947.String()
	for _, t := range queryTerms {
		if strings.Contains(bag, t) {
			return true
		}
	}

	return false
}
