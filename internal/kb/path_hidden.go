package kb

import (
	"path/filepath"
	"strings"
)

// pathHasHiddenSegment reports whether rel (relative to base) contains a path
// component whose name starts with "." (e.g. .cursor, .git). Segments "." and ".."
// are not treated as hidden; rel "." or "" is not hidden.
func pathHasHiddenSegment(rel string) bool {
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" {
		return false
	}
	for seg := range strings.SplitSeq(rel, "/") {
		if seg == "" || seg == "." || seg == ".." {
			continue
		}
		if strings.HasPrefix(seg, ".") {
			return true
		}
	}

	return false
}
