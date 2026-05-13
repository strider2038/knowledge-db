package ingestion

import (
	"context"
	"encoding/json"
	"path"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/index"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/kb"
)

// PlacementBuilder builds compact local placement context for LLM ingestion.
type PlacementBuilder struct {
	store      *kb.Store
	basePath   string
	indexStore *index.IndexStore
}

// PlacementBuildInput contains local signals available before LLM orchestration.
type PlacementBuildInput struct {
	Text           string
	SourceURL      string
	SourceAuthor   string
	SourceKind     string
	ContentProfile string
	Type           string
}

// NewPlacementBuilder creates a builder that always supports file fallback and may use an index.
func NewPlacementBuilder(store *kb.Store, basePath string, indexStore *index.IndexStore) *PlacementBuilder {
	return &PlacementBuilder{
		store:      store,
		basePath:   basePath,
		indexStore: indexStore,
	}
}

// Build prepares candidate themes, candidate keywords and similar nodes.
// The second return value is estimatedLegacyPromptContextSize: approximate JSON
// payload size of the pre-placement prompt context (flat theme paths + unique
// keywords from all nodes), for diagnostics next to EstimatedPlacementContextSize.
func (b *PlacementBuilder) Build(ctx context.Context, input PlacementBuildInput) (*llm.PlacementContext, int, error) {
	tree, err := b.store.ReadTree(ctx, b.basePath)
	if err != nil {
		return nil, 0, errors.Errorf("build placement context: read tree: %w", err)
	}
	nodes, err := b.loadNodeProfiles(ctx)
	if err != nil {
		return nil, 0, errors.Errorf("build placement context: load nodes: %w", err)
	}

	query := buildPlacementQuery(input)
	source := "fallback"
	indexSimilar, curatedKeywords, indexErr := b.searchIndex(ctx, query)
	if indexErr == nil && (len(indexSimilar) > 0 || len(curatedKeywords) > 0) {
		source = "index"
	} else if indexErr != nil {
		clog.Warn(ctx, "ingest: placement index search failed, using file fallback", "error", indexErr)
	}

	explicitTheme := findExplicitThemePath(input.Text, collectThemes(tree))
	terms := extractPlacementTerms(query)
	similarNodes := mergeSimilarNodes(indexSimilar, scoreSimilarNodes(nodes, terms, input))
	themeProfiles := buildThemeProfiles(tree, nodes)
	themeMap := buildThemeMap(themeProfiles)
	candidateThemes := scoreThemeCandidates(themeProfiles, similarNodes, terms, input, explicitTheme)
	candidateKeywords := scoreKeywordCandidates(nodes, candidateThemes, similarNodes, terms, curatedKeywords)

	legacyPromptContextSize := estimatedLegacyPromptContextSize(tree, nodes)

	return &llm.PlacementContext{
		Source:            source,
		ExplicitThemePath: explicitTheme,
		ThemeMap:          limitThemeSummary(themeMap, llm.PlacementThemeMapLimit),
		CandidateThemes:   limitThemeCandidates(candidateThemes, llm.PlacementCandidateThemeLimit),
		CandidateKeywords: limitKeywordCandidates(candidateKeywords, llm.PlacementCandidateKeywordLimit),
		SimilarNodes:      limitSimilarNodes(similarNodes, llm.PlacementSimilarNodeLimit),
	}, legacyPromptContextSize, nil
}

// EstimatedPlacementContextSize returns a stable approximate prompt payload size.
func EstimatedPlacementContextSize(ctx llm.PlacementContext) int {
	data, err := json.Marshal(ctx)
	if err != nil {
		return 0
	}

	return len(data)
}

// estimatedLegacyPromptContextSize approximates the old ingestion prompt attachment:
// all theme paths from the tree plus all unique keywords from node frontmatter,
// encoded as JSON for a size metric comparable to EstimatedPlacementContextSize.
func estimatedLegacyPromptContextSize(tree *kb.TreeNode, nodes []nodeProfile) int {
	themes := collectThemes(tree)
	seen := make(map[string]struct{}, 256)
	keywords := make([]string, 0, 256)
	for _, n := range nodes {
		for _, kw := range n.Keywords {
			kw = strings.TrimSpace(kw)
			if kw == "" {
				continue
			}
			if _, ok := seen[kw]; ok {
				continue
			}
			seen[kw] = struct{}{}
			keywords = append(keywords, kw)
		}
	}
	slices.Sort(keywords)
	slices.Sort(themes)
	data, err := json.Marshal(struct {
		Themes   []string `json:"themes"`
		Keywords []string `json:"keywords"`
	}{
		Themes:   themes,
		Keywords: keywords,
	})
	if err != nil {
		return 0
	}

	return len(data)
}

