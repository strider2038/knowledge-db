package ingestion

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
	"github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/ingestion/translation"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationqueue"
	"github.com/strider2038/knowledge-db/internal/kb"
	"github.com/strider2038/knowledge-db/internal/pkg/urlutil"
)

// TitleGenerator — интерфейс для генерации заголовка через LLM.
// Используется как fallback, когда LLM-оркестратор вернул пустой title
// и извлечь его из контента не получилось.
type TitleGenerator interface {
	GenerateTitle(ctx context.Context, content string) (string, error)
}

// PipelineIngester — полноценный ingestion pipeline с LLM-оркестратором.
type PipelineIngester struct {
	store            *kb.Store
	orchestrator     llm.LLMOrchestrator
	contentFetcher   fetcher.ContentFetcher
	committer        git.GitCommitter
	basePath         string
	autoTranslate    bool
	expandURLs       bool
	translator       translation.Translator
	titleGenerator   TitleGenerator
	translationQueue *translationqueue.Queue
}

// NewPipelineIngester создаёт PipelineIngester.
// translationQueue — опционально; при nil используется синхронный перевод.
func NewPipelineIngester(
	store *kb.Store,
	orchestrator llm.LLMOrchestrator,
	contentFetcher fetcher.ContentFetcher,
	committer git.GitCommitter,
	basePath string,
	autoTranslate bool,
	expandURLs bool,
	translator translation.Translator,
	titleGenerator TitleGenerator,
	translationQueue *translationqueue.Queue,
) *PipelineIngester {
	return &PipelineIngester{
		store:            store,
		orchestrator:     orchestrator,
		contentFetcher:   contentFetcher,
		committer:        committer,
		basePath:         basePath,
		autoTranslate:    autoTranslate,
		expandURLs:       expandURLs,
		translator:       translator,
		titleGenerator:   titleGenerator,
		translationQueue: translationQueue,
	}
}

// IngestText обрабатывает входной текст через LLM-оркестратор и сохраняет узел.
func (p *PipelineIngester) IngestText(ctx context.Context, req IngestRequest) (*kb.Node, error) {
	clog.Info(ctx, "ingest text: start", "text_len", len(req.Text))

	profile := ClassifySource(req.SourceURL, "", req.Text, req.TypeHint)
	processInput, err := p.buildProcessInput(ctx, req.Text, req.SourceURL, req.SourceAuthor, req.TypeHint, profile)
	if err != nil {
		return nil, errors.Errorf("ingest text: build context: %w", err)
	}

	llmStart := time.Now()
	clog.Info(ctx, "ingest: llm process")
	result, err := p.orchestrator.Process(ctx, processInput)
	if err != nil {
		return nil, errors.Errorf("ingest text: orchestrate: %w", err)
	}
	applyProfileToResult(result, profile)
	clog.Info(ctx, "ingest: llm process done", "duration_ms", time.Since(llmStart).Milliseconds())

	node, err := p.saveNode(ctx, result)
	if err != nil {
		return nil, err
	}
	if err := p.maybeTranslateAndSave(ctx, result, node); err != nil {
		clog.Errorf(ctx, "ingest text: translation failed: %w", err)
	}
	clog.Info(ctx, "ingest text: complete", "theme", result.ThemePath, "slug", result.Slug)

	return node, nil
}

