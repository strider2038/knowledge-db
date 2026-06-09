package ingestion

import (
	"strings"

	"github.com/strider2038/knowledge-db/internal/kb"
)

// ContentMode describes how ingest should obtain and transform node body.
type ContentMode string

const (
	ContentModeAuto         ContentMode = "auto"
	ContentModeVerbatim     ContentMode = "verbatim"
	ContentModeFullFetch    ContentMode = "full_fetch"
	ContentModeDigest       ContentMode = "digest"
	ContentModeLinkBookmark ContentMode = "link_bookmark"
)

// ParseContentMode validates a request value. Empty string is treated as auto.
func ParseContentMode(raw string) (ContentMode, bool) {
	switch strings.TrimSpace(raw) {
	case "", string(ContentModeAuto):
		return ContentModeAuto, true
	case string(ContentModeVerbatim):
		return ContentModeVerbatim, true
	case string(ContentModeFullFetch):
		return ContentModeFullFetch, true
	case string(ContentModeDigest):
		return ContentModeDigest, true
	case string(ContentModeLinkBookmark):
		return ContentModeLinkBookmark, true
	default:
		return "", false
	}
}

func (m ContentMode) Resolved() bool {
	return m != "" && m != ContentModeAuto
}

// ResolveInput carries fields needed for deterministic mode resolution.
type ResolveInput struct {
	RawContent  string
	SourceURL   string
	TypeHint    string
	ContentMode ContentMode
	Profile     SourceProfile
}

// ResolveContentMode picks the effective content mode before LLM orchestration.
func ResolveContentMode(input ResolveInput) ContentMode {
	if input.ContentMode.Resolved() {
		return input.ContentMode
	}

	text := strings.TrimSpace(input.RawContent)
	if mode, ok := detectIntentMarkers(text); ok {
		return mode
	}

	typeHint := strings.TrimSpace(input.TypeHint)
	if typeHint == nodeTypeArticle {
		if isURLOnlyInput(text) {
			return ContentModeFullFetch
		}
		if hasSubstantialBody(text) {
			return ContentModeVerbatim
		}
	}

	if isTelegramDeliveryURL(input.SourceURL) && looksLikeLongFormMessage("", text) {
		return ContentModeVerbatim
	}

	if isURLOnlyInput(text) {
		if typeHint == nodeTypeArticle {
			return ContentModeFullFetch
		}
		if hasDigestProfile(input.Profile) {
			return ContentModeDigest
		}

		return ContentModeLinkBookmark
	}

	if hasSubstantialBody(text) || looksLikeLongFormMessage("", text) {
		return ContentModeVerbatim
	}

	if strings.TrimSpace(input.SourceURL) != "" || containsHTTPURL(text) {
		if hasDigestProfile(input.Profile) {
			return ContentModeDigest
		}
		if containsHTTPURL(text) || strings.TrimSpace(input.SourceURL) != "" {
			return ContentModeLinkBookmark
		}
	}

	if text != "" {
		return ContentModeVerbatim
	}

	return ContentModeDigest
}

// ResolveRefreshContentMode infers mode from stored node fields.
func ResolveRefreshContentMode(nodeType, contentProfile, sourceURL, body string) ContentMode {
	nodeType = strings.TrimSpace(nodeType)
	contentProfile = strings.TrimSpace(contentProfile)
	sourceURL = strings.TrimSpace(sourceURL)
	body = strings.TrimSpace(body)

	switch {
	case nodeType == nodeTypeArticle && sourceURL != "":
		return ContentModeFullFetch
	case nodeType == nodeTypeLink:
		profile := kb.ContentProfile(contentProfile)
		if profile == "" || profile == kb.ContentProfileLinkBookmark {
			return ContentModeLinkBookmark
		}
		if kb.IsValidContentProfile(contentProfile) {
			return ContentModeDigest
		}

		return ContentModeLinkBookmark
	case nodeType == nodeTypeNote && (contentProfile == string(kb.ContentProfileConceptualDigest) || contentProfile == string(kb.ContentProfileBriefDigest)) && sourceURL != "":
		return ContentModeDigest
	default:
		if body == "" && sourceURL != "" {
			if nodeType == nodeTypeArticle {
				return ContentModeFullFetch
			}
			if hasDigestProfile(SourceProfile{ContentProfile: kb.ContentProfile(contentProfile)}) {
				return ContentModeDigest
			}

			return ContentModeLinkBookmark
		}

		return ContentModeVerbatim
	}
}

func detectIntentMarkers(text string) (ContentMode, bool) {
	lower := strings.ToLower(text)
	for _, marker := range []struct {
		substr string
		mode   ContentMode
	}{
		{"сохрани полную статью", ContentModeFullFetch},
		{"полная статья", ContentModeFullFetch},
		{"full article", ContentModeFullFetch},
		{"full fetch", ContentModeFullFetch},
		{"выжимка", ContentModeDigest},
		{"концептуально", ContentModeDigest},
		{"digest", ContentModeDigest},
		{"как есть", ContentModeVerbatim},
		{"без изменений", ContentModeVerbatim},
		{"verbatim", ContentModeVerbatim},
	} {
		if strings.Contains(lower, marker.substr) {
			return marker.mode, true
		}
	}

	return "", false
}

func isURLOnlyInput(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || !containsHTTPURL(text) {
		return false
	}
	withoutURLs := strings.TrimSpace(stripURLs(text))

	return withoutURLs == "" || len([]rune(withoutURLs)) < 40
}

func hasSubstantialBody(text string) bool {
	withoutURLs := strings.TrimSpace(stripURLs(text))
	if len([]rune(withoutURLs)) >= 500 {
		return true
	}

	return len(strings.Fields(withoutURLs)) >= 80
}

func stripURLs(text string) string {
	var parts []string
	for field := range strings.FieldsSeq(text) {
		trimmed := strings.Trim(field, ".,;:!?)>]\"'")
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			continue
		}
		parts = append(parts, field)
	}

	return strings.Join(parts, " ")
}

func containsHTTPURL(text string) bool {
	lower := strings.ToLower(text)

	return strings.Contains(lower, "http://") || strings.Contains(lower, "https://")
}

func isTelegramDeliveryURL(rawURL string) bool {
	rawURL = strings.TrimSpace(strings.ToLower(rawURL))

	return strings.HasPrefix(rawURL, "https://t.me/") || strings.HasPrefix(rawURL, "http://t.me/")
}

func hasDigestProfile(profile SourceProfile) bool {
	if !profile.HasProfile() {
		return false
	}

	switch profile.ContentProfile {
	case kb.ContentProfileLinkBookmark:
		return false
	default:
		return kb.IsValidContentProfile(string(profile.ContentProfile))
	}
}
