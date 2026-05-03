package index

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"github.com/muonsoft/errors"
	"github.com/spf13/afero"

	"github.com/strider2038/knowledge-db/internal/kb"
)

// RetrievalMode defines retrieval tuning for search or chat.
type RetrievalMode string

const (
	RetrievalModeSearch RetrievalMode = "search"
	RetrievalModeChat   RetrievalMode = "chat"
)

// RetrievalOptions controls hybrid retrieval.
type RetrievalOptions struct {
	Query           string
	Mode            RetrievalMode
	Types           []string
	Path            string
	Recursive       bool
	ManualProcessed *bool
	Limit           int
	TopK            int
	SourcePaths     []string
}

// HybridSearchResult is a ranked node card returned by retrieval.
type HybridSearchResult struct {
	Path         string
	Title        string
	Type         string
	Annotation   string
	Keywords     []string
	SourceURL    string
	Score        float64
	Rank         int
	MatchReasons []string
	SourceKinds  []string
	Fragments    []HybridFragment
}

// HybridFragment is a relevant article or note fragment.
type HybridFragment struct {
	Heading   string
	Snippet   string
	Content   string
	Score     float64
	MatchType string
}

// RetrievalService builds unified hybrid search results for search and chat.
type RetrievalService struct {
	store    *IndexStore
	provider EmbeddingProvider
}

// NewRetrievalService creates a retrieval service.
func NewRetrievalService(store *IndexStore, provider EmbeddingProvider) *RetrievalService {
	return &RetrievalService{store: store, provider: provider}
}

// Retrieve runs keyword, vector node and vector chunk retrieval, then fuses candidates.
func (s *RetrievalService) Retrieve(ctx context.Context, opts RetrievalOptions) ([]HybridSearchResult, error) {
	opts = normalizeRetrievalOptions(opts)
	if opts.Query == "" {
		return nil, nil
	}
	acc := make(map[string]*hybridAccumulator)

	if err := s.addKeywordCandidates(ctx, opts, acc); err != nil {
		return nil, err
	}
	if s.provider != nil {
		if err := s.addVectorCandidates(ctx, opts, acc); err != nil {
			return nil, err
		}
	}

	results := make([]HybridSearchResult, 0, len(acc))
	for _, item := range acc {
		item.result.MatchReasons = sortedKeys(item.matchReasons)
		item.result.SourceKinds = sortedKeys(item.sourceKinds)
		item.result.Score = item.score
		if opts.Mode == RetrievalModeChat && item.isWeakVectorOnly() {
			continue
		}
		if !matchesRetrievalFilters(item.result, opts, item.manualProcessed) {
			continue
		}
		results = append(results, item.result)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Path < results[j].Path
		}

		return results[i].Score > results[j].Score
	})
	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}
	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}

func (s *RetrievalService) addKeywordCandidates(ctx context.Context, opts RetrievalOptions, acc map[string]*hybridAccumulator) error {
	nodeHits, chunkHits, err := KeywordSearch(ctx, s.store, opts.Query, opts.TopK)
	if err != nil {
		return errors.Errorf("keyword search: %w", err)
	}

	for _, hit := range nodeHits {
		item := ensureHybridAccumulator(acc, hit.Path)
		item.applyKeywordNode(hit)
	}
	for _, hit := range chunkHits {
		item := ensureHybridAccumulator(acc, hit.NodePath)
		doc, err := s.loadNodeSearchDocument(ctx, hit.NodePath)
		if err == nil {
			item.applyDocument(doc)
		}
		item.addSource("keyword_chunk", hit.Rank, 1.6, hit.RawScore)
		item.matchReasons["chunk:"+strings.Join(hit.MatchFields, ",")] = struct{}{}
		item.result.Fragments = append(item.result.Fragments, HybridFragment{
			Heading:   hit.Heading,
			Snippet:   hit.Snippet,
			Content:   hit.Content,
			Score:     hit.RawScore,
			MatchType: "keyword",
		})
	}

	return nil
}

