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
	"github.com/strider2038/knowledge-db/internal/kb"
)

// PipelineIngester — полноценный ingestion pipeline с LLM-оркестратором.
type PipelineIngester struct {
	store          *kb.Store
	orchestrator   llm.LLMOrchestrator
	contentFetcher fetcher.ContentFetcher
	committer      git.GitCommitter
	basePath       string
	autoTranslate  bool
	translator     translation.Translator
}

// NewPipelineIngester создаёт PipelineIngester.
func NewPipelineIngester(
	store *kb.Store,
	orchestrator llm.LLMOrchestrator,
	contentFetcher fetcher.ContentFetcher,
	committer git.GitCommitter,
	basePath string,
	autoTranslate bool,
	translator translation.Translator,
) *PipelineIngester {
	return &PipelineIngester{
		store:          store,
		orchestrator:   orchestrator,
		contentFetcher: contentFetcher,
		committer:      committer,
		basePath:       basePath,
		autoTranslate:  autoTranslate,
		translator:     translator,
	}
}

// IngestText обрабатывает входной текст через LLM-оркестратор и сохраняет узел.
func (p *PipelineIngester) IngestText(ctx context.Context, req IngestRequest) (*kb.Node, error) {
	clog.FromContext(ctx).Info("ingest text: start", "text_len", len(req.Text))

	processInput, err := p.buildProcessInput(ctx, req.Text, req.SourceURL, req.SourceAuthor)
	if err != nil {
		return nil, errors.Errorf("ingest text: build context: %w", err)
	}

	llmStart := time.Now()
	clog.FromContext(ctx).Info("ingest: llm process")
	result, err := p.orchestrator.Process(ctx, processInput)
	if err != nil {
		return nil, errors.Errorf("ingest text: orchestrate: %w", err)
	}
	clog.FromContext(ctx).Info("ingest: llm process done", "duration_ms", time.Since(llmStart).Milliseconds())

	node, err := p.saveNode(ctx, result)
	if err != nil {
		return nil, err
	}
	if err := p.maybeTranslateAndSave(ctx, result, node); err != nil {
		clog.Errorf(ctx, "ingest text: translation failed: %w", err)
	}
	clog.FromContext(ctx).Info("ingest text: complete", "theme", result.ThemePath, "slug", result.Slug)

	return node, nil
}

// IngestURL явно загружает контент по URL через ContentFetcher, затем обрабатывает через LLM.
func (p *PipelineIngester) IngestURL(ctx context.Context, url string) (*kb.Node, error) {
	clog.FromContext(ctx).Info("ingest url: start", "url", url)

	fetchStart := time.Now()
	fetchResult, err := p.contentFetcher.Fetch(ctx, url)
	if err != nil {
		clog.FromContext(ctx).Warn("ingest url: fetch failed, passing url as text", "url", url, "error", err)
		fetchResult = nil
	} else {
		clog.FromContext(ctx).Info("ingest url: fetch complete", "url", url, "title", fetchResult.Title, "duration_ms", time.Since(fetchStart).Milliseconds())
	}

	var text string
	var sourceAuthor string
	if fetchResult != nil {
		text = fmt.Sprintf("URL: %s\nTitle: %s\n\n%s", url, fetchResult.Title, fetchResult.Content)
		sourceAuthor = fetchResult.Author
	} else {
		text = url
	}

	processInput, err := p.buildProcessInput(ctx, text, url, sourceAuthor)
	if err != nil {
		return nil, errors.Errorf("ingest url: build context: %w", err)
	}

	llmStart := time.Now()
	clog.FromContext(ctx).Info("ingest: llm process")
	result, err := p.orchestrator.Process(ctx, processInput)
	if err != nil {
		return nil, errors.Errorf("ingest url: orchestrate: %w", err)
	}
	clog.FromContext(ctx).Info("ingest: llm process done", "duration_ms", time.Since(llmStart).Milliseconds())

	node, err := p.saveNode(ctx, result)
	if err != nil {
		return nil, err
	}
	if err := p.maybeTranslateAndSave(ctx, result, node); err != nil {
		clog.Errorf(ctx, "ingest url: translation failed: %w", err)
	}
	clog.FromContext(ctx).Info("ingest url: complete", "url", url, "theme", result.ThemePath, "slug", result.Slug)

	return node, nil
}