func (b *PlacementBuilder) searchIndex(ctx context.Context, query string) ([]llm.SimilarNode, []string, error) {
	if b.indexStore == nil || strings.TrimSpace(query) == "" {
		return nil, nil, nil
	}

	hits, err := index.KeywordSearchNodes(ctx, b.indexStore, query, llm.PlacementSimilarNodeLimit*2)
	if err != nil {
		return nil, nil, errors.Errorf("keyword search nodes: %w", err)
	}
	similar := make([]llm.SimilarNode, 0, len(hits))
	for _, hit := range hits {
		similar = append(similar, llm.SimilarNode{
			Path:         hit.Path,
			Title:        hit.Title,
			Annotation:   hit.Annotation,
			ThemePath:    themePathFromNodePath(hit.Path),
			Keywords:     hit.Keywords,
			Score:        hit.RawScore,
			MatchReasons: append([]string{"index"}, hit.MatchFields...),
		})
	}

	vocabulary, err := b.indexStore.SearchVocabulary(ctx, index.SearchVocabularyOptions{
		Limit:                     llm.PlacementCandidateKeywordLimit,
		MaxDocumentFrequencyRatio: 0.5,
	})
	if err != nil {
		return nil, nil, errors.Errorf("search vocabulary: %w", err)
	}

	return similar, vocabulary, nil
}

type nodeProfile struct {
	Path           string
	ThemePath      string
	Title          string
	Annotation     string
	Content        string
	Keywords       []string
	SourceKind     string
	ContentProfile string
	Type           string
}

func (b *PlacementBuilder) loadNodeProfiles(ctx context.Context) ([]nodeProfile, error) {
	items, err := b.store.ListAllNodes(ctx, b.basePath)
	if err != nil {
		return nil, err
	}

	profiles := make([]nodeProfile, 0, len(items))
	for _, item := range items {
		if strings.HasSuffix(item.Path, ".ru") {
			continue
		}
		node, err := b.store.GetNode(ctx, b.basePath, item.Path)
		if err != nil {
			continue
		}
		profiles = append(profiles, nodeProfile{
			Path:           node.Path,
			ThemePath:      themePathFromNodePath(node.Path),
			Title:          metadataString(node.Metadata, "title"),
			Annotation:     firstNonEmptyString(metadataString(node.Metadata, "annotation"), node.Annotation),
			Content:        node.Content,
			Keywords:       metadataStringSlice(node.Metadata, "keywords"),
			SourceKind:     metadataString(node.Metadata, "source_kind"),
			ContentProfile: metadataString(node.Metadata, "content_profile"),
			Type:           metadataString(node.Metadata, "type"),
		})
	}

	return profiles, nil
}

type themeProfile struct {
	Path            string
	ParentPath      string
	NodeCount       int
	KeywordCounts   map[string]int
	SourceKinds     map[string]int
	ContentProfiles map[string]int
	Examples        []string
}

type scoredTheme struct {
	candidate llm.ThemeCandidate
	reasons   map[string]struct{}
}

func buildThemeProfiles(tree *kb.TreeNode, nodes []nodeProfile) map[string]*themeProfile {
	profiles := make(map[string]*themeProfile)
	for _, theme := range collectThemes(tree) {
		profiles[theme] = newThemeProfile(theme)
	}
	for _, node := range nodes {
		if node.ThemePath == "" {
			continue
		}
		profile := profiles[node.ThemePath]
		if profile == nil {
			profile = newThemeProfile(node.ThemePath)
			profiles[node.ThemePath] = profile
		}
		profile.NodeCount++
		for _, keyword := range node.Keywords {
			profile.KeywordCounts[keyword]++
		}
		if node.SourceKind != "" {
			profile.SourceKinds[node.SourceKind]++
		}
		if node.ContentProfile != "" {
			profile.ContentProfiles[node.ContentProfile]++
		}
		if len(profile.Examples) < 3 {
			profile.Examples = append(profile.Examples, firstNonEmptyString(node.Title, node.Path))
		}
	}

	return profiles
}

