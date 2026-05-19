package kb

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"github.com/muonsoft/errors"
)

const (
	// MaxLabelsPerNode — максимум меток на один узел.
	MaxLabelsPerNode = 32
	// MaxLabelLength — максимальная длина одной метки после trim.
	MaxLabelLength = 64
)

// ErrInvalidLabels — невалидный список меток (лимиты, запятая в значении и т.п.).
var ErrInvalidLabels = errors.New("invalid labels")

// LabelsEffective returns normalized labels from node metadata (empty slice if absent).
func LabelsEffective(meta map[string]any) []string {
	raw := extractStringList(meta, "labels")
	if len(raw) == 0 {
		return []string{}
	}
	normalized, err := NormalizeLabels(raw)
	if err != nil {
		return []string{}
	}

	return normalized
}

// NormalizeLabels trims, dedupes case-insensitively (keeps first spelling), enforces limits.
func NormalizeLabels(labels []string) ([]string, error) {
	if len(labels) == 0 {
		return []string{}, nil
	}
	seen := make(map[string]struct{}, len(labels))
	out := make([]string, 0, len(labels))
	for _, label := range labels {
		normalized := strings.TrimSpace(label)
		if normalized == "" {
			continue
		}
		if strings.ContainsRune(normalized, ',') {
			return nil, errors.Errorf("normalize labels: %w: comma not allowed", ErrInvalidLabels)
		}
		if len([]rune(normalized)) > MaxLabelLength {
			return nil, errors.Errorf("normalize labels: %w: label too long", ErrInvalidLabels)
		}
		for _, r := range normalized {
			if unicode.IsControl(r) {
				return nil, errors.Errorf("normalize labels: %w: control characters not allowed", ErrInvalidLabels)
			}
		}
		key := strings.ToLower(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
		if len(out) > MaxLabelsPerNode {
			return nil, errors.Errorf("normalize labels: %w: too many labels", ErrInvalidLabels)
		}
	}

	return out, nil
}

// NodeHasAllLabels reports whether meta contains every required label (case-insensitive).
func NodeHasAllLabels(meta map[string]any, required []string) bool {
	if len(required) == 0 {
		return true
	}
	nodeLabels := LabelsEffective(meta)
	if len(nodeLabels) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(nodeLabels))
	for _, l := range nodeLabels {
		set[strings.ToLower(l)] = struct{}{}
	}
	for _, req := range required {
		req = strings.TrimSpace(req)
		if req == "" {
			continue
		}
		if _, ok := set[strings.ToLower(req)]; !ok {
			return false
		}
	}

	return true
}

func extractStringList(meta map[string]any, key string) []string {
	if meta == nil {
		return nil
	}
	raw, ok := meta[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}

		return out
	default:
		return nil
	}
}

// ListLabelSuggestions returns unique labels used in the knowledge base (sorted).
func (s *Store) ListLabelSuggestions(ctx context.Context, basePath string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 500
	}
	items, _, err := s.ListNodesWithOptions(ctx, basePath, ListNodesOptions{
		Recursive: true,
		Limit:     100000,
		Offset:    0,
		Sort:      "title",
		Order:     "asc",
	})
	if err != nil {
		return nil, errors.Errorf("list label suggestions: %w", err)
	}
	seen := make(map[string]string)
	for _, item := range items {
		for _, label := range item.Labels {
			key := strings.ToLower(label)
			if _, ok := seen[key]; !ok {
				seen[key] = label
			}
		}
	}
	out := make([]string, 0, len(seen))
	for _, label := range seen {
		out = append(out, label)
	}
	sort.Strings(out)
	if len(out) > limit {
		out = out[:limit]
	}

	return out, nil
}
