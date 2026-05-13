package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strider2038/knowledge-db/internal/index"
	"github.com/strider2038/knowledge-db/internal/kb"
)

var errSemanticSearchUnavailable = errors.New("semantic_search unavailable: embeddings are disabled, use search_notes")

type searchNotesInput struct {
	Query           string   `json:"query" jsonschema:"Search query text for notes retrieval"`
	Limit           int      `json:"limit,omitempty" jsonschema:"Maximum number of results to return, from 1 to 50"`
	Path            string   `json:"path,omitempty" jsonschema:"Filter by node path or subtree root"`
	Recursive       bool     `json:"recursive,omitempty" jsonschema:"When true, search recursively under the provided path"`
	Types           []string `json:"types,omitempty" jsonschema:"Optional node type filters, for example note, article, link"`
	ManualProcessed *bool    `json:"manual_processed,omitempty" jsonschema:"Optional filter by manual_processed flag"`
}

type semanticSearchInput struct {
	Query string `json:"query" jsonschema:"Semantic search query text"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of results to return, from 1 to 50"`
}

type getNoteInput struct {
	Path           string `json:"path" jsonschema:"Knowledge-base node path, for example topic/node"`
	IncludeContent *bool  `json:"include_content,omitempty" jsonschema:"Whether to include full note content in response, default true"`
	MaxChars       int    `json:"max_chars,omitempty" jsonschema:"Optional max content length in characters"`
}

type toolResult struct {
	Results []toolResultItem `json:"results"`
	Total   int              `json:"total"`
}

type toolResultItem struct {
	Path      string   `json:"path"`
	Title     string   `json:"title"`
	Type      string   `json:"type,omitempty"`
	Score     float64  `json:"score"`
	Snippets  []string `json:"snippets,omitempty"`
	SourceURL string   `json:"source_url,omitempty"`
}