func newThemeProfile(themePath string) *themeProfile {
	return &themeProfile{
		Path:            themePath,
		ParentPath:      parentThemePath(themePath),
		KeywordCounts:   make(map[string]int),
		SourceKinds:     make(map[string]int),
		ContentProfiles: make(map[string]int),
	}
}

func scoreSimilarNodes(nodes []nodeProfile, terms []string, input PlacementBuildInput) []llm.SimilarNode {
	similar := make([]llm.SimilarNode, 0, len(nodes))
	for _, node := range nodes {
		score, reasons := scoreNode(node, terms, input)
		if score <= 0 {
			continue
		}
		similar = append(similar, llm.SimilarNode{
			Path:           node.Path,
			Title:          node.Title,
			Annotation:     node.Annotation,
			ThemePath:      node.ThemePath,
			Keywords:       node.Keywords,
			SourceKind:     node.SourceKind,
			ContentProfile: node.ContentProfile,
			Score:          score,
			MatchReasons:   reasons,
		})
	}
	sort.Slice(similar, func(i, j int) bool {
		if similar[i].Score == similar[j].Score {
			return similar[i].Path < similar[j].Path
		}

		return similar[i].Score > similar[j].Score
	})

	return similar
}

func scoreNode(node nodeProfile, terms []string, input PlacementBuildInput) (float64, []string) {
	var score float64
	reasonSet := make(map[string]struct{})
	title := normalizePlacementText(node.Title)
	annotation := normalizePlacementText(node.Annotation)
	body := normalizePlacementText(node.Content)
	pathText := normalizePlacementText(strings.ReplaceAll(node.Path, "/", " "))
	keywords := normalizePlacementText(strings.Join(node.Keywords, " "))

	for _, term := range terms {
		if strings.Contains(pathText, term) {
			score += 4
			reasonSet["path"] = struct{}{}
		}
		if strings.Contains(title, term) {
			score += 3
			reasonSet["title"] = struct{}{}
		}
		if strings.Contains(keywords, term) {
			score += 3
			reasonSet["keywords"] = struct{}{}
		}
		if strings.Contains(annotation, term) {
			score += 1.5
			reasonSet["annotation"] = struct{}{}
		}
		if strings.Contains(body, term) {
			score += 0.5
			reasonSet["body"] = struct{}{}
		}
	}
	if input.SourceKind != "" && node.SourceKind == input.SourceKind {
		score += 1
		reasonSet["source_kind"] = struct{}{}
	}
	if input.ContentProfile != "" && node.ContentProfile == input.ContentProfile {
		score += 1
		reasonSet["content_profile"] = struct{}{}
	}

	return score, sortedKeys(reasonSet)
}

func scoreThemeCandidates(
	profiles map[string]*themeProfile,
	similarNodes []llm.SimilarNode,
	terms []string,
	input PlacementBuildInput,
	explicitTheme string,
) []llm.ThemeCandidate {
	scored := make(map[string]*scoredTheme, len(profiles))
	for themePath, profile := range profiles {
		score, reasons := scoreThemeProfile(themePath, profile, terms, input, explicitTheme)
		scored[themePath] = &scoredTheme{candidate: llm.ThemeCandidate{
			Path:        themePath,
			ParentPath:  profile.ParentPath,
			NodeCount:   profile.NodeCount,
			Score:       score,
			Examples:    profile.Examples,
			TopKeywords: topKeywordCounts(profile.KeywordCounts, 5),
		}, reasons: reasons}
	}

	for _, node := range similarNodes {
		themePath := node.ThemePath
		if themePath == "" {
			continue
		}
		addThemeSignal(scored, profiles, themePath, node.Score, "similar_node", node.Title)
		parent := parentThemePath(themePath)
		if parent != "" {
			addThemeSignal(scored, profiles, parent, node.Score*0.35, "similar_node_parent", node.Title)
		}
	}

	candidates := make([]llm.ThemeCandidate, 0, len(scored))
	for _, item := range scored {
		if item.candidate.Score <= 0 {
			continue
		}
		item.candidate.Reasons = sortedKeys(item.reasons)
		candidates = append(candidates, item.candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Path < candidates[j].Path
		}

		return candidates[i].Score > candidates[j].Score
	})

	return candidates
}

