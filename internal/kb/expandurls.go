package kb

import (
	"context"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/strider2038/knowledge-db/internal/pkg/urlutil"
)

var (
	reFrontmatterBlock = regexp.MustCompile(`(?s)^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$`)
	// Markdown links and images: [text](url) and ![alt](url).
	reMarkdownHTTP = regexp.MustCompile(`\[[^\]]*\]\((https?://[^)]+)\)`)
	// Autolink <https://...>
	reAngleHTTP = regexp.MustCompile(`<(https?://[^>]+)>`)
	// Single-line frontmatter: key: "url" or key: url.
	reFMLineHTTP = regexp.MustCompile(`(?m)^\s*[\w._-]+\s*:\s*["']?(https?://[^\s"']+)["']?\s*$`)
)

// ExpandURLsResult describes expand-urls run on one file.
type ExpandURLsResult struct {
	// Pairs lists each unique old→new where New differs from Old (for dry-run output).
	Pairs []ExpandURLPair
	// Replacements is the number of non-overlapping substring replacements applied in the file.
	Replacements int
	// FailedURLs lists source URLs where HEAD normalization failed (network/timeout).
	FailedURLs []string
	// Changed is true if file content differs from input.
	Changed bool
}

// ExpandURLPair is one normalized URL mapping.
type ExpandURLPair struct {
	Old, New string
}

// ExpandURLsInString returns a copy of s with http(s) URLs normalized via urlutil.TryNormalizeURL.
func ExpandURLsInString(ctx context.Context, s string) (string, ExpandURLsResult) {
	var res ExpandURLsResult
	seen := map[string]string{}
	var order []string

	for _, u := range collectHTTPURLs(s) {
		if _, ok := seen[u]; ok {
			continue
		}
		newU, ok := urlutil.TryNormalizeURL(ctx, u)
		if !ok {
			res.FailedURLs = append(res.FailedURLs, u)
		}
		seen[u] = newU
		order = append(order, u)
	}

	for _, old := range order {
		newU := seen[old]
		if newU == old {
			continue
		}
		res.Pairs = append(res.Pairs, ExpandURLPair{Old: old, New: newU})
	}

	if len(res.Pairs) == 0 {
		return s, res
	}

	out := s

	keys := make([]string, 0, len(res.Pairs))
	for _, p := range res.Pairs {
		keys = append(keys, p.Old)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })

	repl := make(map[string]string, len(res.Pairs))
	for _, p := range res.Pairs {
		repl[p.Old] = p.New
	}

	for _, k := range keys {
		n := strings.Count(out, k)
		if n == 0 {
			continue
		}
		res.Replacements += n
		out = strings.ReplaceAll(out, k, repl[k])
	}

	res.Changed = out != s

	return out, res
}

func collectHTTPURLs(full string) []string {
	var fm, body string
	if m := reFrontmatterBlock.FindStringSubmatch(full); m != nil {
		fm, body = m[1], m[2]
	} else {
		body = full
	}

	seen := map[string]struct{}{}
	var urls []string
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		urls = append(urls, u)
	}

	for _, m := range reFMLineHTTP.FindAllStringSubmatch(fm, -1) {
		if len(m) >= 2 {
			add(m[1])
		}
	}
	for _, re := range []*regexp.Regexp{reMarkdownHTTP, reAngleHTTP} {
		for _, m := range re.FindAllStringSubmatch(body, -1) {
			if len(m) >= 2 {
				add(m[1])
			}
		}
	}

	return urls
}

// WriteExpandURLsFile reads path, expands URLs, and writes back unless dryRun.
func WriteExpandURLsFile(ctx context.Context, path string, dryRun bool) (ExpandURLsResult, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ExpandURLsResult{}, err
	}
	st, statErr := os.Stat(path)
	var perm os.FileMode = 0o644
	if statErr == nil {
		perm = st.Mode().Perm()
	}

	newS, res := ExpandURLsInString(ctx, string(b))
	if dryRun || !res.Changed {
		return res, nil
	}

	if err := os.WriteFile(path, []byte(newS), perm); err != nil {
		return ExpandURLsResult{}, err
	}

	return res, nil
}
