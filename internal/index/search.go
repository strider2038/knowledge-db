package index

import (
	"context"
	"math"
	"sort"
	"strconv"
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
	TokenBoost   float64
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
	tokens := keywordQueryTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	if store.KeywordIndexMode() == keywordIndexModeFTS5 {
		return fallbackKeywordSearchNodes(ctx, store, query, tokens, topK, keywordSearchNodesFTS)
	}

	return fallbackKeywordSearchNodes(ctx, store, query, tokens, topK, keywordSearchNodesScan)
}

// KeywordSearchChunks выполняет keyword/FTS поиск по searchable text чанков.
func KeywordSearchChunks(ctx context.Context, store *IndexStore, query string, topK int) ([]KeywordChunkHit, error) {
	if topK <= 0 {
		topK = 5
	}
	tokens := keywordQueryTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	if store.KeywordIndexMode() == keywordIndexModeFTS5 {
		return fallbackKeywordSearchChunks(ctx, store, query, tokens, topK, keywordSearchChunksFTS)
	}

	return fallbackKeywordSearchChunks(ctx, store, query, tokens, topK, keywordSearchChunksScan)
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
		hit.ExactBoost = exactNodeBoost(query, hit)
		hit.TokenBoost = exactTokenBoost(tokens, hit)
		hit.MatchFields = matchNodeFields(query, tokens, hit)
		hit.RawScore = -rawRank + hit.ExactBoost + hit.TokenBoost
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
		hit.ExactBoost = exactNodeBoost(query, hit)
		hit.TokenBoost = exactTokenBoost(tokens, hit)
		hit.MatchFields = matchNodeFields(query, tokens, hit)
		hit.RawScore = float64(len(hit.MatchFields)) + hit.ExactBoost + hit.TokenBoost
		hit.ManualScoped = manualProcessed == 1
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rankKeywordNodeHits(hits, topK), nil
}

type keywordNodeSearchFunc func(context.Context, *IndexStore, string, []string, int) ([]KeywordNodeHit, error)