func scoreThemeProfile(
	themePath string,
	profile *themeProfile,
	terms []string,
	input PlacementBuildInput,
	explicitTheme string,
) (float64, map[string]struct{}) {
	score := densityScore(profile.NodeCount)
	reasons := map[string]struct{}{}
	if score > 0 {
		reasons["theme_density"] = struct{}{}
	}
	score += scoreThemeTextSignals(themePath, profile.KeywordCounts, terms, reasons)
	score += scoreThemeProfileSignals(profile, input, reasons)
	if explicitTheme != "" && themePath == explicitTheme {
		score += 100
		reasons["explicit_user_instruction"] = struct{}{}
	}

	return score, reasons
}

func scoreThemeTextSignals(themePath string, keywordCounts map[string]int, terms []string, reasons map[string]struct{}) float64 {
	var score float64
	pathText := normalizePlacementText(strings.ReplaceAll(themePath, "/", " "))
	for _, term := range terms {
		if strings.Contains(pathText, term) {
			score += 4
			reasons["path"] = struct{}{}
		}
		if themeKeywordsContainTerm(keywordCounts, term) {
			score += 2
			reasons["top_keywords"] = struct{}{}
		}
	}

	return score
}

func themeKeywordsContainTerm(keywordCounts map[string]int, term string) bool {
	for keyword := range keywordCounts {
		if strings.Contains(normalizePlacementText(keyword), term) {
			return true
		}
	}

	return false
}

func scoreThemeProfileSignals(profile *themeProfile, input PlacementBuildInput, reasons map[string]struct{}) float64 {
	var score float64
	if input.SourceKind != "" && profile.SourceKinds[input.SourceKind] > 0 {
		score += 1.5
		reasons["source_kind"] = struct{}{}
	}
	if input.ContentProfile != "" && profile.ContentProfiles[input.ContentProfile] > 0 {
		score += 1.5
		reasons["content_profile"] = struct{}{}
	}

	return score
}

func addThemeSignal(
	scored map[string]*scoredTheme,
	profiles map[string]*themeProfile,
	themePath string,
	score float64,
	reason string,
	example string,
) {
	item := scored[themePath]
	if item == nil {
		profile := profiles[themePath]
		if profile == nil {
			profile = newThemeProfile(themePath)
		}
		item = &scoredTheme{candidate: llm.ThemeCandidate{
			Path:        themePath,
			ParentPath:  profile.ParentPath,
			NodeCount:   profile.NodeCount,
			Examples:    profile.Examples,
			TopKeywords: topKeywordCounts(profile.KeywordCounts, 5),
		}, reasons: make(map[string]struct{})}
		scored[themePath] = item
	}
	item.candidate.Score += score
	item.reasons[reason] = struct{}{}
	if example != "" && len(item.candidate.Examples) < 3 && !containsString(item.candidate.Examples, example) {
		item.candidate.Examples = append(item.candidate.Examples, example)
	}
}

type keywordScore struct {
	keyword   string
	score     float64
	frequency int
	themes    map[string]struct{}
	sources   map[string]struct{}
}

func scoreKeywordCandidates(
	nodes []nodeProfile,
	candidateThemes []llm.ThemeCandidate,
	similarNodes []llm.SimilarNode,
	terms []string,
	curatedKeywords []string,
) []llm.KeywordCandidate {
	scores := make(map[string]*keywordScore)
	themeSet := addCandidateThemeKeywords(scores, candidateThemes)
	addNodeKeywordFrequencies(scores, nodes, themeSet)
	addSimilarNodeKeywords(scores, similarNodes)
	addInputTermKeywords(scores, nodes, terms)
	addCuratedKeywords(scores, curatedKeywords)

	candidates := make([]llm.KeywordCandidate, 0, len(scores))
	for _, item := range scores {
		if strings.TrimSpace(item.keyword) == "" {
			continue
		}
		candidates = append(candidates, llm.KeywordCandidate{
			Keyword:   item.keyword,
			Score:     item.score,
			Frequency: item.frequency,
			Themes:    sortedKeys(item.themes),
			Sources:   sortedKeys(item.sources),
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			if candidates[i].Frequency == candidates[j].Frequency {
				return candidates[i].Keyword < candidates[j].Keyword
			}

			return candidates[i].Frequency > candidates[j].Frequency
		}

		return candidates[i].Score > candidates[j].Score
	})

	return candidates
}

