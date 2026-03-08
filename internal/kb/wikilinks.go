package kb

import "regexp"

var wikilinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]*)?\]\]`)

// ParseWikilinks извлекает все wikilinks из markdown: [[target]] и [[target|label]].
// Возвращает слайс target (без label).
func ParseWikilinks(content string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(content, -1)
	result := make([]string, 0, len(matches))
	seen := make(map[string]struct{})

	for _, m := range matches {
		if len(m) >= 2 && m[1] != "" {
			target := m[1]
			if _, ok := seen[target]; !ok {
				seen[target] = struct{}{}
				result = append(result, target)
			}
		}
	}

	return result
}