// IngestURL явно загружает контент по URL через ContentFetcher, затем обрабатывает через LLM.
func (p *PipelineIngester) IngestURL(ctx context.Context, url string) (*kb.Node, error) {
	if url != "" {
		if normalized, err := urlutil.NormalizeURL(ctx, url); err == nil {
			url = normalized
		}
	}
	clog.Info(ctx, "ingest url: start", "url", url)

	fetchStart := time.Now()
	fetchResult, err := p.contentFetcher.Fetch(ctx, url)
	if err != nil {
		clog.Warn(ctx, "ingest url: fetch failed, passing url as text", "url", url, "error", err)
		fetchResult = nil
	} else {
		clog.Info(ctx, "ingest url: fetch complete", "url", url, "title", fetchResult.Title, "duration_ms", time.Since(fetchStart).Milliseconds())
	}

	var text string
	var sourceAuthor string
	if fetchResult != nil {
		text = fmt.Sprintf("URL: %s\nTitle: %s\n\n%s", url, fetchResult.Title, fetchResult.Content)
		sourceAuthor = fetchResult.Author
	} else {
		text = url
	}

	title := ""
	content := ""
	if fetchResult != nil {
		title = fetchResult.Title
		content = fetchResult.Content
	}
	profile := ClassifySource(url, title, content, "")
	processInput, err := p.buildProcessInput(ctx, text, url, sourceAuthor, "", profile)
	if err != nil {
		return nil, errors.Errorf("ingest url: build context: %w", err)
	}

	llmStart := time.Now()
	clog.Info(ctx, "ingest: llm process")
	result, err := p.orchestrator.Process(ctx, processInput)
	if err != nil {
		return nil, errors.Errorf("ingest url: orchestrate: %w", err)
	}
	applyProfileToResult(result, profile)
	clog.Info(ctx, "ingest: llm process done", "duration_ms", time.Since(llmStart).Milliseconds())

	node, err := p.saveNode(ctx, result)
	if err != nil {
		return nil, err
	}
	if err := p.maybeTranslateAndSave(ctx, result, node); err != nil {
		clog.Errorf(ctx, "ingest url: translation failed: %w", err)
	}
	clog.Info(ctx, "ingest url: complete", "url", url, "theme", result.ThemePath, "slug", result.Slug)

	return node, nil
}

func (p *PipelineIngester) RefreshDescription(ctx context.Context, path string) (*kb.Node, error) {
	clog.Info(ctx, "refresh description: start", "path", path)

	current, err := p.store.GetNodeFile(ctx, p.basePath, path)
	if err != nil {
		return nil, errors.Errorf("refresh description: get node: %w", err)
	}
	sourceURL, _ := current.Frontmatter["source_url"].(string)
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return nil, errors.Errorf("refresh description: %w", ErrSourceURLRequired)
	}

	fetchResult, err := p.contentFetcher.Fetch(ctx, sourceURL)
	if err != nil {
		return nil, errors.Errorf("refresh description: fetch source: %w", err)
	}

	text := fmt.Sprintf("URL: %s\nTitle: %s\n\n%s", sourceURL, fetchResult.Title, fetchResult.Content)
	profile := ClassifySource(sourceURL, fetchResult.Title, fetchResult.Content, "")
	processInput, err := p.buildProcessInput(ctx, text, sourceURL, fetchResult.Author, "", profile)
	if err != nil {
		return nil, errors.Errorf("refresh description: build context: %w", err)
	}

	result, err := p.orchestrator.Process(ctx, processInput)
	if err != nil {
		return nil, errors.Errorf("refresh description: orchestrate: %w", err)
	}
	applyProfileToResult(result, profile)
	if result.SourceURL == "" {
		result.SourceURL = sourceURL
	}
	if result.SourceAuthor == "" {
		if existing, ok := current.Frontmatter["source_author"].(string); ok {
			result.SourceAuthor = existing
		}
	}
	if result.SourceDate == nil {
		if existing, ok := current.Frontmatter["source_date"].(string); ok && existing != "" {
			if parsed, parseErr := time.Parse("2006-01-02", existing); parseErr == nil {
				result.SourceDate = &parsed
			}
		}
	}

	frontmatter := cloneFrontmatter(current.Frontmatter)
	p.applyResultToExistingFrontmatter(ctx, frontmatter, result)
	node, err := p.store.UpdateNode(ctx, p.basePath, path, kb.UpdateNodeParams{
		Frontmatter: frontmatter,
		Content:     result.Content,
	})
	if err != nil {
		return nil, errors.Errorf("refresh description: update node: %w", err)
	}

	clog.Info(ctx, "refresh description: complete", "path", path)

	return node, nil
}