func fallbackKeywordSearchNodes(ctx context.Context, store *IndexStore, query string, tokens []string, topK int, search keywordNodeSearchFunc) ([]KeywordNodeHit, error) {
	hits, err := search(ctx, store, query, tokens, topK)
	if err != nil || len(hits) > 0 || len(tokens) <= 1 {
		return hits, err
	}

	merged := make(map[string]KeywordNodeHit)
	for _, token := range tokens {
		tokenHits, tokenErr := search(ctx, store, query, []string{token}, topK)
		if tokenErr != nil {
			return nil, tokenErr
		}
		for _, hit := range tokenHits {
			if existing, ok := merged[hit.Path]; ok && existing.RawScore >= hit.RawScore {
				continue
			}
			merged[hit.Path] = hit
		}
	}

	hits = make([]KeywordNodeHit, 0, len(merged))
	for _, hit := range merged {
		hits = append(hits, hit)
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

type keywordChunkSearchFunc func(context.Context, *IndexStore, string, []string, int) ([]KeywordChunkHit, error)

func fallbackKeywordSearchChunks(ctx context.Context, store *IndexStore, query string, tokens []string, topK int, search keywordChunkSearchFunc) ([]KeywordChunkHit, error) {
	hits, err := search(ctx, store, query, tokens, topK)
	if err != nil || len(hits) > 0 || len(tokens) <= 1 {
		return hits, err
	}

	merged := make(map[string]KeywordChunkHit)
	for _, token := range tokens {
		tokenHits, tokenErr := search(ctx, store, query, []string{token}, topK)
		if tokenErr != nil {
			return nil, tokenErr
		}
		for _, hit := range tokenHits {
			key := hit.NodePath + "\x00" + strconv.Itoa(hit.ChunkIndex)
			if existing, ok := merged[key]; ok && existing.RawScore >= hit.RawScore {
				continue
			}
			merged[key] = hit
		}
	}

	hits = make([]KeywordChunkHit, 0, len(merged))
	for _, hit := range merged {
		hits = append(hits, hit)
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
	result := matchedFields(fieldValues, query, tokens)
	if len(result) == 0 {
		result = append(result, "searchable_text")
	}
	if hit.TokenBoost > 0 {
		result = append(result, "exact_token")
	}
	sort.Strings(result)

	return result
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

func exactTokenBoost(tokens []string, hit KeywordNodeHit) float64 {
	var boost float64
	pathSegments := queryTokens(hit.Path)
	titleTokens := queryTokens(hit.Title)
	for _, token := range tokens {
		if containsToken(pathSegments, token) {
			boost += 8
		}
		if containsToken(hit.Keywords, token) {
			boost += 8
		}
		if containsToken(hit.Aliases, token) {
			boost += 8
		}
		if containsToken(titleTokens, token) {
			boost += 6
		}
	}

	return boost
}

func containsToken(values []string, token string) bool {
	token = normalizeSearchToken(token)
	if token == "" {
		return false
	}
	for _, value := range values {
		if normalizeSearchToken(value) == token {
			return true
		}
	}

	return false
}

func normalizeSearchToken(token string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	if len([]rune(token)) <= 4 {
		return token
	}
	for _, suffix := range russianSearchSuffixes {
		if strings.HasSuffix(token, suffix) && len([]rune(token)) > len([]rune(suffix))+3 {
			return strings.TrimSuffix(token, suffix)
		}
	}

	return token
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

func keywordQueryTokens(query string) []string {
	tokens := queryTokens(query)
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := keywordStopWords[token]; ok {
			continue
		}
		filtered = append(filtered, token)
	}
	if len(filtered) == 0 {
		return tokens
	}

	return filtered
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
	lowerRunes := []rune(strings.ToLower(content))
	contentRunes := []rune(content)
	start := -1
	for _, token := range tokens {
		if idx := indexRunes(lowerRunes, []rune(strings.ToLower(token))); idx >= 0 && (start == -1 || idx < start) {
			start = idx
		}
	}
	if start == -1 {
		return truncateSnippet(content)
	}
	from := max(0, start-60)
	to := min(len(contentRunes), start+180)
	snippet := strings.TrimSpace(string(contentRunes[from:to]))
	if from > 0 {
		snippet = "..." + snippet
	}
	if to < len(contentRunes) {
		snippet += "..."
	}

	return snippet
}

func indexRunes(text, pattern []rune) int {
	if len(pattern) == 0 || len(pattern) > len(text) {
		return -1
	}
	for i := 0; i <= len(text)-len(pattern); i++ {
		matched := true
		for j := range pattern {
			if text[i+j] != pattern[j] {
				matched = false

				break
			}
		}
		if matched {
			return i
		}
	}

	return -1
}

func truncateSnippet(content string) string {
	content = strings.TrimSpace(content)
	runes := []rune(content)
	if len(runes) <= 240 {
		return content
	}

	return string(runes[:240]) + "..."
}

const keywordOversampleFactor = 5

var keywordStopWords = map[string]struct{}{
	"a":        {},
	"an":       {},
	"are":      {},
	"for":      {},
	"how":      {},
	"in":       {},
	"is":       {},
	"of":       {},
	"on":       {},
	"the":      {},
	"what":     {},
	"where":    {},
	"which":    {},
	"ai":       {},
	"база":     {},
	"базе":     {},
	"в":        {},
	"во":       {},
	"где":      {},
	"для":      {},
	"есть":     {},
	"из":       {},
	"ии":       {},
	"как":      {},
	"какая":    {},
	"какие":    {},
	"какой":    {},
	"какое":    {},
	"найди":    {},
	"на":       {},
	"по":       {},
	"покажи":   {},
	"про":      {},
	"расскажи": {},
	"что":      {},
}

var russianSearchSuffixes = []string{
	"ами",
	"ями",
	"ого",
	"его",
	"ому",
	"ему",
	"ыми",
	"ими",
	"ах",
	"ях",
	"ом",
	"ем",
	"ой",
	"ый",
	"ий",
	"ая",
	"ое",
	"ые",
	"ую",
	"юю",
	"а",
	"я",
	"ы",
	"и",
	"е",
	"у",
	"ю",
}
