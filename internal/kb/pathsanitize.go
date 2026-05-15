package kb

import (
	"net/url"
	"path"
	"strings"
)

var cyrillicToLatin = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i",
	'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
	'у': "u", 'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch", 'ъ': "", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "yu", 'я': "ya",
}

// SanitizePathSegment normalizes a single node path segment to a strict ASCII slug.
// Allowed chars: [a-z0-9-].
func SanitizePathSegment(raw string) string {
	decoded := strings.TrimSpace(raw)
	if decoded == "" {
		return ""
	}
	if unescaped, err := url.PathUnescape(decoded); err == nil {
		decoded = unescaped
	}
	decoded = strings.ToLower(decoded)

	var b strings.Builder
	b.Grow(len(decoded))
	prevDash := false

	for _, r := range decoded {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			prevDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.' || r == '/':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			if t, ok := cyrillicToLatin[r]; ok {
				for _, tr := range t {
					if tr >= 'a' && tr <= 'z' {
						b.WriteRune(tr)
						prevDash = false
					}
				}

				continue
			}
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "node"
	}

	return out
}

// SanitizeNodePath normalizes relative node path to strict ASCII path.
// Each segment becomes [a-z0-9-], path separators are '/'.
func SanitizeNodePath(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return ""
	}

	parts := strings.Split(trimmed, "/")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		seg := SanitizePathSegment(p)
		if seg == "" {
			continue
		}
		clean = append(clean, seg)
	}
	if len(clean) == 0 {
		return ""
	}

	return path.Clean(strings.Join(clean, "/"))
}
