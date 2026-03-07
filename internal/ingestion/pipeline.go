package ingestion

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
	"github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/kb"
)

// PipelineIngester — полноценный ingestion pipeline с LLM-оркестратором.
type PipelineIngester struct {
	store          *kb.Store
	orchestrator   llm.LLMOrchestrator
	contentFetcher fetcher.ContentFetcher
	committer      git.GitCommitter
	basePath       string
}

// NewPipelineIngester создаёт PipelineIngester.
func NewPipelineIngester(
	store *kb.Store,
	orchestrator llm.LLMOrchestrator,
	contentFetcher fetcher.ContentFetcher,
	committer git.GitCommitter,
	basePath string,
) *PipelineIngester {
	return &PipelineIngester{
		store:          store,
		orchestrator:   orchestrator,
		contentFetcher: contentFetcher,
		committer:      committer,
		basePath:       basePath,
	}
}

// IngestText обрабатывает входной текст через LLM-оркестратор и сохраняет узел.
func (p *PipelineIngester) IngestText(ctx context.Context, text string) (*kb.Node, error) {
	clog.FromContext(ctx).Info("ingest text: start", "text_len", len(text))

	processInput, err := p.buildProcessInput(ctx, text)
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
	if fetchResult != nil {
		text = fmt.Sprintf("URL: %s\nTitle: %s\n\n%s", url, fetchResult.Title, fetchResult.Content)
	} else {
		text = url
	}

	processInput, err := p.buildProcessInput(ctx, text)
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
	clog.FromContext(ctx).Info("ingest url: complete", "url", url, "theme", result.ThemePath, "slug", result.Slug)

	return node, nil
}

func (p *PipelineIngester) buildProcessInput(ctx context.Context, text string) (llm.ProcessInput, error) {
	tree, err := p.store.ReadTree(ctx, p.basePath)
	if err != nil {
		return llm.ProcessInput{}, errors.Errorf("read tree: %w", err)
	}

	themes := collectThemes(tree)
	keywords, err := p.collectKeywords(ctx)
	if err != nil {
		clog.FromContext(ctx).Warn("ingest: failed to collect keywords, proceeding without them", "error", err)
	}

	return llm.ProcessInput{
		Text:             text,
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
	if result.Title != "" {
		frontmatter["title"] = result.Title
		frontmatter["aliases"] = []string{result.Title}
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
