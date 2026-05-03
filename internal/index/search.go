package index

import (
	"context"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/strider2038/knowledge-db/internal/kb"
)

// SearchResult — результат векторного поиска.
type SearchResult struct {
	Path       string
	Title      string
	Annotation string
	Score      float32
}

// ChunkSearchResult — результат поиска по чанкам.
type ChunkSearchResult struct {
	NodePath string
	Heading  string
	Content  string
	Score    float32
}

// KeywordNodeHit — node-level keyword search candidate.
type KeywordNodeHit struct {
	Path         string
	Title        string
	Type         string
	Aliases      []string
	Annotation   string
	Keywords     []string
	SourceURL    string
	MatchFields  []string
	RawScore     float64
	Rank         int
	ExactBoost   float64
	ManualScoped bool
}

// KeywordChunkHit — chunk-level keyword search candidate.
type KeywordChunkHit struct {
	NodePath    string
	ChunkIndex  int
	Heading     string
	Content     string
	Snippet     string
	MatchFields []string
	RawScore    float64
	Rank        int
}

// VectorSearch выполняет поиск по node-level эмбеддингам.
func VectorSearch(ctx context.Context, store *IndexStore, provider EmbeddingProvider, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	embeddings, err := store.GetAllNodeEmbeddings(ctx)
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, nil
	}

	vec, err := provider.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	queryVec := vec[0]

	type scored struct {
		path  string
		score float32
	}
	scores := make([]scored, 0, len(embeddings))
	for _, ne := range embeddings {
		s := cosineSimilarity(queryVec, ne.Vector)
		scores = append(scores, scored{path: ne.Path, score: s})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if len(scores) > topK {
		scores = scores[:topK]
	}

	results := make([]SearchResult, len(scores))
	for i, s := range scores {
		title := s.path
		annotation := ""
		node, nodeErr := kb.GetNode(ctx, store.DataPath(), s.path)
		if nodeErr == nil {
			if metaTitle, ok := node.Metadata["title"].(string); ok && metaTitle != "" {
				title = metaTitle
			}
			if ann, ok := node.Metadata["annotation"].(string); ok {
				annotation = ann
			}
		}
		results[i] = SearchResult{
			Path:       s.path,
			Title:      title,
			Annotation: annotation,
			Score:      s.score,
		}
	}

	return results, nil
}

// KeywordSearch выполняет keyword/FTS поиск по нодам и чанкам.
func KeywordSearch(ctx context.Context, store *IndexStore, query string, topK int) ([]KeywordNodeHit, []KeywordChunkHit, error) {
	nodes, err := KeywordSearchNodes(ctx, store, query, topK)
	if err != nil {
		return nil, nil, err
	}
	chunks, err := KeywordSearchChunks(ctx, store, query, topK)
	if err != nil {
		return nil, nil, err
	}

	return nodes, chunks, nil
}

// KeywordSearchNodes выполняет keyword/FTS поиск по searchable text нод.
func KeywordSearchNodes(ctx context.Context, store *IndexStore, query string, topK int) ([]KeywordNodeHit, error) {
	if topK <= 0 {
		topK = 5
	}
	tokens := queryTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	if store.KeywordIndexMode() == "fts5" {
		return keywordSearchNodesFTS(ctx, store, query, tokens, topK)
	}

	return keywordSearchNodesScan(ctx, store, query, tokens, topK)
}

// KeywordSearchChunks выполняет keyword/FTS поиск по searchable text чанков.
func KeywordSearchChunks(ctx context.Context, store *IndexStore, query string, topK int) ([]KeywordChunkHit, error) {
	if topK <= 0 {
		topK = 5
	}
	tokens := queryTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	if store.KeywordIndexMode() == "fts5" {
		return keywordSearchChunksFTS(ctx, store, query, tokens, topK)
	}

	return keywordSearchChunksScan(ctx, store, query, tokens, topK)
}

// ChunkSearch выполняет поиск по chunk-level эмбеддингам.
func ChunkSearch(ctx context.Context, store *IndexStore, provider EmbeddingProvider, query string, topK int) ([]ChunkSearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	chunks, err := store.GetAllChunkEmbeddings(ctx)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, nil
	}

	vec, err := provider.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	queryVec := vec[0]

	type scored struct {
		chunk ChunkEmbedding
		score float32
	}
	scores := make([]scored, 0, len(chunks))
	for _, c := range chunks {
		s := cosineSimilarity(queryVec, c.Vector)
		scores = append(scores, scored{chunk: c, score: s})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if len(scores) > topK {
		scores = scores[:topK]
	}

	results := make([]ChunkSearchResult, len(scores))
	for i, s := range scores {
		results[i] = ChunkSearchResult{
			NodePath: s.chunk.NodePath,
			Heading:  s.chunk.Heading,
			Content:  s.chunk.Content,
			Score:    s.score,
		}
	}

	return results, nil
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / float32(math.Sqrt(float64(normA))*math.Sqrt(float64(normB)))
}

