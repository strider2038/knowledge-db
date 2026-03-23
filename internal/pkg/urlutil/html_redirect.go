package urlutil

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const maxHTMLRedirectBody = 64 * 1024

var (
	reWindowLocation = regexp.MustCompile(`(?i)window\.location(?:\.href)?\s*=\s*["']([^"']+)["']`)
	reMetaRefreshURL = regexp.MustCompile(`(?i)<meta[^>]+http-equiv=["']refresh["'][^>]+content=["'][^;"']*url\s*=\s*([^"';]+)`)
	reAnchorHTTPS    = regexp.MustCompile(`(?i)<a[^>]+href=["'](https?://[^"']+)["']`)
)

// tryExtractRedirectFromHTMLGET loads the page with GET (bounded body) and tries to find an HTTP(S)
// redirect target embedded in HTML (JS window.location, meta refresh, or first external <a href>).
func tryExtractRedirectFromHTMLGET(ctx context.Context, client *http.Client, pageURL string) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", "knowledge-db/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", false
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/xhtml") {
		return "", false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLRedirectBody))
	if err != nil {
		return "", false
	}

	s := string(body)

	pageParsed, err := url.Parse(pageURL)
	if err != nil {
		return "", false
	}
	pageHost := strings.ToLower(strings.TrimPrefix(pageParsed.Host, "www."))

	if m := reWindowLocation.FindStringSubmatch(s); m != nil {
		if t := strings.TrimSpace(m[1]); isHTTPURL(t) {
			return t, true
		}
	}

	if m := reMetaRefreshURL.FindStringSubmatch(s); m != nil {
		if t := strings.TrimSpace(m[1]); isHTTPURL(t) {
			return t, true
		}
	}

	for _, m := range reAnchorHTTPS.FindAllStringSubmatch(s, -1) {
		if len(m) < 2 {
			continue
		}
		t := strings.TrimSpace(m[1])
		if !isHTTPURL(t) {
			continue
		}
		uu, err := url.Parse(t)
		if err != nil {
			continue
		}
		linkHost := strings.ToLower(strings.TrimPrefix(uu.Host, "www."))
		if linkHost != "" && linkHost != pageHost {
			return t, true
		}
	}

	return "", false
}

func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	default:
		return false
	}
}
