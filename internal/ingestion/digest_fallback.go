package ingestion

import (
	"fmt"
	"strings"

	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
)

func buildFallbackModeContent(mode ContentMode, result *llm.ProcessResult, rawContent string) string {
	if result == nil {
		return ""
	}

	title := strings.TrimSpace(result.Title)
	annotation := strings.TrimSpace(result.Annotation)
	sourceURL := strings.TrimSpace(result.SourceURL)

	switch mode {
	case ContentModeLinkBookmark:
		return buildLinkBookmarkFallback(title, annotation, sourceURL)
	case ContentModeDigest:
		return buildDigestFallback(result.ContentProfile, title, annotation, sourceURL, rawContent)
	default:
		return ""
	}
}

func buildLinkBookmarkFallback(title, annotation, sourceURL string) string {
	var parts []string
	if title != "" {
		parts = append(parts, title)
	}
	if sourceURL != "" {
		if len(parts) > 0 {
			parts[len(parts)-1] = parts[len(parts)-1] + " — " + sourceURL
		} else {
			parts = append(parts, sourceURL)
		}
	}
	if annotation != "" {
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, annotation)
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func buildDigestFallback(profile, title, annotation, sourceURL, rawContent string) string {
	if annotation == "" && title == "" && sourceURL == "" {
		return ""
	}

	sections := digestFallbackSections(profile)
	var parts []string
	for _, section := range sections {
		body := digestFallbackSectionBody(section, title, annotation, sourceURL, rawContent)
		if body == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("## %s\n\n%s", section, body))
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func digestFallbackSections(profile string) []string {
	switch profile {
	case "repository_profile":
		return []string{"Назначение", "Ключевая идея", "Источник"}
	case "product_profile":
		return []string{"Назначение", "Основной workflow", "Источник"}
	case "documentation_profile":
		return []string{"Ментальная модель", "Типовые сценарии", "Источник"}
	case "online_tool_profile":
		return []string{"Решаемая задача", "Ограничения", "Источник"}
	case "directory_profile":
		return []string{"Область каталога", "Принцип организации", "Источник"}
	case "learning_resource_profile":
		return []string{"Чему учит", "Практический результат", "Источник"}
	case "brief_digest":
		return []string{"Суть", "Ограничения информации", "Источник"}
	default:
		return []string{"Главная идея", "Ограничения", "Источник"}
	}
}

func digestFallbackSectionBody(section, title, annotation, sourceURL, rawContent string) string {
	switch section {
	case "Источник":
		var parts []string
		if title != "" {
			parts = append(parts, "- Заголовок: "+title)
		}
		if sourceURL != "" {
			parts = append(parts, "- URL: "+sourceURL)
		}
		if len(parts) == 0 {
			excerpt := excerptRawContent(rawContent, 240)
			if excerpt != "" {
				parts = append(parts, "- Контекст: "+excerpt)
			}
		}

		return strings.Join(parts, "\n")
	default:
		if annotation != "" {
			return annotation
		}
		if title != "" {
			return title
		}

		return excerptRawContent(rawContent, 400)
	}
}

func excerptRawContent(rawContent string, maxLen int) string {
	text := strings.TrimSpace(rawContent)
	if text == "" || maxLen <= 0 {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}

	return strings.TrimSpace(text[:maxLen]) + "…"
}