func (p *PipelineIngester) buildProcessInput(ctx context.Context, text, sourceURL, sourceAuthor string) (llm.ProcessInput, error) {
	tree, err := p.store.ReadTree(ctx, p.basePath)
	if err != nil {
		return llm.ProcessInput{}, errors.Errorf("read tree: %w", err)
	}

	themes := collectThemes(tree)
	keywords, err := p.collectKeywords(ctx)
	if err != nil {
		clog.FromContext(ctx).Warn("ingest: failed to collect keywords, proceeding without them", "error", err)
	}

	if sourceURL != "" || sourceAuthor != "" {
		text = fmt.Sprintf("Метаданные источника: ссылка: %s, автор: %s\n\n%s", sourceURL, sourceAuthor, text)
	}

	return llm.ProcessInput{
		Text:             text,
		SourceURL:        sourceURL,
		SourceAuthor:     sourceAuthor,
		ExistingThemes:   themes,
		ExistingKeywords: keywords,
	}, nil
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
	if title == "" && result.Slug != "" {
		title = slugToTitle(result.Slug)
	}
	if title != "" {
		frontmatter["title"] = title
		frontmatter["aliases"] = []string{title}
	}
	if result.Type != "" {
		frontmatter["type"] = result.Type
	}
	if result.SourceURL != "" {
		frontmatter["source_url"] = result.SourceURL
	}
	if result.SourceDate != nil {
		frontmatter["source_date"] = result.SourceDate.Format("2006-01-02")
	}
	if result.SourceAuthor != "" {
		frontmatter["source_author"] = result.SourceAuthor
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
	clog.FromContext(ctx).Info("ingest: node created", "theme", result.ThemePath, "slug", result.Slug, "duration_ms", time.Since(saveStart).Milliseconds())

	commitMsg := fmt.Sprintf("add: %s/%s", result.ThemePath, result.Slug)
	nodeMdPath := filepath.Join(p.basePath, filepath.FromSlash(result.ThemePath), result.Slug+".md")
	if err := p.committer.CommitNode(ctx, nodeMdPath, commitMsg); err != nil {
		clog.Errorf(ctx, "save node: git commit failed: %w", err)
	}

	return node, nil
}

func (p *PipelineIngester) maybeTranslateAndSave(ctx context.Context, result *llm.ProcessResult, node *kb.Node) error {
	if !p.autoTranslate || p.translator == nil {
		return nil
	}
	if result.Type != "article" {
		return nil
	}
	if !translation.NeedsTranslation(result.Content) {
		return nil
	}

	translated, err := p.translator.Translate(ctx, result.Content)
	if err != nil {
		return errors.Errorf("translate: %w", err)
	}

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
		translationFrontmatter["source_url"] = result.SourceURL
	}
	if result.SourceDate != nil {
		translationFrontmatter["source_date"] = result.SourceDate.Format("2006-01-02")
	}
	if result.SourceAuthor != "" {
		translationFrontmatter["source_author"] = result.SourceAuthor
	}

	contentWithLink := translated
	if !strings.HasSuffix(contentWithLink, fmt.Sprintf("[[%s|Original]]", result.Slug)) {
		contentWithLink = strings.TrimSuffix(contentWithLink, "\n") + "\n\n" + fmt.Sprintf("[[%s|Original]]", result.Slug) + "\n"
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

	clog.FromContext(ctx).Info("translation: added", "theme", result.ThemePath, "slug", result.Slug, "translation_slug", result.Slug+".ru")

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

// slugToTitle converts a slug (e.g. "professor-donald-knuth-clause-cycles") to Title Case.
// Used as fallback when LLM returns empty title.
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