type getNoteResult struct {
	Path       string   `json:"path"`
	Title      string   `json:"title"`
	Type       string   `json:"type,omitempty"`
	Annotation string   `json:"annotation,omitempty"`
	SourceURL  string   `json:"source_url,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	Content    string   `json:"content"`
	Truncated  bool     `json:"truncated"`
}

// Handler serves MCP endpoint /api/mcp with bearer auth.
type Handler struct {
	apiKey   string
	handler  http.Handler
	services *searchServices
}

type searchServices struct {
	indexStore *index.IndexStore
	provider   index.EmbeddingProvider
}

// NewHandler creates MCP handler with Bearer auth and search tools.
func NewHandler(apiKey string, indexStore *index.IndexStore, provider index.EmbeddingProvider) http.Handler {
	services := &searchServices{
		indexStore: indexStore,
		provider:   provider,
	}
	server := newServer(services)

	transport := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		return server
	}, &sdkmcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	return &Handler{
		apiKey:   strings.TrimSpace(apiKey),
		handler:  transport,
		services: services,
	}
}

// ServeHTTP serves MCP endpoint with Bearer auth.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok || token != h.apiKey {
		w.Header().Set("WWW-Authenticate", `Bearer realm="kb-mcp"`)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	h.handler.ServeHTTP(w, r)
}

func newServer(services *searchServices) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "knowledge-db-mcp",
		Version: "1.0.0",
	}, nil)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "search_notes",
		Description: "Search notes by keyword/hybrid retrieval",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input searchNotesInput) (*sdkmcp.CallToolResult, toolResult, error) {
		result, err := services.searchNotes(ctx, input)
		if err != nil {
			return nil, toolResult{}, err
		}

		return &sdkmcp.CallToolResult{}, result, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "semantic_search",
		Description: "Search notes with semantic retrieval",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input semanticSearchInput) (*sdkmcp.CallToolResult, toolResult, error) {
		result, err := services.semanticSearch(ctx, input)
		if err != nil {
			return nil, toolResult{}, err
		}

		return &sdkmcp.CallToolResult{}, result, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_note",
		Description: "Read a note/article by path from knowledge base",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, input getNoteInput) (*sdkmcp.CallToolResult, getNoteResult, error) {
		result, err := services.getNote(ctx, input)
		if err != nil {
			return nil, getNoteResult{}, err
		}

		return &sdkmcp.CallToolResult{}, result, nil
	})

	return server
}

func (s *searchServices) searchNotes(ctx context.Context, input searchNotesInput) (toolResult, error) {
	if s.indexStore == nil {
		return toolResult{}, errors.New("search unavailable: index is not initialized")
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return toolResult{}, errors.New("query is required")
	}
	limit := normalizeLimit(input.Limit)

	retrieval := index.NewRetrievalService(s.indexStore, s.provider)
	results, err := retrieval.Retrieve(ctx, index.RetrievalOptions{
		Query:           query,
		Mode:            index.RetrievalModeSearch,
		Types:           input.Types,
		Path:            input.Path,
		Recursive:       input.Recursive,
		ManualProcessed: input.ManualProcessed,
		Limit:           limit,
		TopK:            max(limit, 10),
	})
	if err != nil {
		return toolResult{}, fmt.Errorf("search_notes: %w", err)
	}

	return mapHybridResults(results), nil
}

func (s *searchServices) semanticSearch(ctx context.Context, input semanticSearchInput) (toolResult, error) {
	if s.indexStore == nil {
		return toolResult{}, errors.New("search unavailable: index is not initialized")
	}
	if s.provider == nil {
		return toolResult{}, errSemanticSearchUnavailable
	}
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return toolResult{}, errors.New("query is required")
	}
	limit := normalizeLimit(input.Limit)

	retrieval := index.NewRetrievalService(s.indexStore, s.provider)
	results, err := retrieval.Retrieve(ctx, index.RetrievalOptions{
		Query: query,
		Mode:  index.RetrievalModeSearch,
		Limit: limit,
		TopK:  max(limit, 10),
	})
	if err != nil {
		return toolResult{}, fmt.Errorf("semantic_search: %w", err)
	}

	return mapHybridResults(results), nil
}

func (s *searchServices) getNote(ctx context.Context, input getNoteInput) (getNoteResult, error) {
	dataPath := s.dataPath()
	if dataPath == "" {
		return getNoteResult{}, errors.New("get_note unavailable: index is not initialized")
	}
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return getNoteResult{}, errors.New("path is required")
	}

	node, err := kb.GetNode(ctx, dataPath, path)
	if err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			return getNoteResult{}, fmt.Errorf("node not found: %s", path)
		}

		return getNoteResult{}, fmt.Errorf("get_note: %w", err)
	}

	includeContent := input.IncludeContent == nil || *input.IncludeContent
	content := ""
	truncated := false
	if includeContent {
		content, truncated = truncateContent(node.Content, input.MaxChars)
	}
	result := getNoteResult{
		Path:       node.Path,
		Title:      node.Path,
		Annotation: node.Annotation,
		Content:    content,
		Truncated:  truncated,
	}
	if title, ok := node.Metadata["title"].(string); ok && strings.TrimSpace(title) != "" {
		result.Title = title
	}
	if nodeType, ok := node.Metadata["type"].(string); ok {
		result.Type = nodeType
	}
	if sourceURL, ok := node.Metadata["source_url"].(string); ok {
		result.SourceURL = sourceURL
	}
	if keywords, ok := node.Metadata["keywords"].([]any); ok {
		result.Keywords = normalizeKeywords(keywords)
	}
	if keywords, ok := node.Metadata["keywords"].([]string); ok {
		result.Keywords = append(result.Keywords[:0], keywords...)
	}

	return result, nil
}

func mapHybridResults(results []index.HybridSearchResult) toolResult {
	out := toolResult{
		Results: make([]toolResultItem, 0, len(results)),
		Total:   len(results),
	}
	for _, result := range results {
		item := toolResultItem{
			Path:      result.Path,
			Title:     result.Title,
			Type:      result.Type,
			Score:     result.Score,
			SourceURL: result.SourceURL,
		}
		for _, fragment := range result.Fragments {
			snippet := strings.TrimSpace(fragment.Snippet)
			if snippet != "" {
				item.Snippets = append(item.Snippets, snippet)
			}
		}
		out.Results = append(out.Results, item)
	}

	return out
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}

	return limit
}

func truncateContent(content string, maxChars int) (string, bool) {
	if maxChars <= 0 {
		return content, false
	}
	runes := []rune(content)
	if len(runes) <= maxChars {
		return content, false
	}

	return string(runes[:maxChars]), true
}

func normalizeKeywords(values []any) []string {
	keywords := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		keywords = append(keywords, text)
	}

	return keywords
}

func (s *searchServices) dataPath() string {
	if s.indexStore == nil {
		return ""
	}

	return strings.TrimSpace(s.indexStore.DataPath())
}

func bearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}

	return token, true
}