func (p *PipelineIngester) buildProcessInput(ctx context.Context, text, sourceURL, sourceAuthor, typeHint string, profile SourceProfile) (llm.ProcessInput, error) {
	tree, err := p.store.ReadTree(ctx, p.basePath)
	if err != nil {
		return llm.ProcessInput{}, errors.Errorf("read tree: %w", err)
	}

	themes := collectThemes(tree)
	keywords, err := p.collectKeywords(ctx)
	if err != nil {
		clog.Warn(ctx, "ingest: failed to collect keywords, proceeding without them", "error", err)
	}

	if sourceURL != "" || sourceAuthor != "" {
		var sourcePrefix string
		if strings.HasPrefix(sourceURL, "https://t.me/") {
			// t.me — это канал доставки (Telegram), а не URL сохраняемого ресурса.
			// Даём явный лейбл, чтобы LLM не использовал его как source_url при type: link/article.
			sourcePrefix = fmt.Sprintf("Telegram-канал (откуда получен контент, НЕ является source_url ресурса): %s, автор: %s", sourceURL, sourceAuthor)
		} else {
			sourcePrefix = fmt.Sprintf("Метаданные источника: ссылка: %s, автор: %s", sourceURL, sourceAuthor)
		}
		text = sourcePrefix + "\n\n" + text
	}
	if profile.HasProfile() {
		text = fmt.Sprintf("Профиль источника: source_kind=%s, content_profile=%s, recommended_type=%s\n\n%s", profile.SourceKind, profile.ContentProfile, profile.RecommendedType, text)
	}

	return llm.ProcessInput{
		Text:             text,
		SourceURL:        sourceURL,
		SourceAuthor:     sourceAuthor,
		TypeHint:         typeHint,
		SourceKind:       string(profile.SourceKind),
		ContentProfile:   string(profile.ContentProfile),
		RecommendedType:  profile.RecommendedType,
		ExistingThemes:   themes,
		ExistingKeywords: keywords,
	}, nil
}

// expandMarkdownURLs раскрывает короткие ссылки и снимает UTM в теле markdown (как kb-cli expand-urls).
func (p *PipelineIngester) expandMarkdownURLs(ctx context.Context, s string) string {
	out, res := kb.ExpandURLsInString(ctx, s)
	if len(res.FailedURLs) > 0 {
		for _, u := range res.FailedURLs {
			clog.Warn(ctx, "ingest: expand URL failed", "url", u)
		}
	}
	if res.Changed {
		clog.Info(ctx, "ingest: URL postprocess", "replacements", res.Replacements)
	}

	return out
}

