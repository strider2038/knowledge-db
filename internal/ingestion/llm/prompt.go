package llm

import (
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3/responses"
)

func buildSystemPrompt(input ProcessInput) string {
	var sb strings.Builder

	if input.TypeHint != "" && input.TypeHint != "auto" && (input.TypeHint == "article" || input.TypeHint == "link" || input.TypeHint == "note") {
		fmt.Fprintf(&sb, "Пользователь указал тип: %s. Используй именно этот тип при вызове create_node.\n\n", input.TypeHint)
	}
	if input.SourceKind != "" || input.ContentProfile != "" || input.RecommendedType != "" {
		fmt.Fprintf(&sb, "Предварительная классификация источника: source_kind=%s, content_profile=%s, рекомендуемый type=%s. Учитывай её при create_node; меняй только если доступный источник явно противоречит классификации.\n\n", input.SourceKind, input.ContentProfile, input.RecommendedType)
	}

	sb.WriteString(`Ты — ассистент для управления базой знаний. Твоя задача — проанализировать входные данные (текст, URL или их сочетание) и сохранить их как структурированный узел базы знаний.

## Твоя роль

Определи тип контента:
- **article**: Полная статья из интернета, которую нужно сохранить целиком. Используй fetch_url_content для получения полного содержимого.
- **link**: Закладка на внешний ресурс, репозиторий, документацию, сервис, онлайн-инструмент, каталог или учебный ресурс. Используй fetch_url_meta для заголовка, описания и README/content preview.
- **note**: Личная заметка, концептуальная выжимка длинной статьи без полного копирования, краткая выжимка новости или social post.

## Профиль источника
- source_kind описывает природу внешнего источника: repository, documentation, product_service, online_tool, directory_catalog, learning_resource, article, news, social_post, unknown.
- content_profile описывает локальную форму digest: repository_profile, product_profile, documentation_profile, online_tool_profile, directory_profile, learning_resource_profile, conceptual_digest, brief_digest, link_bookmark.
- Репозитории, сервисы, инструменты, каталоги, учебные ресурсы и документация обычно сохраняются как type: link с профильным markdown digest в content.
- Длинные статьи без полного копирования, новости и social posts сохраняются как type: note с digest в content.
- Если пользователь явно просит полную локальную копию или type hint article, сохраняй type: article и используй полный fetch_url_content.

## Правила для notes
- Для type: note сохраняй markdown-разметку (bold, italic, code, ссылки) в content без изменений — она может приходить из Telegram и других источников.
- Для note с content_profile conceptual_digest или brief_digest создай новое markdown-тело digest на русском языке, а не копируй источник целиком.

## Правила выбора типа
- Если в тексте есть URL на блог-пост, туториал или статью и нет явного запроса полной копии → type: note, source_kind: article, content_profile: conceptual_digest, вызови fetch_url_content только как материал для выжимки
- Если пользователь явно просит полную копию статьи → type: article, вызови fetch_url_content
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

## Digest body
- Для repository_profile включи разделы второго уровня про назначение, ключевую идею, основные абстракции, архитектурные свойства, применимость, ограничения и связанные понятия.
- Для product_profile включи какую работу закрывает продукт/сервис, для кого, основной workflow, данные/интеграции, ограничения и связанные понятия.
- Для documentation_profile включи ментальную модель документации, основные сущности, типовые сценарии, границы применимости и связанные понятия.
- Для online_tool_profile включи входные/выходные данные, решаемую задачу, ограничения, приватность/локальность и когда инструмент полезнее локальной альтернативы.
- Для directory_profile включи область каталога, принцип организации, классы ресурсов и связанные понятия.
- Для learning_resource_profile включи чему учит ресурс, структуру пути, prerequisites, практический результат и связанные понятия.
- Для conceptual_digest включи главную идею, ключевые принципы, эвристики, антипаттерны/ограничения и связанные понятия.
- Для brief_digest включи суть новости, технически важные изменения, возможную значимость, ограничения информации и связанные понятия.
- Не переноси installation, quick start, команды запуска, длинные usage/API-примеры, API reference, badges, changelog, license, sponsor, contributing и benchmark-таблицы без концептуального вывода.
- Digest должен опираться только на README, метаданные, preview или извлечённый контент. Если источник только позиционирует свойство, так и формулируй.

## Создание узла
Когда у тебя есть вся необходимая информация, вызови create_node:
- keywords: 3-7 релевантных ключевых слов на русском (переиспользуй каноничные keywords из placement context ниже, если применимо)
- annotation: описание 2–5 предложений на русском
- theme_path: путь в дереве тем (например, "go/concurrency", "devops/docker") — предпочитай существующие темы
- slug: kebab-case идентификатор узла (из заголовка или содержимого, транслитерируй при необходимости)
- type: "article", "link" или "note"
- source_kind: одно из допустимых значений, если источник классифицирован
- content_profile: одно из допустимых значений, если создаёшь digest/profile; для обычной закладки можно не указывать или указать link_bookmark
- content: для articles — оставь ПУСТЫМ (""), полный контент будет взят из результата fetch_url_content автоматически; для обычных notes — исходный текст; для профильных links и digest notes — markdown digest на русском языке
- source_url: для type: article и type: link — URL ресурса, который ты передаёшь в fetch_url_content или fetch_url_meta. Если ты загрузил веб-страницу (статья), в source_url указывай URL этой страницы, а не второстепенные ссылки из её текста (цитаты, сноски, примеры вроде claude.md). Если исходник — сообщение из мессенджера и в нём явно указан URL (в том числе репозиторий на github.com), это и есть source_url ресурса — используй именно его. НЕ подставляй общий «документационный» домен вместо конкретной ссылки из сообщения. НЕ используй ссылки на мессенджеры (t.me, telegram.org) как source_url — они лишь канал доставки
- source_date: дата публикации, если известна (формат YYYY-MM-DD)
- source_author: автор источника (если указан в метаданных или в результате fetch_url_content)
- title: читаемый заголовок (обязателен; при отсутствии в источнике — сгенерируй на основе контента)

## Правила размещения
- Используй placement context ниже как основной источник существующих тем, каноничных keywords и похожих узлов.
- Предпочитай candidate themes и candidate keywords, если они подходят по смыслу.
- Не создавай новый keyword-синоним, если в candidate keywords уже есть подходящее написание.
- Если пользователь явно попросил путь темы (например, "сохрани в go/concurrency"), эта инструкция важнее автоматического shortlist.
- Вызывай search_placement_candidates только если первичных candidates недостаточно, материал явно относится к другой ветке или есть сомнение между близкими темами.
- Финальным действием всё равно всегда должен быть create_node.

`)

	if hasPlacementContext(input.PlacementContext) {
		sb.WriteString("## Placement context\n\n")
		writePlacementContext(&sb, input.PlacementContext)
	}

	sb.WriteString("Always call create_node as your final action to save the content.")

	return sb.String()
}

