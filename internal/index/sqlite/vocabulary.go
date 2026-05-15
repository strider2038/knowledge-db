package sqlite

import (
	"context"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/index"
)

func (s *Store) SearchVocabulary(ctx context.Context, opts index.SearchVocabularyOptions) ([]string, error) {
	opts = normalizeSearchVocabularyOptions(opts)
	rows, err := s.QueryContext(ctx, `SELECT title, aliases, keywords FROM node_search`)
	if err != nil {
		return nil, errors.Errorf("search vocabulary: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]*vocabularyTermStats)
	totalDocs := 0
	for rows.Next() {
		totalDocs++
		var title, aliases, keywords string
		if err := rows.Scan(&title, &aliases, &keywords); err != nil {
			return nil, errors.Errorf("search vocabulary scan: %w", err)
		}

		seen := make(map[string]struct{})
		addVocabularyTerms(stats, seen, splitSearchList(aliases), 4)
		addVocabularyTerms(stats, seen, splitSearchList(keywords), 3)
		addVocabularyTerms(stats, seen, titleVocabularyTerms(title), 2)
		for term := range seen {
			stats[term].documents++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if totalDocs == 0 {
		return nil, nil
	}

	maxDocs := max(1, int(float64(totalDocs)*opts.MaxDocumentFrequencyRatio))
	terms := make([]vocabularyTermStats, 0, len(stats))
	for _, stat := range stats {
		if stat.documents > maxDocs || !validVocabularyTerm(stat.term, opts) {
			continue
		}
		terms = append(terms, *stat)
	}
	sort.Slice(terms, func(i, j int) bool {
		if terms[i].score == terms[j].score {
			if terms[i].documents == terms[j].documents {
				return terms[i].term < terms[j].term
			}

			return terms[i].documents < terms[j].documents
		}

		return terms[i].score > terms[j].score
	})
	if len(terms) > opts.Limit {
		terms = terms[:opts.Limit]
	}

	result := make([]string, len(terms))
	for i, term := range terms {
		result[i] = term.term
	}

	return result, nil
}

type vocabularyTermStats struct {
	term      string
	score     float64
	documents int
}

func normalizeSearchVocabularyOptions(opts index.SearchVocabularyOptions) index.SearchVocabularyOptions {
	if opts.Limit <= 0 {
		opts.Limit = 150
	}
	if opts.MaxDocumentFrequencyRatio <= 0 {
		opts.MaxDocumentFrequencyRatio = 0.3
	}
	if opts.MinTermRunes <= 0 {
		opts.MinTermRunes = 3
	}
	if opts.MaxTermRunes <= 0 {
		opts.MaxTermRunes = 64
	}
	if opts.MaxWords <= 0 {
		opts.MaxWords = 5
	}

	return opts
}

func addVocabularyTerms(stats map[string]*vocabularyTermStats, seen map[string]struct{}, terms []string, weight float64) {
	for _, term := range terms {
		term = normalizeVocabularyTerm(term)
		if term == "" {
			continue
		}
		stat, ok := stats[term]
		if !ok {
			stat = &vocabularyTermStats{term: term}
			stats[term] = stat
		}
		stat.score += weight
		seen[term] = struct{}{}
	}
}

func titleVocabularyTerms(title string) []string {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}

	terms := []string{title}
	for _, part := range strings.FieldsFunc(title, func(r rune) bool {
		return r == ':' || r == '-' || r == '—' || r == '–' || r == '|' || unicode.Is(unicode.Pd, r)
	}) {
		part = strings.TrimSpace(part)
		if part != "" && part != title {
			terms = append(terms, part)
		}
	}

	return terms
}

func validVocabularyTerm(term string, opts index.SearchVocabularyOptions) bool {
	runes := utf8.RuneCountInString(term)
	if runes < opts.MinTermRunes || runes > opts.MaxTermRunes {
		return false
	}
	if len(strings.Fields(term)) > opts.MaxWords {
		return false
	}

	if _, ok := index.KeywordStopWords[term]; ok {
		return false
	}

	return true
}

func normalizeVocabularyTerm(term string) string {
	term = strings.TrimSpace(term)
	term = strings.Trim(term, ".,;:!?()[]{}\"'`")
	term = strings.Join(strings.Fields(term), " ")

	return term
}

func splitSearchList(value string) []string {
	if value == "" {
		return nil
	}

	return strings.Fields(value)
}
