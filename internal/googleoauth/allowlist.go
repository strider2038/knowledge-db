package googleoauth

import "strings"

// ParseEmailAllowlist builds a set from a comma-separated list.
func ParseEmailAllowlist(s string) map[string]struct{} {
	out := make(map[string]struct{})
	for p := range strings.SplitSeq(s, ",") {
		e := strings.TrimSpace(strings.ToLower(p))
		if e != "" {
			out[e] = struct{}{}
		}
	}

	return out
}

// IsEmailAllowed reports whether email (case-insensitive) is in allowlist.
func IsEmailAllowed(allowlist map[string]struct{}, email string) bool {
	e := strings.TrimSpace(strings.ToLower(email))
	_, ok := allowlist[e]

	return ok
}
