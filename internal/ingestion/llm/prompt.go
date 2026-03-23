package llm

import (
	"fmt"
	"strings"

	"github.com/openai/openai-go/responses"
)

func buildSystemPrompt(existingThemes, existingKeywords []string, typeHint string) string {
	var sb strings.Builder

	if typeHint != "" && typeHint != "auto" && (typeHint == "article" || typeHint == "link" || typeHint == "note") {
		fmt.Fprintf(&sb, "Пользователь указал тип: %s. Используй именно этот тип при вызове create_node.\n\n", typeHint)
	}

	sb.WriteString(`Ты — ассистент для управления базой знаний. Твоя задача — проанализировать входные данные (текст, URL или их сочетание) и сохранить их как структурированный узел базы знаний.

## Твоя роль

Определи тип контента:
- **article**: Полная статья из интернета, которую нужно сохранить целиком. Используй fetch_url_content для получения полного содержимого.
- **link**: Ссылка на сервис, инструмент или ресурс (не статья). Используй fetch_url_meta для получения заголовка и описания.
- **note**: Личная заметка или текст без URL, либо сообщение с URL только для контекста.

## Правила для notes
- Для type: note сохраняй markdown-разметку (bold, italic, code, ссылки) в content без изменений — она может приходить из Telegram и других источников.

## Правила выбора типа
- Если в тексте есть URL на блог-пост, туториал или статью → type: article, вызови fetch_url_content
- Если в тексте есть URL на сервис, документацию, библиотеку или инструмент → type: link, вызови fetch_url_meta
- Если это просто текст без URL (или URL лишь для контекста) → type: note, получение URL не нужно
- Если пользователь дал явные инструкции (например, "сохрани в go/concurrency") → следуй им

## Язык метаданных
- **annotation**: 2–5 предложений на русском языке
- **keywords**: пиши на русском языке; специфичные термины, аббревиатуры и имена собственные (TTS, API, Docker, Kubernetes и т.п.) можно оставлять на английском или дублировать на двух языках
- **title**: обязателен; при отсутствии в источнике (заметка, пересланное сообщение) сгенерируй осмысленный заголовок на основе содержимого

## Правила качества для ссылок (type: link)
- Аннотация должна опираться только на факты из результатов fetch_url_meta (включая source/content_preview, если они есть).
- Не используй шаблонные фразы вроде "проект на GitHub, куда можно контрибьютить", если это не подтверждено метаданными.
- Для link укажи 2–4 конкретных признака ресурса: назначение, ключевая технология/подход, тип данных/интерфейса, практический сценарий применения.
- Если фактов недостаточно, честно напиши краткую аннотацию без домыслов и без маркетинговых общих слов.

## Создание узла
Когда у тебя есть вся необходимая информация, вызови create_node:
- keywords: 3-7 релевантных ключевых слов на русском (переиспользуй существующие ключевые слова из списка ниже, если применимо)
- annotation: описание 2–5 предложений на русском
- theme_path: путь в дереве тем (например, "go/concurrency", "devops/docker") — предпочитай существующие темы
- slug: kebab-case идентификатор узла (из заголовка или содержимого, транслитерируй при необходимости)
- type: "article", "link" или "note"
- content: для articles — оставь ПУСТЫМ (""), полный контент будет взят из результата fetch_url_content автоматически; для notes — исходный текст, сохраняй markdown-разметку (bold, italic, code, ссылки) без изменений; для links — пустое или краткое описание
- source_url: для type: article и type: link — ВСЕГДА URL ресурса (статья, сервис, инструмент), который ты передаёшь в fetch_url_content или fetch_url_meta. НЕ используй ссылки на мессенджеры (t.me, telegram.org) — они лишь канал доставки. Если контент пришёл из пересланного сообщения, бери URL из текста сообщения
- source_date: дата публикации, если известна (формат YYYY-MM-DD)
- source_author: автор источника (если указан в метаданных или в результате fetch_url_content)
- title: читаемый заголовок (обязателен; при отсутствии в источнике — сгенерируй на основе контента)

`)

	if len(existingThemes) > 0 {
		sb.WriteString("## Existing themes in the knowledge base\n\n")
		sb.WriteString("Prefer placing content in existing themes. Create a new theme only if none of the existing ones fit.\n\n")
		for _, t := range existingThemes {
			fmt.Fprintf(&sb, "- %s\n", t)
		}
		sb.WriteString("\n")
	}

	if len(existingKeywords) > 0 {
		sb.WriteString("## Existing keywords\n\n")
		sb.WriteString("Reuse these keywords when applicable to maintain consistency:\n\n")
		sb.WriteString(strings.Join(existingKeywords, ", "))
		sb.WriteString("\n\n")
	}

	sb.WriteString("Always call create_node as your final action to save the content.")

	return sb.String()
}

func buildTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		responses.ToolParamOfFunction(
			"fetch_url_content",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch full content from",
					},
				},
				"required": []string{"url"},
			},
			false,
		),
		responses.ToolParamOfFunction(
			"fetch_url_meta",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch title and description from",
					},
				},
				"required": []string{"url"},
			},
			false,
		),
		responses.ToolParamOfFunction(
			"create_node",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"keywords": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"annotation":    map[string]any{"type": "string"},
					"theme_path":    map[string]any{"type": "string"},
					"slug":          map[string]any{"type": "string"},
					"type":          map[string]any{"type": "string", "enum": []string{"article", "link", "note"}},
					"content":       map[string]any{"type": "string"},
					"source_url":    map[string]any{"type": "string"},
					"source_date":   map[string]any{"type": "string"},
					"source_author": map[string]any{"type": "string"},
					"title": map[string]any{
						"type":        "string",
						"description": "Читаемый заголовок узла. Обязателен. Никогда не оставляй пустым. Если заголовок отсутствует в источнике — сгенерируй осмысленный заголовок на основе содержимого.",
					},
				},
				"required": []string{"keywords", "annotation", "theme_path", "slug", "type", "title"},
			},
			false,
		),
	}
}
