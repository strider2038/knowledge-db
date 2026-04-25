package googleoauth

import (
	"path"
	"strings"

	"github.com/muonsoft/errors"
)

// SanitizeReturnPath normalizes a relative in-app path from a query param (open redirect hardening).
func SanitizeReturnPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		return "/"
	}
	if strings.HasPrefix(p, "//") {
		return "/"
	}
	if strings.Contains(p, "://") {
		return "/"
	}
	cleaned := path.Clean(p)
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	if !strings.HasPrefix(cleaned, "/") {
		return "/"
	}

	return cleaned
}

// AppendQueryPath appends a relative path and query to baseURL.
func AppendQueryPath(baseURL, relPath, query string) (string, error) {
	b := strings.TrimRight(baseURL, "/")
	if b == "" {
		return "", errors.New("empty base url")
	}
	rel := relPath
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	if query != "" {
		rel = rel + "?" + query
	}
	if strings.HasPrefix(b, "http://") || strings.HasPrefix(b, "https://") {
		return b + rel, nil
	}

	return b + rel, nil
}