func addCandidateThemeKeywords(scores map[string]*keywordScore, candidateThemes []llm.ThemeCandidate) map[string]struct{} {
	themeSet := make(map[string]struct{}, len(candidateThemes))
	for i, theme := range candidateThemes {
		themeSet[theme.Path] = struct{}{}
		weight := maxFloat(1, float64(len(candidateThemes)-i))
		for _, keyword := range theme.TopKeywords {
			addKeywordScore(scores, keyword, weight, theme.Path, "candidate_theme")
		}
	}

	return themeSet
}

func addNodeKeywordFrequencies(scores map[string]*keywordScore, nodes []nodeProfile, themeSet map[string]struct{}) {
	for _, node := range nodes {
		score, source := nodeKeywordWeight(node.ThemePath, themeSet)
		for _, keyword := range node.Keywords {
			addKeywordScore(scores, keyword, score, node.ThemePath, source)
		}
	}
}

func nodeKeywordWeight(themePath string, themeSet map[string]struct{}) (float64, string) {
	if _, ok := themeSet[themePath]; ok {
		return 1.5, "candidate_theme"
	}

	return 0.25, "frequency"
}

func addSimilarNodeKeywords(scores map[string]*keywordScore, similarNodes []llm.SimilarNode) {
	for _, node := range similarNodes {
		for _, keyword := range node.Keywords {
			addKeywordScore(scores, keyword, 4+node.Score*0.1, node.ThemePath, "similar_node")
		}
	}
}

func addInputTermKeywords(scores map[string]*keywordScore, nodes []nodeProfile, terms []string) {
	for _, term := range terms {
		if len([]rune(term)) < 3 {
			continue
		}
		addMatchingNodeKeywords(scores, nodes, term)
	}
}

func addMatchingNodeKeywords(scores map[string]*keywordScore, nodes []nodeProfile, term string) {
	for _, node := range nodes {
		for _, keyword := range node.Keywords {
			if strings.Contains(normalizePlacementText(keyword), term) {
				addKeywordScore(scores, keyword, 3, node.ThemePath, "input_term")
			}
		}
	}
}

func addCuratedKeywords(scores map[string]*keywordScore, curatedKeywords []string) {
	for _, keyword := range curatedKeywords {
		addKeywordScore(scores, keyword, 1, "", "curated_vocabulary")
	}
}

func addKeywordScore(scores map[string]*keywordScore, keyword string, score float64, themePath, source string) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return
	}
	key := normalizePlacementText(keyword)
	item := scores[key]
	if item == nil {
		item = &keywordScore{
			keyword: keyword,
			themes:  make(map[string]struct{}),
			sources: make(map[string]struct{}),
		}
		scores[key] = item
	}
	item.score += score
	item.frequency++
	if themePath != "" {
		item.themes[themePath] = struct{}{}
	}
	if source != "" {
		item.sources[source] = struct{}{}
	}
}

func buildThemeMap(profiles map[string]*themeProfile) []llm.ThemeSummary {
	summaries := make([]llm.ThemeSummary, 0, len(profiles))
	for _, profile := range profiles {
		summaries = append(summaries, llm.ThemeSummary{
			Path:        profile.Path,
			NodeCount:   profile.NodeCount,
			TopKeywords: topKeywordCounts(profile.KeywordCounts, 3),
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].NodeCount == summaries[j].NodeCount {
			return summaries[i].Path < summaries[j].Path
		}

		return summaries[i].NodeCount > summaries[j].NodeCount
	})

	return summaries
}

func mergeSimilarNodes(primary, fallback []llm.SimilarNode) []llm.SimilarNode {
	merged := make(map[string]llm.SimilarNode, len(primary)+len(fallback))
	for _, node := range fallback {
		merged[node.Path] = node
	}
	for _, node := range primary {
		existing, ok := merged[node.Path]
		if ok {
			node.Score += existing.Score
			node.MatchReasons = uniqueStrings(append(node.MatchReasons, existing.MatchReasons...))
			if node.SourceKind == "" {
				node.SourceKind = existing.SourceKind
			}
			if node.ContentProfile == "" {
				node.ContentProfile = existing.ContentProfile
			}
		}
		merged[node.Path] = node
	}
	result := make([]llm.SimilarNode, 0, len(merged))
	for _, node := range merged {
		result = append(result, node)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].Path < result[j].Path
		}

		return result[i].Score > result[j].Score
	})

	return result
}

