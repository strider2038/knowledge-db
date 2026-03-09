package urlutil

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// skipRedirectHosts — хосты, для которых не выполняем разрешение редиректов
// (канал доставки, не сам ресурс).
var skipRedirectHosts = map[string]bool{
	"t.me":         true,
	"telegram.org": true,
}

// NormalizeURL разрешает короткие ссылки (редиректы) и удаляет UTM-параметры.
// При ошибке сети/таймауте возвращает исходный URL без ошибки (не ломает ingestion).
func NormalizeURL(ctx context.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, nil
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return rawURL, nil
	}

	host := strings.ToLower(strings.TrimPrefix(parsed.Host, "www."))
	if skipRedirectHosts[host] {
		return stripUTMParams(rawURL), nil
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return rawURL, nil
	}
	req.Header.Set("User-Agent", "knowledge-db/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return rawURL, nil
	}
	defer func() { _ = resp.Body.Close() }()

	finalURL := resp.Request.URL.String()

	return stripUTMParams(finalURL), nil
}

// stripUTMParams удаляет query-параметры, начинающиеся с utm_.
func stripUTMParams(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if len(parsed.RawQuery) == 0 {
		return rawURL
	}

	q := parsed.Query()
	changed := false
	for key := range q {
		if strings.HasPrefix(strings.ToLower(key), "utm_") {
			q.Del(key)
			changed = true
		}
	}
	if !changed {
		return rawURL
	}

	parsed.RawQuery = q.Encode()

	return parsed.String()
}
