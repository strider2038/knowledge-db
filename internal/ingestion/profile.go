package ingestion

import (
	"net/url"
	"strings"

	"github.com/strider2038/knowledge-db/internal/kb"
)

type SourceProfile struct {
	SourceKind      kb.SourceKind
	ContentProfile  kb.ContentProfile
	RecommendedType string
}

func (p SourceProfile) HasProfile() bool {
	return p.SourceKind != "" && p.ContentProfile != "" && p.RecommendedType != ""
}

func ClassifySource(rawURL, title, content, typeHint string) SourceProfile {
	rawURL = strings.TrimSpace(rawURL)
	if typeHint == "article" {
		return SourceProfile{SourceKind: kb.SourceKindArticle, RecommendedType: "article"}
	}
	if rawURL == "" {
		return SourceProfile{}
	}

	host, path := parseSourceURL(rawURL)
	text := strings.ToLower(title + " " + content + " " + host + " " + path)

	if isRepositoryURL(host, path) {
		return SourceProfile{
			SourceKind:      kb.SourceKindRepository,
			ContentProfile:  kb.ContentProfileRepository,
			RecommendedType: "link",
		}
	}
	if isSocialHost(host) {
		return SourceProfile{
			SourceKind:      kb.SourceKindSocialPost,
			ContentProfile:  kb.ContentProfileBriefDigest,
			RecommendedType: "note",
		}
	}
	if looksLikeNews(host, path, text) {
		return SourceProfile{
			SourceKind:      kb.SourceKindNews,
			ContentProfile:  kb.ContentProfileBriefDigest,
			RecommendedType: "note",
		}
	}
	if looksLikeDocumentation(host, path, text) {
		return SourceProfile{
			SourceKind:      kb.SourceKindDocumentation,
			ContentProfile:  kb.ContentProfileDocumentation,
			RecommendedType: "link",
		}
	}
	if looksLikeDirectory(host, path, text) {
		return SourceProfile{
			SourceKind:      kb.SourceKindDirectoryCatalog,
			ContentProfile:  kb.ContentProfileDirectory,
			RecommendedType: "link",
		}
	}
	if looksLikeLearningResource(host, path, text) {
		return SourceProfile{
			SourceKind:      kb.SourceKindLearningResource,
			ContentProfile:  kb.ContentProfileLearningResource,
			RecommendedType: "link",
		}
	}
	if looksLikeOnlineTool(host, path, text) {
		return SourceProfile{
			SourceKind:      kb.SourceKindOnlineTool,
			ContentProfile:  kb.ContentProfileOnlineTool,
			RecommendedType: "link",
		}
	}
	if looksLikeProduct(host, path, text) {
		return SourceProfile{
			SourceKind:      kb.SourceKindProductService,
			ContentProfile:  kb.ContentProfileProduct,
			RecommendedType: "link",
		}
	}
	if looksLikeArticle(path, text) {
		return SourceProfile{
			SourceKind:      kb.SourceKindArticle,
			ContentProfile:  kb.ContentProfileConceptualDigest,
			RecommendedType: "note",
		}
	}
	if looksLikeLongFormMessage(title, content) {
		return SourceProfile{
			SourceKind:      kb.SourceKindArticle,
			ContentProfile:  kb.ContentProfileConceptualDigest,
			RecommendedType: "note",
		}
	}

	return SourceProfile{
		SourceKind:      kb.SourceKindUnknown,
		ContentProfile:  kb.ContentProfileLinkBookmark,
		RecommendedType: "link",
	}
}

func parseSourceURL(rawURL string) (string, string) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", strings.ToLower(rawURL)
	}

	return strings.ToLower(parsed.Hostname()), strings.ToLower(parsed.EscapedPath())
}

func isRepositoryURL(host, path string) bool {
	if host == "github.com" || host == "gitlab.com" || host == "codeberg.org" {
		parts := strings.Split(strings.Trim(path, "/"), "/")

		return len(parts) >= 2 && parts[0] != "" && parts[1] != ""
	}

	return false
}

func isSocialHost(host string) bool {
	return host == "x.com" || host == "twitter.com" || host == "bsky.app" ||
		host == "mastodon.social" || strings.HasSuffix(host, ".social")
}

func looksLikeNews(host, path, text string) bool {
	if strings.Contains(path, "/news") || strings.Contains(path, "/release") {
		return true
	}
	for _, marker := range []string{"techcrunch.com", "theverge.com", "wired.com", "venturebeat.com", "reuters.com"} {
		if host == marker || strings.HasSuffix(host, "."+marker) {
			return true
		}
	}

	return containsAny(text, "релиз", "анонс", "launches", "released", "breaking news", "press release")
}

func looksLikeDocumentation(host, path, text string) bool {
	return strings.HasPrefix(host, "docs.") ||
		strings.Contains(host, "readthedocs") ||
		strings.Contains(path, "/docs") ||
		strings.Contains(path, "/documentation") ||
		containsAny(text, "documentation", "api reference", "руководство", "документация")
}

func looksLikeDirectory(host, path, text string) bool {
	return containsAny(host+" "+path+" "+text, "awesome-", "directory", "catalog", "catalogue", "curated list", "каталог", "подборка")
}

func looksLikeLearningResource(host, path, text string) bool {
	return containsAny(host+" "+path+" "+text, "course", "tutorial", "learn", "guide", "workshop", "курс", "урок", "туториал")
}

func looksLikeOnlineTool(host, path, text string) bool {
	return containsAny(host+" "+path+" "+text, "playground", "calculator", "converter", "generator", "validator", "tool", "sandbox", "инструмент")
}

func looksLikeProduct(host, path, text string) bool {
	return containsAny(host+" "+path+" "+text, "pricing", "product", "platform", "service", "saas", "интеграции", "workflow")
}

func looksLikeArticle(path, text string) bool {
	if strings.Contains(path, "/blog") || strings.Contains(path, "/article") || strings.Contains(path, "/posts") {
		return true
	}

	return strings.Count(text, " ") > 300 || containsAny(text, "article", "essay", "blog post", "статья")
}

func looksLikeLongFormMessage(title, content string) bool {
	combined := strings.TrimSpace(title + "\n" + content)
	if combined == "" {
		return false
	}
	// Telegram-forwarded and similar long posts should be treated as notes/digests,
	// even if they include one or more URLs in the body.
	words := len(strings.Fields(combined))
	if words >= 120 {
		return true
	}

	return strings.Count(combined, "\n\n") >= 4 && words >= 80
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}

	return false
}