func (p *PipelineIngester) saveNode(ctx context.Context, result *llm.ProcessResult) (*kb.Node, error) {
	saveStart := time.Now()
	now := time.Now().UTC().Format(time.RFC3339)
	frontmatter := map[string]any{
		"keywords":   result.Keywords,
		"created":    now,
		"updated":    now,
		"annotation": result.Annotation,
	}
	title := result.Title
	if title == "" {
		title = extractTitleFromContent(result.Content)
	}
	if title == "" && p.titleGenerator != nil {
		generated, genErr := p.titleGenerator.GenerateTitle(ctx, result.Content)
		if genErr != nil {
			clog.Errorf(ctx, "saveNode: generate title: %w", genErr)
		} else {
			title = generated
		}
	}
	if title == "" && result.Slug != "" {
		title = slugToTitle(result.Slug)
	}
	if title != "" {
		title = stripMarkdownFromTitle(title)
		frontmatter["title"] = title
		frontmatter["aliases"] = []string{title}
	}
	if result.Type != "" {
		frontmatter["type"] = result.Type
	}
	applyProfileToFrontmatter(frontmatter, result.SourceKind, result.ContentProfile)
	if result.SourceURL != "" {
		sourceURL := result.SourceURL
		sourceURL = urlutil.StripTrackingParamsFromURL(sourceURL)
		frontmatter["source_url"] = sourceURL
	}
	if result.SourceDate != nil {
		frontmatter["source_date"] = result.SourceDate.Format("2006-01-02")
	}
	if result.SourceAuthor != "" {
		frontmatter["source_author"] = result.SourceAuthor
	}

	if p.expandURLs {
		result.Content = p.expandMarkdownURLs(ctx, result.Content)
		if ann, ok := frontmatter["annotation"].(string); ok && ann != "" {
			frontmatter["annotation"] = p.expandMarkdownURLs(ctx, ann)
		}
	}

	node, err := p.store.CreateNode(ctx, p.basePath, kb.CreateNodeParams{
		ThemePath:   result.ThemePath,
		Slug:        result.Slug,
		Frontmatter: frontmatter,
		Content:     result.Content,
	})
	if err != nil {
		return nil, errors.Errorf("save node: %w", err)
	}
	clog.Info(ctx, "ingest: node created", "theme", result.ThemePath, "slug", result.Slug, "duration_ms", time.Since(saveStart).Milliseconds())

	commitMsg := fmt.Sprintf("add: %s/%s", result.ThemePath, result.Slug)
	nodeMdPath := filepath.Join(p.basePath, filepath.FromSlash(result.ThemePath), result.Slug+".md")
	if err := p.committer.CommitNode(ctx, nodeMdPath, commitMsg); err != nil {
		clog.Errorf(ctx, "save node: git commit failed: %w", err)
	}

	return node, nil
}

func (p *PipelineIngester) applyResultToExistingFrontmatter(ctx context.Context, frontmatter map[string]any, result *llm.ProcessResult) {
	now := time.Now().UTC().Format(time.RFC3339)
	frontmatter["updated"] = now
	frontmatter["keywords"] = result.Keywords
	frontmatter["annotation"] = result.Annotation
	if result.Title != "" {
		title := stripMarkdownFromTitle(result.Title)
		frontmatter["title"] = title
		frontmatter["aliases"] = []string{title}
	}
	if result.Type != "" {
		frontmatter["type"] = result.Type
	}
	applyProfileToFrontmatter(frontmatter, result.SourceKind, result.ContentProfile)
	if result.SourceURL != "" {
		frontmatter["source_url"] = urlutil.StripTrackingParamsFromURL(result.SourceURL)
	}
	if result.SourceDate != nil {
		frontmatter["source_date"] = result.SourceDate.Format("2006-01-02")
	}
	if result.SourceAuthor != "" {
		frontmatter["source_author"] = result.SourceAuthor
	}
	if p.expandURLs {
		result.Content = p.expandMarkdownURLs(ctx, result.Content)
		if ann, ok := frontmatter["annotation"].(string); ok && ann != "" {
			frontmatter["annotation"] = p.expandMarkdownURLs(ctx, ann)
		}
	}
}

func applyProfileToResult(result *llm.ProcessResult, profile SourceProfile) {
	if result == nil || !profile.HasProfile() {
		return
	}
	if result.SourceKind == "" {
		result.SourceKind = string(profile.SourceKind)
	}
	if result.ContentProfile == "" {
		result.ContentProfile = string(profile.ContentProfile)
	}
	if result.Type == "" {
		result.Type = profile.RecommendedType
	}
}

func applyProfileToFrontmatter(frontmatter map[string]any, sourceKind, contentProfile string) {
	if sourceKind != "" && kb.IsValidSourceKind(sourceKind) {
		frontmatter["source_kind"] = sourceKind
	}
	if contentProfile != "" && kb.IsValidContentProfile(contentProfile) {
		frontmatter["content_profile"] = contentProfile
	}
}

func cloneFrontmatter(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}

	return dst
}