func (s *RetrievalService) addVectorCandidates(ctx context.Context, opts RetrievalOptions, acc map[string]*hybridAccumulator) error {
	nodeResults, err := VectorSearch(ctx, s.store, s.provider, opts.Query, opts.TopK)
	if err != nil {
		return errors.Errorf("vector node search: %w", err)
	}
	for rank, result := range nodeResults {
		item := ensureHybridAccumulator(acc, result.Path)
		doc, err := s.loadNodeSearchDocument(ctx, result.Path)
		if err == nil {
			item.applyDocument(doc)
		} else {
			item.result.Path = result.Path
			item.result.Title = result.Title
			item.result.Annotation = result.Annotation
		}
		item.addSource("vector_node", rank+1, 1.0, float64(result.Score))
		item.maxVectorScore = max(item.maxVectorScore, float64(result.Score))
		item.matchReasons["vector"] = struct{}{}
	}

	chunkResults, err := ChunkSearch(ctx, s.store, s.provider, opts.Query, opts.TopK)
	if err != nil {
		return errors.Errorf("vector chunk search: %w", err)
	}
	for rank, result := range chunkResults {
		item := ensureHybridAccumulator(acc, result.NodePath)
		doc, err := s.loadNodeSearchDocument(ctx, result.NodePath)
		if err == nil {
			item.applyDocument(doc)
		}
		item.addSource("vector_chunk", rank+1, 1.0, float64(result.Score))
		item.maxVectorScore = max(item.maxVectorScore, float64(result.Score))
		item.matchReasons["vector"] = struct{}{}
		item.result.Fragments = append(item.result.Fragments, HybridFragment{
			Heading:   result.Heading,
			Snippet:   buildSnippet(result.Content, queryTokens(opts.Query)),
			Content:   result.Content,
			Score:     float64(result.Score),
			MatchType: "vector",
		})
	}

	return nil
}

func (s *RetrievalService) loadNodeSearchDocument(ctx context.Context, path string) (NodeSearchDocument, error) {
	var doc NodeSearchDocument
	var aliases, keywords string
	var manualProcessed int
	err := s.store.queryRowContext(ctx, `
		SELECT path, title, type, aliases, annotation, keywords, source_url, manual_processed, body
		FROM node_search WHERE path = ?`, path,
	).Scan(&doc.Path, &doc.Title, &doc.Type, &aliases, &doc.Annotation, &keywords, &doc.SourceURL, &manualProcessed, &doc.Body)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return doc, err
		}

		return s.loadNodeSearchDocumentFromFS(ctx, path, err)
	}
	doc.Aliases = splitSearchList(aliases)
	doc.Keywords = splitSearchList(keywords)
	doc.ManualProcessed = manualProcessed == 1

	return doc, nil
}

func (s *RetrievalService) loadNodeSearchDocumentFromFS(ctx context.Context, path string, cause error) (NodeSearchDocument, error) {
	dataPath := s.store.DataPath()
	if dataPath == "" {
		return NodeSearchDocument{}, cause
	}

	node, err := kb.NewStore(afero.NewOsFs()).GetNode(ctx, dataPath, path)
	if err != nil {
		return NodeSearchDocument{}, cause
	}

	doc := NodeSearchDocument{
		Path:            node.Path,
		Annotation:      node.Annotation,
		ManualProcessed: kb.ManualProcessedEffective(node.Metadata),
	}
	if title, ok := node.Metadata["title"].(string); ok {
		doc.Title = title
	}
	if nodeType, ok := node.Metadata["type"].(string); ok {
		doc.Type = nodeType
	}
	if sourceURL, ok := node.Metadata["source_url"].(string); ok {
		doc.SourceURL = sourceURL
	}
	doc.Aliases = metadataStringSlice(node.Metadata["aliases"])
	doc.Keywords = metadataStringSlice(node.Metadata["keywords"])

	return doc, nil
}

