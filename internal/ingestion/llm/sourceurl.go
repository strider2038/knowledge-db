package llm

import (
	"context"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/strider2038/knowledge-db/internal/pkg/urlutil"
)

var reHTTPInText = regexp.MustCompile(`https?://[^\s\)\]>'"]+`)

// pickResourceURLFromMessageText извлекает из полного текста пользователя (в т.ч. с префиксом
// «Telegram-канал…») HTTP(S)-URL ресурса, исключая URL доставки через Telegram и низкоспецифичные
// подстановки вроде docs.github.com, если в том же тексте есть владелец/репозиторий на github.com.
func pickResourceURLFromMessageText(_ context.Context, text string) string {
	raws := reHTTPInText.FindAllString(text, -1)
	if len(raws) == 0 {
		return ""
	}

	seen := make(map[string]struct{})
	var candidates []string
	for _, raw := range raws {
		u := trimURLTrailingJunk(raw)
		if u == "" {
			continue
		}
		if isTelegramDeliveryURL(u) {
			continue
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		key := strings.ToLower(parsed.String())
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		candidates = append(candidates, u)
	}

	if len(candidates) == 0 {
		return ""
	}
	if len(candidates) == 1 {
		return normalizeURLOrEmpty(candidates[0])
	}

	return normalizeURLOrEmpty(pickBestResourceURL(candidates))
}

func trimURLTrailingJunk(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ".,;:!?\"'")

	return s
}

func pickBestResourceURL(candidates []string) string {
	var hasGitHubRepo bool
	if slices.ContainsFunc(candidates, isGitHubRepoURL) {
		hasGitHubRepo = true
	}
	if hasGitHubRepo {
		for _, c := range candidates {
			if isGitHubRepoURL(c) {
				return c
			}
		}
	}

	var withoutDocs []string
	for _, c := range candidates {
		if isDocsGitHubHost(c) {
			continue
		}
		withoutDocs = append(withoutDocs, c)
	}
	if len(withoutDocs) == 1 {
		return withoutDocs[0]
	}
	if len(withoutDocs) > 0 {
		return withoutDocs[0]
	}

	return candidates[0]
}

func isGitHubRepoURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimPrefix(parsed.Host, "www."))
	if host != "github.com" {
		return false
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")

	return len(parts) >= 2 && parts[0] != "" && parts[1] != ""
}

func isDocsGitHubHost(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}

	return strings.EqualFold(parsed.Host, "docs.github.com")
}

func normalizeURLOrEmpty(raw string) string {
	// Без HTTP HEAD: NormalizeURL может следовать редиректу github.com → docs.github.com
	// и испортить URL, явно указанный пользователем в сообщении.
	return urlutil.StripTrackingParamsFromURL(strings.TrimSpace(raw))
}