func (p *PipelineIngester) maybeTranslateAndSave(ctx context.Context, result *llm.ProcessResult, node *kb.Node) error {
	log := clog.FromContext(ctx)
	if !p.autoTranslate || p.translator == nil {
		log.Debug("translation: skipped", "reason", "auto_translate_disabled_or_no_translator", "theme", result.ThemePath, "slug", result.Slug)

		return nil
	}
	if result.Type != "article" {
		log.Debug("translation: skipped", "reason", "type_not_article", "type", result.Type, "theme", result.ThemePath, "slug", result.Slug)

		return nil
	}
	if !translation.NeedsTranslation(result.Content) {
		log.Debug("translation: skipped", "reason", "content_already_russian", "theme", result.ThemePath, "slug", result.Slug)

		return nil
	}

	// Асинхронный режим: ставим в очередь вместо синхронного перевода.
	if p.translationQueue != nil {
		translationPath := result.ThemePath + "/" + result.Slug + ".ru"
		if _, err := p.store.GetNode(ctx, p.basePath, translationPath); err == nil {
			log.Debug("translation: skipped", "reason", "translation_exists", "theme", result.ThemePath, "slug", result.Slug)

			return nil
		}
		status, _ := p.translationQueue.Enqueue(result.ThemePath, result.Slug)
		log.Info("translation: enqueued", "theme", result.ThemePath, "slug", result.Slug, "status", status)

		return nil
	}

	// Синхронный режим (для тестов и обратной совместимости).
	log.Info("translation: start", "theme", result.ThemePath, "slug", result.Slug, "content_len", len(result.Content))
	translateStart := time.Now()
	translated, err := p.translator.Translate(ctx, result.Content)
	if err != nil {
		return errors.Errorf("translate: %w", err)
	}
	log.Info("translation: llm done", "theme", result.ThemePath, "slug", result.Slug, "duration_ms", time.Since(translateStart).Milliseconds(), "translated_len", len(translated))

	saveStart := time.Now()
	translationFrontmatter := map[string]any{
		"translation_of": result.Slug,
		"lang":           "ru",
		"keywords":       result.Keywords,
		"created":        node.Metadata["created"],
		"updated":        node.Metadata["updated"],
		"annotation":     result.Annotation,
		"type":           "article",
	}
	if title, ok := node.Metadata["title"]; ok {
		translationFrontmatter["title"] = title
	}
	if aliases, ok := node.Metadata["aliases"]; ok {
		translationFrontmatter["aliases"] = aliases
	}
	if result.SourceURL != "" {
		if u, ok := node.Metadata["source_url"].(string); ok && u != "" {
			translationFrontmatter["source_url"] = u
		} else {
			sourceURL := result.SourceURL
			sourceURL = urlutil.StripTrackingParamsFromURL(sourceURL)
			translationFrontmatter["source_url"] = sourceURL
		}
	}
	if result.SourceDate != nil {
		translationFrontmatter["source_date"] = result.SourceDate.Format("2006-01-02")
	}
	if result.SourceAuthor != "" {
		translationFrontmatter["source_author"] = result.SourceAuthor
	}
	applyProfileToFrontmatter(translationFrontmatter, result.SourceKind, result.ContentProfile)

	contentWithLink := translated
	if !strings.HasSuffix(contentWithLink, fmt.Sprintf("[[%s|Original]]", result.Slug)) {
		contentWithLink = strings.TrimSuffix(contentWithLink, "\n") + "\n\n" + fmt.Sprintf("[[%s|Original]]", result.Slug) + "\n"
	}

	if p.expandURLs {
		contentWithLink = p.expandMarkdownURLs(ctx, contentWithLink)
	}

	if err := p.store.CreateTranslationFile(ctx, p.basePath, result.ThemePath, result.Slug, "ru", translationFrontmatter, contentWithLink); err != nil {
		return errors.Errorf("create translation file: %w", err)
	}
	if err := p.store.AppendTranslationsToOriginal(ctx, p.basePath, result.ThemePath, result.Slug, result.Slug+".ru"); err != nil {
		return errors.Errorf("append translations to original: %w", err)
	}

	translationPath := filepath.Join(p.basePath, filepath.FromSlash(result.ThemePath), result.Slug+".ru.md")
	if err := p.committer.CommitNode(ctx, translationPath, fmt.Sprintf("add: %s/%s.ru (translation)", result.ThemePath, result.Slug)); err != nil {
		clog.Errorf(ctx, "maybeTranslateAndSave: git commit translation failed: %w", err)
	}
	originalPath := filepath.Join(p.basePath, filepath.FromSlash(result.ThemePath), result.Slug+".md")
	if err := p.committer.CommitNode(ctx, originalPath, fmt.Sprintf("add: %s/%s (translation link)", result.ThemePath, result.Slug)); err != nil {
		clog.Errorf(ctx, "maybeTranslateAndSave: git commit original update failed: %w", err)
	}

	log.Info("translation: complete", "theme", result.ThemePath, "slug", result.Slug, "translation_slug", result.Slug+".ru", "save_duration_ms", time.Since(saveStart).Milliseconds())

	return nil
}