func hasPlacementContext(ctx PlacementContext) bool {
	return len(ctx.ThemeMap) > 0 ||
		len(ctx.CandidateThemes) > 0 ||
		len(ctx.CandidateKeywords) > 0 ||
		len(ctx.SimilarNodes) > 0 ||
		ctx.ExplicitThemePath != ""
}

func writePlacementContext(sb *strings.Builder, ctx PlacementContext) {
	if ctx.Source != "" {
		fmt.Fprintf(sb, "Candidate source: %s\n\n", ctx.Source)
	}
	if ctx.ExplicitThemePath != "" {
		fmt.Fprintf(sb, "Explicit user theme instruction detected: %s\n\n", ctx.ExplicitThemePath)
	}
	writeThemeMap(sb, ctx.ThemeMap)
	writeCandidateThemes(sb, ctx.CandidateThemes)
	writeCandidateKeywords(sb, ctx.CandidateKeywords)
	writeSimilarNodes(sb, ctx.SimilarNodes)
}

func writeThemeMap(sb *strings.Builder, themes []ThemeSummary) {
	if len(themes) == 0 {
		return
	}
	sb.WriteString("### Compact theme map\n")
	for _, theme := range themes {
		fmt.Fprintf(sb, "- %s (nodes: %d", theme.Path, theme.NodeCount)
		if len(theme.TopKeywords) > 0 {
			fmt.Fprintf(sb, ", top keywords: %s", strings.Join(theme.TopKeywords, ", "))
		}
		sb.WriteString(")\n")
	}
	sb.WriteString("\n")
}

