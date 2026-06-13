package kb

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	markdownCodeBlockRe = regexp.MustCompile("(?s)```.*?```")
	markdownHeadingRe   = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	markdownInlineRe    = []*regexp.Regexp{
		regexp.MustCompile(`\*\*(.+?)\*\*`),
		regexp.MustCompile(`\*(.+?)\*`),
		regexp.MustCompile(`__(.+?)__`),
		regexp.MustCompile(`_(.+?)_`),
		regexp.MustCompile("`(.+?)`"),
		regexp.MustCompile(`\[(.+?)\]\([^)]+\)`),
	}
)

func markdownPlainText(content string) string {
	content = markdownCodeBlockRe.ReplaceAllString(content, " ")
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if m := markdownHeadingRe.FindStringSubmatch(line); m != nil {
			out = append(out, stripInlineMarkdown(m[2]))
			continue
		}
		out = append(out, stripInlineMarkdown(line))
	}

	return collapseWhitespace(strings.Join(out, "\n"))
}

func stripInlineMarkdown(text string) string {
	for _, re := range markdownInlineRe {
		text = re.ReplaceAllString(text, "$1")
	}

	return strings.TrimSpace(text)
}

func markdownSectionForHeading(content, headingID string) string {
	headingID = strings.TrimSpace(headingID)
	if headingID == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	start := -1
	startLevel := 0
	slugger := newHeadingSlugger()
	for i, line := range lines {
		m := markdownHeadingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		level := len(m[1])
		slug := slugger.slug(stripInlineMarkdown(m[2]))
		if slug == headingID {
			start = i + 1
			startLevel = level
			break
		}
	}
	if start < 0 {
		return content
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		m := markdownHeadingRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		if len(m[1]) <= startLevel {
			end = i
			break
		}
	}

	return strings.Join(lines[start:end], "\n")
}

type headingSlugger struct {
	seen map[string]int
}

func newHeadingSlugger() *headingSlugger {
	return &headingSlugger{seen: map[string]int{}}
}

func (s *headingSlugger) slug(text string) string {
	base := slugifyHeading(text)
	if base == "" {
		return ""
	}
	count := s.seen[base]
	s.seen[base] = count + 1
	if count == 0 {
		return base
	}

	return base + "-" + strconv.Itoa(count)
}

func slugifyHeading(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	lastDash := false
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")

	return out
}

func resolveTextQuote(content string, anchor *AnnotationAnchor) bool {
	if anchor == nil || anchor.Exact == "" {
		return false
	}
	section := markdownSectionForHeading(content, anchor.HeadingID)
	plain := markdownPlainText(section)
	candidates := []string{plain, collapseWhitespace(plain)}
	for _, text := range candidates {
		if resolveInText(text, anchor) {
			return true
		}
	}

	return false
}

func resolveInText(text string, anchor *AnnotationAnchor) bool {
	exact := anchor.Exact
	if !strings.Contains(text, exact) {
		return false
	}
	if anchor.Prefix == "" && anchor.Suffix == "" {
		return true
	}
	start := 0
	for {
		idx := strings.Index(text[start:], exact)
		if idx < 0 {
			return false
		}
		pos := start + idx
		if contextMatchesRunes(text, pos, len([]rune(exact)), anchor.Prefix, anchor.Suffix) {
			return true
		}
		start = pos + 1
	}
}

func contextMatchesRunes(text string, pos, exactRunes int, prefix, suffix string) bool {
	runes := []rune(text)
	if prefix != "" {
		prefixRunes := []rune(prefix)
		beforeStart := pos - len(prefixRunes)
		if beforeStart < 0 {
			return false
		}
		if string(runes[beforeStart:pos]) != prefix {
			return false
		}
	}
	if suffix != "" {
		suffixRunes := []rune(suffix)
		afterStart := pos + exactRunes
		if afterStart+len(suffixRunes) > len(runes) {
			return false
		}
		if string(runes[afterStart:afterStart+len(suffixRunes)]) != suffix {
			return false
		}
	}

	return true
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}

			continue
		}
		space = false
		b.WriteRune(r)
	}

	return b.String()
}

func remapAnnotationContentPath(contentPath, oldBase, newBase string) (string, bool) {
	contentPath = filepathToSlash(contentPath)
	oldBase = filepathToSlash(oldBase)
	newBase = filepathToSlash(newBase)
	if contentPath == oldBase {
		return newBase, true
	}
	prefix := oldBase + "."
	if strings.HasPrefix(contentPath, prefix) {
		suffix := contentPath[len(prefix):]
		if strings.Contains(suffix, "/") {
			return contentPath, false
		}

		return newBase + "." + suffix, true
	}

	return contentPath, false
}

func filepathToSlash(path string) string {
	return strings.TrimSuffix(strings.ReplaceAll(path, "\\", "/"), "/")
}