func (p *PipelineIngester) collectKeywords(ctx context.Context) ([]string, error) {
	nodes, err := p.store.ListAllNodes(ctx, p.basePath)
	if err != nil {
		return nil, err
	}

	keywordSet := make(map[string]struct{})
	for _, n := range nodes {
		node, err := p.store.GetNode(ctx, p.basePath, n.Path)
		if err != nil {
			continue
		}
		if kws, ok := node.Metadata["keywords"]; ok {
			switch v := kws.(type) {
			case []any:
				for _, kw := range v {
					if s, ok := kw.(string); ok {
						keywordSet[s] = struct{}{}
					}
				}
			case []string:
				for _, s := range v {
					keywordSet[s] = struct{}{}
				}
			}
		}
	}

	result := make([]string, 0, len(keywordSet))
	for kw := range keywordSet {
		result = append(result, kw)
	}

	return result, nil
}

func collectThemes(tree *kb.TreeNode) []string {
	var themes []string
	var walk func(node *kb.TreeNode)
	walk = func(node *kb.TreeNode) {
		for _, child := range node.Children {
			themes = append(themes, child.Path)
			walk(child)
		}
	}
	walk(tree)

	return themes
}

// stripMarkdownFromTitle убирает markdown-разметку (**bold**, __bold__, `code`) из заголовка.
func stripMarkdownFromTitle(s string) string {
	s = strings.TrimSpace(s)
	for {
		before := s
		if strings.HasPrefix(s, "**") && strings.HasSuffix(s, "**") && len(s) > 4 {
			s = strings.TrimPrefix(strings.TrimSuffix(s, "**"), "**")
		}
		if strings.HasPrefix(s, "__") && strings.HasSuffix(s, "__") && len(s) > 4 {
			s = strings.TrimPrefix(strings.TrimSuffix(s, "__"), "__")
		}
		if strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") && len(s) > 2 {
			s = strings.TrimPrefix(strings.TrimSuffix(s, "`"), "`")
		}
		s = strings.TrimSpace(s)
		if s == before {
			break
		}
	}

	return s
}

// extractTitleFromContent извлекает заголовок из первой непустой строки контента.
// Обрабатывает markdown-заголовки (# Title). Возвращает пустую строку, если
// подходящего заголовка нет (пустой контент, первая строка слишком длинная).
func extractTitleFromContent(content string) string {
	const maxTitleLen = 150
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) <= maxTitleLen {
			return line
		}

		return ""
	}

	return ""
}

// slugToTitle converts a slug (e.g. "professor-donald-knuth-clause-cycles") to Title Case.
// Used as last resort fallback when LLM returns empty title.
func slugToTitle(slug string) string {
	if slug == "" {
		return ""
	}
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(string(p[0])) + strings.ToLower(p[1:])
		}
	}

	return strings.Join(parts, " ")
}
