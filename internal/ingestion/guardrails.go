package ingestion

import (
	"context"
	"strings"
	"unicode"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func (p *PipelineIngester) applyContentModeGuardrails(
	ctx context.Context,
	mode ContentMode,
	rawContent string,
	processInput llm.ProcessInput,
	result *llm.ProcessResult,
) error {
	if result == nil {
		return nil
	}

	switch mode {
	case ContentModeVerbatim:
		body := extractVerbatimBody(rawContent)
		if body != "" {
			result.Content = body
		}
	case ContentModeFullFetch:
		if err := p.ensureFullFetchContent(ctx, result); err != nil {
			return err
		}
	case ContentModeDigest, ContentModeLinkBookmark:
		if err := p.ensureModeContent(ctx, mode, result, processInput); err != nil {
			return err
		}
	}

	normalizeResultTitle(result)

	return nil
}

func (p *PipelineIngester) ensureFullFetchContent(ctx context.Context, result *llm.ProcessResult) error {
	if result == nil || result.Type != nodeTypeArticle {
		return nil
	}
	if strings.TrimSpace(result.SourceURL) == "" {
		return nil
	}
	if strings.TrimSpace(result.Content) != "" {
		return nil
	}

	if p.contentFetcher == nil {
		return errors.Errorf("%w: fetch unavailable", ErrArticleContentEmpty)
	}

	fetched, err := p.contentFetcher.Fetch(ctx, result.SourceURL)
	if err != nil || fetched == nil || strings.TrimSpace(fetched.Content) == "" {
		return errors.Errorf("%w: %v", ErrArticleContentEmpty, err)
	}

	clog.Info(ctx, "ingest: full_fetch content fetched", "url", result.SourceURL, "content_len", len(fetched.Content))
	result.Content = fetched.Content
	if result.Title == "" && fetched.Title != "" {
		result.Title = fetched.Title
	}
	if result.SourceAuthor == "" && fetched.Author != "" {
		result.SourceAuthor = fetched.Author
	}
	if result.SourceDate == nil && fetched.SourceDate != nil {
		result.SourceDate = fetched.SourceDate
	}

	return nil
}

func (p *PipelineIngester) ensureModeContent(
	ctx context.Context,
	mode ContentMode,
	result *llm.ProcessResult,
	processInput llm.ProcessInput,
) error {
	if strings.TrimSpace(result.Content) != "" {
		return nil
	}

	retryInput := processInput
	switch mode {
	case ContentModeDigest:
		retryInput.Text += "\n\nКритично: поле content обязательно и должно содержать структурированный markdown digest по content_profile. Пустой content недопустим."
	case ContentModeLinkBookmark:
		retryInput.Text += "\n\nКритично: для link_bookmark создай короткое semantic body из доступных фактов (title, URL, metadata, preview). Пустой content недопустим."
	default:
		return nil
	}

	retried, err := p.orchestrator.Process(ctx, retryInput)
	if err != nil {
		return errors.Errorf("%w: retry failed: %v", ErrDigestContentEmpty, err)
	}
	if retried != nil && strings.TrimSpace(retried.Content) != "" {
		*result = *retried

		return nil
	}

	return ErrDigestContentEmpty
}

func extractVerbatimBody(rawContent string) string {
	return strings.TrimSpace(rawContent)
}

func normalizeResultTitle(result *llm.ProcessResult) {
	if result == nil {
		return
	}
	if result.Title != "" {
		result.Title = normalizeTitle(result.Title)
	}
}

// normalizeTitle strips markdown noise and moves leading emoji to the end.
func normalizeTitle(title string) string {
	title = stripMarkdownFromTitle(title)
	title = stripMarkdownLinkFromTitle(title)
	title = normalizeTitleDecorators(title)
	title = strings.TrimSpace(title)
	title = strings.TrimRight(title, ".,;:!?")

	return strings.TrimSpace(title)
}

func stripMarkdownLinkFromTitle(title string) string {
	for {
		start := strings.Index(title, "[")
		if start < 0 {
			break
		}
		mid := strings.Index(title[start:], "](")
		if mid < 0 {
			break
		}
		mid += start
		end := strings.Index(title[mid+2:], ")")
		if end < 0 {
			break
		}
		end += mid + 2
		inner := title[start+1 : mid]
		title = strings.TrimSpace(title[:start] + inner + title[end+1:])
	}

	return strings.TrimSpace(title)
}

func normalizeTitleDecorators(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return title
	}

	runes := []rune(title)
	start := 0
	for start < len(runes) {
		r := runes[start]
		if unicode.IsSpace(r) {
			start++

			continue
		}
		if isLeadingDecorator(r) {
			start++

			continue
		}

		break
	}
	if start == 0 {
		return title
	}

	leading := strings.TrimSpace(string(runes[:start]))
	rest := strings.TrimSpace(string(runes[start:]))
	if rest == "" {
		return leading
	}
	if leading == "" {
		return rest
	}

	return rest + " " + leading
}

func isLeadingDecorator(r rune) bool {
	if r == '#' || r == '*' || r == '_' || r == '`' {
		return true
	}

	return isEmojiLike(r)
}

func isEmojiLike(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1FAFF:
		return true
	case r >= 0x2600 && r <= 0x27BF:
		return true
	default:
		return false
	}
}

func requiresModeContent(mode ContentMode, result *llm.ProcessResult) bool {
	if result == nil {
		return false
	}
	switch mode {
	case ContentModeDigest:
		return result.Type == "link" || result.Type == "note"
	case ContentModeLinkBookmark:
		return result.Type == "link"
	default:
		return false
	}
}

func applyModeToFrontmatterTitle(frontmatter map[string]any) {
	title, ok := frontmatter["title"].(string)
	if !ok || title == "" {
		return
	}
	normalized := normalizeTitle(title)
	frontmatter["title"] = normalized
	if aliases, ok := frontmatter["aliases"].([]string); ok && len(aliases) == 1 {
		frontmatter["aliases"] = []string{normalized}
	} else if aliasesAny, ok := frontmatter["aliases"].([]any); ok && len(aliasesAny) == 1 {
		if s, ok := aliasesAny[0].(string); ok {
			frontmatter["aliases"] = []string{normalizeTitle(s)}
		}
	}
}

func profileDigestMode(profile kb.ContentProfile) bool {
	if profile == "" || profile == kb.ContentProfileLinkBookmark {
		return false
	}

	return kb.IsValidContentProfile(string(profile))
}
