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

// trackingQueryKeys — распространённые трекинг-параметры (не utm_*), которые снимаем вместе с UTM.
var trackingQueryKeys = map[string]bool{
	"fbclid":   true,
	"gclid":    true,
	"mc_cid":   true,
	"igshid":   true,
	"mkt_tok":  true,
	"msclkid":  true,
	"dclid":    true,
	"yclid":    true,
	"wickedid": true,
}

// TryNormalizeURL разрешает редиректы (короткие ссылки) и удаляет UTM/трекинг query.
// ok=false, если требовался HTTP HEAD и запрос завершился ошибкой (сеть, таймаут);
// в этом случае возвращается stripTrackingParams(rawURL).
func TryNormalizeURL(ctx context.Context, rawURL string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", true
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, true
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return rawURL, true
	}

	host := strings.ToLower(strings.TrimPrefix(parsed.Host, "www."))
	if skipRedirectHosts[host] {
		return stripTrackingParams(rawURL), true
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return stripTrackingParams(rawURL), false
	}
	req.Header.Set("User-Agent", "knowledge-db/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return stripTrackingParams(rawURL), false
	}
	defer func() { _ = resp.Body.Close() }()

	finalURL := resp.Request.URL.String()
	strippedFinal := stripTrackingParams(finalURL)
	strippedRaw := stripTrackingParams(rawURL)

	if strippedFinal != strippedRaw {
		return strippedFinal, true
	}

	// Нет HTTP-редиректа (URL тот же). Частый случай: HTML-страница с window.location / meta / <a>.
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml") {
		if target, ok := tryExtractRedirectFromHTMLGET(ctx, client, rawURL); ok {
			return stripTrackingParams(target), true
		}
	}

	return strippedFinal, true
}

// NormalizeURL разрешает короткие ссылки (редиректы) и удаляет UTM-параметры.
// При ошибке сети/таймауте возвращает исходный URL без ошибки (не ломает ingestion).
func NormalizeURL(ctx context.Context, rawURL string) (string, error) {
	out, _ := TryNormalizeURL(ctx, rawURL)

	return out, nil
}

// StripTrackingParamsFromURL удаляет utm_* и распространённые трекинг-параметры из query
// без HTTP-запросов. Используй для URL, извлечённых из текста (например, Telegram): иначе
// HEAD по github.com может вернуть редирект на docs.github.com и подменить явную ссылку на репозиторий.
func StripTrackingParamsFromURL(rawURL string) string {
	return stripTrackingParams(rawURL)
}

// stripTrackingParams удаляет query-параметры utm_* и распространённые трекинг-ключи.
func stripTrackingParams(rawURL string) string {
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
		lk := strings.ToLower(key)
		if strings.HasPrefix(lk, "utm_") || trackingQueryKeys[lk] {
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