func buildPlacementQuery(input PlacementBuildInput) string {
	parts := []string{input.Text, input.SourceURL, input.SourceAuthor, input.SourceKind, input.ContentProfile, input.Type}

	return strings.Join(parts, " ")
}

func extractPlacementTerms(text string) []string {
	fields := strings.FieldsFunc(normalizePlacementText(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	seen := make(map[string]struct{}, len(fields))
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len([]rune(field)) < 2 {
			continue
		}
		if _, ok := placementStopWords[field]; ok {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		terms = append(terms, field)
	}
	if len(terms) > 80 {
		terms = terms[:80]
	}

	return terms
}

func findExplicitThemePath(text string, themes []string) string {
	normalized := normalizePlacementText(text)
	best := ""
	for _, theme := range themes {
		if theme == "" {
			continue
		}
		needle := normalizePlacementText(theme)
		if strings.Contains(normalized, "сохрани в "+needle) ||
			strings.Contains(normalized, "сохранить в "+needle) ||
			strings.Contains(normalized, "save to "+needle) ||
			strings.Contains(normalized, "put into "+needle) {
			if len(theme) > len(best) {
				best = theme
			}
		}
	}

	return best
}

func normalizePlacementText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = strings.ReplaceAll(text, "_", " ")
	text = strings.ReplaceAll(text, "-", " ")
	text = strings.Join(strings.Fields(text), " ")

	return text
}

func themePathFromNodePath(nodePath string) string {
	nodePath = strings.Trim(path.Clean(strings.TrimSpace(nodePath)), ".")
	if nodePath == "" || nodePath == "/" {
		return ""
	}

	return path.Dir(nodePath)
}

func parentThemePath(themePath string) string {
	parent := path.Dir(strings.TrimSpace(themePath))
	if parent == "." || parent == "/" {
		return ""
	}

	return parent
}

func metadataString(meta map[string]any, key string) string {
	value, _ := meta[key].(string)

	return strings.TrimSpace(value)
}

func metadataStringSlice(meta map[string]any, key string) []string {
	switch value := meta[key].(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		result := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				result = append(result, strings.TrimSpace(s))
			}
		}

		return result
	default:
		return nil
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}

func topKeywordCounts(counts map[string]int, limit int) []string {
	type pair struct {
		keyword string
		count   int
	}
	pairs := make([]pair, 0, len(counts))
	for keyword, count := range counts {
		if strings.TrimSpace(keyword) == "" {
			continue
		}
		pairs = append(pairs, pair{keyword: keyword, count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].keyword < pairs[j].keyword
		}

		return pairs[i].count > pairs[j].count
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	result := make([]string, len(pairs))
	for i, pair := range pairs {
		result[i] = pair.keyword
	}

	return result
}

func densityScore(count int) float64 {
	switch {
	case count >= 10:
		return 2.5
	case count >= 5:
		return 1.5
	case count > 0:
		return 0.5
	default:
		return 0
	}
}

func sortedKeys(set map[string]struct{}) []string {
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	sort.Strings(result)

	return result
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func containsString(values []string, needle string) bool {
	return slices.Contains(values, needle)
}

func limitThemeSummary(items []llm.ThemeSummary, limit int) []llm.ThemeSummary {
	if len(items) > limit {
		return items[:limit]
	}

	return items
}

func limitThemeCandidates(items []llm.ThemeCandidate, limit int) []llm.ThemeCandidate {
	if len(items) > limit {
		return items[:limit]
	}

	return items
}

func limitKeywordCandidates(items []llm.KeywordCandidate, limit int) []llm.KeywordCandidate {
	if len(items) > limit {
		return items[:limit]
	}

	return items
}

func limitSimilarNodes(items []llm.SimilarNode, limit int) []llm.SimilarNode {
	if len(items) > limit {
		return items[:limit]
	}

	return items
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}

var placementStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "for": {}, "from": {}, "in": {}, "is": {}, "of": {}, "on": {}, "or": {}, "the": {}, "to": {}, "with": {},
	"автор": {}, "без": {}, "в": {}, "во": {}, "для": {}, "и": {}, "из": {}, "как": {}, "на": {}, "не": {}, "о": {}, "об": {}, "от": {}, "по": {}, "про": {}, "с": {}, "со": {}, "что": {}, "это": {},
	"source": {}, "kind": {}, "content": {}, "profile": {}, "recommended": {}, "type": {}, "url": {},
}
