package translation

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	cyrillicThreshold = 0.25
	minContentLength  = 200
)

var codeBlockRe = regexp.MustCompile("(?s)```[^`]*```")

// NeedsTranslation определяет, нужен ли перевод контента на русский.
// Эвристика: удаление code blocks, подсчёт доли кириллицы.
// Если cyrillic_ratio < 0.25 и длина >= 200 символов — нужен перевод.
func NeedsTranslation(content string) bool {
	text := codeBlockRe.ReplaceAllString(content, "")
	text = strings.TrimSpace(text)
	if len(text) < minContentLength {
		return false
	}

	var cyrillic, latin int
	for _, r := range text {
		if unicode.Is(unicode.Cyrillic, r) {
			cyrillic++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			latin++
		}
	}

	total := cyrillic + latin
	if total == 0 {
		return false
	}

	ratio := float64(cyrillic) / float64(total)

	return ratio < cyrillicThreshold
}