func keywordSearchNodesFTS(ctx context.Context, store *IndexStore, query string, tokens []string, topK int) ([]KeywordNodeHit, error) {
	rows, err := store.queryContext(ctx, `
		SELECT ns.path, ns.title, ns.type, ns.aliases, ns.annotation, ns.keywords, ns.source_url, ns.manual_processed,
			COALESCE(bm25(node_search_fts), 0) AS raw_rank
		FROM node_search_fts
		JOIN node_search ns ON ns.path = node_search_fts.path
		WHERE node_search_fts MATCH ?
		ORDER BY raw_rank
		LIMIT ?`, buildFTSQuery(tokens), topK*keywordOversampleFactor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []KeywordNodeHit
	for rows.Next() {
		var hit KeywordNodeHit
		var aliasText, keywordText string
		var manualProcessed int
		var rawRank float64
		if err := rows.Scan(&hit.Path, &hit.Title, &hit.Type, &aliasText, &hit.Annotation, &keywordText, &hit.SourceURL, &manualProcessed, &rawRank); err != nil {
			return nil, err
		}
		hit.Aliases = splitSearchList(aliasText)
		hit.Keywords = splitSearchList(keywordText)
		hit.MatchFields = matchNodeFields(query, tokens, hit)
		hit.ExactBoost = exactNodeBoost(query, hit)
		hit.RawScore = -rawRank + hit.ExactBoost
		hit.ManualScoped = manualProcessed == 1
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rankKeywordNodeHits(hits, topK), nil
}

func keywordSearchNodesScan(ctx context.Context, store *IndexStore, query string, tokens []string, topK int) ([]KeywordNodeHit, error) {
	rows, err := store.queryContext(ctx, `
		SELECT path, title, type, aliases, annotation, keywords, source_url, manual_processed, searchable_text
		FROM node_search`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []KeywordNodeHit
	for rows.Next() {
		var hit KeywordNodeHit
		var aliasText, keywordText, searchableText string
		var manualProcessed int
		if err := rows.Scan(&hit.Path, &hit.Title, &hit.Type, &aliasText, &hit.Annotation, &keywordText, &hit.SourceURL, &manualProcessed, &searchableText); err != nil {
			return nil, err
		}
		if !matchesTokens(searchableText, tokens) {
			continue
		}
		hit.Aliases = splitSearchList(aliasText)
		hit.Keywords = splitSearchList(keywordText)
		hit.MatchFields = matchNodeFields(query, tokens, hit)
		hit.ExactBoost = exactNodeBoost(query, hit)
		hit.RawScore = float64(len(hit.MatchFields)) + hit.ExactBoost
		hit.ManualScoped = manualProcessed == 1
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rankKeywordNodeHits(hits, topK), nil
}

func keywordSearchChunksFTS(ctx context.Context, store *IndexStore, query string, tokens []string, topK int) ([]KeywordChunkHit, error) {
	rows, err := store.queryContext(ctx, `
		SELECT node_path, chunk_index, heading, content, COALESCE(bm25(chunk_search_fts), 0) AS raw_rank
		FROM chunk_search_fts
		WHERE chunk_search_fts MATCH ?
		ORDER BY raw_rank
		LIMIT ?`, buildFTSQuery(tokens), topK*keywordOversampleFactor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []KeywordChunkHit
	for rows.Next() {
		var hit KeywordChunkHit
		var rawRank float64
		if err := rows.Scan(&hit.NodePath, &hit.ChunkIndex, &hit.Heading, &hit.Content, &rawRank); err != nil {
			return nil, err
		}
		hit.MatchFields = matchChunkFields(tokens, hit)
		hit.Snippet = buildSnippet(hit.Content, tokens)
		hit.RawScore = -rawRank + float64(len(hit.MatchFields))
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rankKeywordChunkHits(hits, topK), nil
}

func keywordSearchChunksScan(ctx context.Context, store *IndexStore, query string, tokens []string, topK int) ([]KeywordChunkHit, error) {
	rows, err := store.queryContext(ctx, `
		SELECT node_path, chunk_index, heading, content, searchable_text
		FROM chunk_search`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []KeywordChunkHit
	for rows.Next() {
		var hit KeywordChunkHit
		var searchableText string
		if err := rows.Scan(&hit.NodePath, &hit.ChunkIndex, &hit.Heading, &hit.Content, &searchableText); err != nil {
			return nil, err
		}
		if !matchesTokens(searchableText, tokens) {
			continue
		}
		hit.MatchFields = matchChunkFields(tokens, hit)
		hit.Snippet = buildSnippet(hit.Content, tokens)
		hit.RawScore = float64(len(hit.MatchFields))
		if strings.Contains(strings.ToLower(hit.Heading), strings.ToLower(strings.TrimSpace(query))) {
			hit.RawScore += 2
		}
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rankKeywordChunkHits(hits, topK), nil
}

func rankKeywordNodeHits(hits []KeywordNodeHit, topK int) []KeywordNodeHit {
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].RawScore == hits[j].RawScore {
			return hits[i].Path < hits[j].Path
		}

		return hits[i].RawScore > hits[j].RawScore
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}
	for i := range hits {
		hits[i].Rank = i + 1
	}

	return hits
}

func rankKeywordChunkHits(hits []KeywordChunkHit, topK int) []KeywordChunkHit {
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].RawScore == hits[j].RawScore {
			if hits[i].NodePath == hits[j].NodePath {
				return hits[i].ChunkIndex < hits[j].ChunkIndex
			}

			return hits[i].NodePath < hits[j].NodePath
		}

		return hits[i].RawScore > hits[j].RawScore
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}
	for i := range hits {
		hits[i].Rank = i + 1
	}

	return hits
}

func matchNodeFields(query string, tokens []string, hit KeywordNodeHit) []string {
	fieldValues := map[string]string{
		"path":       hit.Path,
		"title":      hit.Title,
		"type":       hit.Type,
		"aliases":    strings.Join(hit.Aliases, " "),
		"annotation": hit.Annotation,
		"keywords":   strings.Join(hit.Keywords, " "),
		"source_url": hit.SourceURL,
	}

	return matchedFields(fieldValues, query, tokens)
}

func matchChunkFields(tokens []string, hit KeywordChunkHit) []string {
	return matchedFields(map[string]string{
		"heading": hit.Heading,
		"content": hit.Content,
	}, "", tokens)
}

func matchedFields(fields map[string]string, query string, tokens []string) []string {
	var result []string
	query = strings.ToLower(strings.TrimSpace(query))
	for field, value := range fields {
		value = strings.ToLower(value)
		if value == "" {
			continue
		}
		if query != "" && strings.Contains(value, query) {
			result = append(result, field)

			continue
		}
		for _, token := range tokens {
			if strings.Contains(value, token) {
				result = append(result, field)

				break
			}
		}
	}
	sort.Strings(result)

	return result
}

func exactNodeBoost(query string, hit KeywordNodeHit) float64 {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0
	}

	var boost float64
	for _, value := range []string{hit.Title, hit.Path} {
		if strings.ToLower(strings.TrimSpace(value)) == query {
			boost += 6
		}
	}
	for _, keyword := range hit.Keywords {
		if strings.ToLower(strings.TrimSpace(keyword)) == query {
			boost += 5
		}
	}
	for _, alias := range hit.Aliases {
		if strings.ToLower(strings.TrimSpace(alias)) == query {
			boost += 5
		}
	}

	return boost
}

func matchesTokens(text string, tokens []string) bool {
	text = strings.ToLower(text)
	for _, token := range tokens {
		if !strings.Contains(text, token) {
			return false
		}
	}

	return true
}

func queryTokens(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	tokens := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		tokens = append(tokens, field)
	}

	return tokens
}

func buildFTSQuery(tokens []string) string {
	quoted := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.ReplaceAll(token, `"`, `""`)
		quoted = append(quoted, `"`+token+`"`)
	}

	return strings.Join(quoted, " AND ")
}

func splitSearchList(value string) []string {
	if value == "" {
		return nil
	}

	return strings.Fields(value)
}

func buildSnippet(content string, tokens []string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	lower := strings.ToLower(content)
	start := -1
	for _, token := range tokens {
		if idx := strings.Index(lower, token); idx >= 0 && (start == -1 || idx < start) {
			start = idx
		}
	}
	if start == -1 {
		return truncateSnippet(content)
	}
	from := max(0, start-60)
	to := min(len(content), start+180)
	snippet := strings.TrimSpace(content[from:to])
	if from > 0 {
		snippet = "..." + snippet
	}
	if to < len(content) {
		snippet += "..."
	}

	return snippet
}

func truncateSnippet(content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= 240 {
		return content
	}

	return content[:240] + "..."
}

const keywordOversampleFactor = 5