func writeCandidateThemes(sb *strings.Builder, themes []ThemeCandidate) {
	if len(themes) == 0 {
		return
	}
	sb.WriteString("### Candidate themes\n")
	for i, theme := range themes {
		fmt.Fprintf(sb, "%d. %s (score: %.1f, nodes: %d)\n", i+1, theme.Path, theme.Score, theme.NodeCount)
		writeStringListLine(sb, "reason", theme.Reasons)
		writeStringListLine(sb, "examples", theme.Examples)
		writeStringListLine(sb, "top keywords", theme.TopKeywords)
	}
	sb.WriteString("\n")
}

func writeCandidateKeywords(sb *strings.Builder, keywords []KeywordCandidate) {
	if len(keywords) == 0 {
		return
	}
	sb.WriteString("### Candidate keywords\n")
	for _, keyword := range keywords {
		fmt.Fprintf(sb, "- %s (score: %.1f, frequency: %d", keyword.Keyword, keyword.Score, keyword.Frequency)
		if len(keyword.Themes) > 0 {
			fmt.Fprintf(sb, ", themes: %s", strings.Join(keyword.Themes, ", "))
		}
		if len(keyword.Sources) > 0 {
			fmt.Fprintf(sb, ", sources: %s", strings.Join(keyword.Sources, ", "))
		}
		sb.WriteString(")\n")
	}
	sb.WriteString("\n")
}

func writeSimilarNodes(sb *strings.Builder, nodes []SimilarNode) {
	if len(nodes) == 0 {
		return
	}
	sb.WriteString("### Similar nodes\n")
	for _, node := range nodes {
		fmt.Fprintf(sb, "- %s", node.Path)
		if node.Title != "" {
			fmt.Fprintf(sb, " — %s", node.Title)
		}
		if len(node.Keywords) > 0 {
			fmt.Fprintf(sb, " (keywords: %s)", strings.Join(node.Keywords, ", "))
		}
		if len(node.MatchReasons) > 0 {
			fmt.Fprintf(sb, "; match: %s", strings.Join(node.MatchReasons, ", "))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func writeStringListLine(sb *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}

	fmt.Fprintf(sb, "   %s: %s\n", label, strings.Join(values, ", "))
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
			"search_placement_candidates",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query for local placement candidates",
					},
					"source_kind": map[string]any{
						"type": "string",
						"enum": []string{"repository", "documentation", "product_service", "online_tool", "directory_catalog", "learning_resource", "article", "news", "social_post", "unknown"},
					},
					"content_profile": map[string]any{
						"type": "string",
						"enum": []string{"repository_profile", "product_profile", "documentation_profile", "online_tool_profile", "directory_profile", "learning_resource_profile", "conceptual_digest", "brief_digest", "link_bookmark"},
					},
					"type": map[string]any{"type": "string", "enum": []string{"article", "link", "note"}},
				},
				"required": []string{"query"},
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
					"annotation": map[string]any{"type": "string"},
					"theme_path": map[string]any{"type": "string"},
					"slug":       map[string]any{"type": "string"},
					"type":       map[string]any{"type": "string", "enum": []string{"article", "link", "note"}},
					"source_kind": map[string]any{
						"type": "string",
						"enum": []string{"repository", "documentation", "product_service", "online_tool", "directory_catalog", "learning_resource", "article", "news", "social_post", "unknown"},
					},
					"content_profile": map[string]any{
						"type": "string",
						"enum": []string{"repository_profile", "product_profile", "documentation_profile", "online_tool_profile", "directory_profile", "learning_resource_profile", "conceptual_digest", "brief_digest", "link_bookmark"},
					},
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
