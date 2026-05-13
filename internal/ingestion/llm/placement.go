package llm

import "context"

const (
	PlacementThemeMapLimit         = 40
	PlacementCandidateThemeLimit   = 8
	PlacementCandidateKeywordLimit = 24
	PlacementSimilarNodeLimit      = 8
)

// PlacementContext is a compact local context for choosing a node theme and keywords.
type PlacementContext struct {
	Source            string             `json:"source,omitempty"`
	ExplicitThemePath string             `json:"explicit_theme_path,omitempty"`
	ThemeMap          []ThemeSummary     `json:"theme_map,omitempty"`
	CandidateThemes   []ThemeCandidate   `json:"candidate_themes,omitempty"`
	CandidateKeywords []KeywordCandidate `json:"candidate_keywords,omitempty"`
	SimilarNodes      []SimilarNode      `json:"similar_nodes,omitempty"`
}

// ThemeSummary describes an existing theme for a compact knowledge base map.
type ThemeSummary struct {
	Path        string   `json:"path"`
	NodeCount   int      `json:"node_count"`
	TopKeywords []string `json:"top_keywords,omitempty"`
}

// ThemeCandidate is a ranked existing theme suggested for the new node.
type ThemeCandidate struct {
	Path        string   `json:"path"`
	ParentPath  string   `json:"parent_path,omitempty"`
	NodeCount   int      `json:"node_count,omitempty"`
	Score       float64  `json:"score,omitempty"`
	Reasons     []string `json:"reasons,omitempty"`
	Examples    []string `json:"examples,omitempty"`
	TopKeywords []string `json:"top_keywords,omitempty"`
}

// KeywordCandidate is a ranked existing or input keyword candidate.
type KeywordCandidate struct {
	Keyword   string   `json:"keyword"`
	Score     float64  `json:"score,omitempty"`
	Frequency int      `json:"frequency,omitempty"`
	Themes    []string `json:"themes,omitempty"`
	Sources   []string `json:"sources,omitempty"`
}

// SimilarNode is a local node that helps explain placement choices.
type SimilarNode struct {
	Path           string   `json:"path"`
	Title          string   `json:"title,omitempty"`
	Annotation     string   `json:"annotation,omitempty"`
	ThemePath      string   `json:"theme_path,omitempty"`
	Keywords       []string `json:"keywords,omitempty"`
	SourceKind     string   `json:"source_kind,omitempty"`
	ContentProfile string   `json:"content_profile,omitempty"`
	Score          float64  `json:"score,omitempty"`
	MatchReasons   []string `json:"match_reasons,omitempty"`
}

// SearchPlacementCandidatesRequest is passed to the placement search tool.
type SearchPlacementCandidatesRequest struct {
	Query          string `json:"query"`
	SourceKind     string `json:"source_kind,omitempty"`
	ContentProfile string `json:"content_profile,omitempty"`
	Type           string `json:"type,omitempty"`
}

// PlacementSearcher runs a local placement search for the LLM tool loop.
type PlacementSearcher func(context.Context, SearchPlacementCandidatesRequest) (*PlacementContext, error)