type hybridAccumulator struct {
	result          HybridSearchResult
	matchReasons    map[string]struct{}
	sourceKinds     map[string]struct{}
	score           float64
	maxVectorScore  float64
	manualProcessed bool
}

func ensureHybridAccumulator(acc map[string]*hybridAccumulator, path string) *hybridAccumulator {
	if item, ok := acc[path]; ok {
		return item
	}
	item := &hybridAccumulator{
		result: HybridSearchResult{
			Path:  path,
			Title: path,
		},
		matchReasons: make(map[string]struct{}),
		sourceKinds:  make(map[string]struct{}),
	}
	acc[path] = item

	return item
}

func (a *hybridAccumulator) applyKeywordNode(hit KeywordNodeHit) {
	a.result.Path = hit.Path
	a.result.Title = fallbackString(hit.Title, hit.Path)
	a.result.Type = hit.Type
	a.result.Annotation = hit.Annotation
	a.result.Keywords = hit.Keywords
	a.result.SourceURL = hit.SourceURL
	a.manualProcessed = hit.ManualScoped
	a.addSource("keyword", hit.Rank, 1.8, hit.RawScore)
	for _, field := range hit.MatchFields {
		a.matchReasons[field] = struct{}{}
	}
	if hit.ExactBoost > 0 {
		a.sourceKinds["exact"] = struct{}{}
		a.matchReasons["exact"] = struct{}{}
		a.score += hit.ExactBoost / 10
	}
}

func (a *hybridAccumulator) applyDocument(doc NodeSearchDocument) {
	a.result.Path = doc.Path
	a.result.Title = fallbackString(doc.Title, doc.Path)
	a.result.Type = doc.Type
	a.result.Annotation = doc.Annotation
	a.result.Keywords = doc.Keywords
	a.result.SourceURL = doc.SourceURL
	a.manualProcessed = doc.ManualProcessed
}

func (a *hybridAccumulator) addSource(kind string, rank int, weight, rawScore float64) {
	a.sourceKinds[kind] = struct{}{}
	a.score += weight/(rrfK+float64(rank)) + rawScore/100
}

func (a *hybridAccumulator) isWeakVectorOnly() bool {
	if len(a.sourceKinds) == 0 {
		return true
	}
	for kind := range a.sourceKinds {
		if !strings.HasPrefix(kind, "vector_") {
			return false
		}
	}

	return a.maxVectorScore < chatVectorCutoff
}

func normalizeRetrievalOptions(opts RetrievalOptions) RetrievalOptions {
	opts.Query = strings.TrimSpace(opts.Query)
	if opts.Mode == "" {
		opts.Mode = RetrievalModeSearch
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.TopK <= 0 {
		opts.TopK = max(opts.Limit, 10)
	}

	return opts
}

func matchesRetrievalFilters(result HybridSearchResult, opts RetrievalOptions, manualProcessed bool) bool {
	if len(opts.SourcePaths) > 0 && !containsString(opts.SourcePaths, result.Path) {
		return false
	}
	if len(opts.Types) > 0 && !containsString(opts.Types, result.Type) {
		return false
	}
	if opts.Path != "" {
		if opts.Recursive {
			if result.Path != opts.Path && !strings.HasPrefix(result.Path, strings.TrimSuffix(opts.Path, "/")+"/") {
				return false
			}
		} else if result.Path != opts.Path {
			return false
		}
	}
	if opts.ManualProcessed != nil && manualProcessed != *opts.ManualProcessed {
		return false
	}

	return true
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)

	return result
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}

	return false
}

func metadataStringSlice(value any) []string {
	switch items := value.(type) {
	case []string:
		return items
	case []any:
		result := make([]string, 0, len(items))
		for _, item := range items {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}

		return result
	default:
		return nil
	}
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

const (
	rrfK             = 60
	chatVectorCutoff = 0.55
)
