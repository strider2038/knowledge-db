package translationworker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	"github.com/pior/runnable"

	"github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion/translation"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationqueue"
	"github.com/strider2038/knowledge-db/internal/kb"
)

const translateTimeout = 5 * time.Minute

// Worker — фоновой воркер перевода статей.
type Worker struct {
	queue     *translationqueue.Queue
	store     *kb.Store
	translator translation.Translator
	committer  git.GitCommitter
	basePath   string
}

// New создаёт Worker.
func New(
	queue *translationqueue.Queue,
	store *kb.Store,
	translator translation.Translator,
	committer git.GitCommitter,
	basePath string,
) *Worker {
	return &Worker{
		queue:     queue,
		store:     store,
		translator: translator,
		committer:  committer,
		basePath:   basePath,
	}
}

// Run обрабатывает очередь переводов до отмены контекста.
func (w *Worker) Run(ctx context.Context) error {
	logger := clog.FromContext(ctx)
	logger.Info("translation worker: started")
	defer logger.Info("translation worker: stopped")

	ch := w.queue.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case key, ok := <-ch:
			if !ok {
				return nil
			}
			w.processOne(ctx, key)
		}
	}
}

func (w *Worker) processOne(ctx context.Context, key translationqueue.ArticleKey) {
	logger := clog.FromContext(ctx)
	themePath := key.ThemePath
	slug := key.Slug

	w.queue.SetInProgress(themePath, slug)
	logger.Info("translation: start", "theme", themePath, "slug", slug)

	start := time.Now()
	err := w.doTranslate(ctx, themePath, slug)
	if err != nil {
		errMsg := err.Error()
		w.queue.SetFailed(themePath, slug, errMsg)
		clog.Errorf(ctx, "translation: failure (theme=%s slug=%s duration_ms=%d): %w", themePath, slug, time.Since(start).Milliseconds(), err)

		return
	}

	w.queue.SetDone(themePath, slug)
	logger.Info("translation: success", "theme", themePath, "slug", slug, "duration_ms", time.Since(start).Milliseconds())
}

func (w *Worker) doTranslate(ctx context.Context, themePath, slug string) error {
	nodePath := themePath + "/" + slug
	node, err := w.store.GetNode(ctx, w.basePath, nodePath)
	if err != nil {
		return errors.Errorf("get node: %w", err)
	}

	// Проверяем, что перевод ещё нужен (мог быть создан параллельно).
	translationPath := themePath + "/" + slug + ".ru"
	if _, err := w.store.GetNode(ctx, w.basePath, translationPath); err == nil {
		return nil // перевод уже есть
	}

	if !translation.NeedsTranslation(node.Content) {
		return nil // контент уже на русском
	}

	translateCtx, cancel := context.WithTimeout(ctx, translateTimeout)
	defer cancel()

	translated, err := w.translator.Translate(translateCtx, node.Content)
	if err != nil {
		return errors.Errorf("translate: %w", err)
	}

	translationFrontmatter := buildTranslationFrontmatter(node, slug)
	contentWithLink := translated
	if !strings.HasSuffix(contentWithLink, fmt.Sprintf("[[%s|Original]]", slug)) {
		contentWithLink = strings.TrimSuffix(contentWithLink, "\n") + "\n\n" + fmt.Sprintf("[[%s|Original]]", slug) + "\n"
	}

	if err := w.store.CreateTranslationFile(ctx, w.basePath, themePath, slug, "ru", translationFrontmatter, contentWithLink); err != nil {
		return errors.Errorf("create translation file: %w", err)
	}
	if err := w.store.AppendTranslationsToOriginal(ctx, w.basePath, themePath, slug, slug+".ru"); err != nil {
		return errors.Errorf("append translations to original: %w", err)
	}

	translationFilePath := filepath.Join(w.basePath, filepath.FromSlash(themePath), slug+".ru.md")
	if err := w.committer.CommitNode(ctx, translationFilePath, fmt.Sprintf("add: %s/%s.ru (translation)", themePath, slug)); err != nil {
		return errors.Errorf("git commit translation: %w", err)
	}
	originalFilePath := filepath.Join(w.basePath, filepath.FromSlash(themePath), slug+".md")
	if err := w.committer.CommitNode(ctx, originalFilePath, fmt.Sprintf("add: %s/%s (translation link)", themePath, slug)); err != nil {
		return errors.Errorf("git commit original: %w", err)
	}

	return nil
}

func buildTranslationFrontmatter(node *kb.Node, slug string) map[string]any {
	meta := node.Metadata
	if meta == nil {
		meta = make(map[string]any)
	}
	fm := map[string]any{
		"translation_of": slug,
		"lang":           "ru",
		"type":           "article",
	}
	if v, ok := meta["keywords"]; ok {
		fm["keywords"] = v
	}
	if v, ok := meta["created"]; ok {
		fm["created"] = v
	}
	if v, ok := meta["updated"]; ok {
		fm["updated"] = v
	}
	if v, ok := meta["annotation"]; ok {
		fm["annotation"] = v
	}
	if v, ok := meta["title"]; ok {
		fm["title"] = v
	}
	if v, ok := meta["aliases"]; ok {
		fm["aliases"] = v
	}
	if v, ok := meta["source_url"]; ok {
		fm["source_url"] = v
	}
	if v, ok := meta["source_date"]; ok {
		fm["source_date"] = v
	}
	if v, ok := meta["source_author"]; ok {
		fm["source_author"] = v
	}
	return fm
}

// Ensure Worker implements runnable.Runnable.
var _ runnable.Runnable = (*Worker)(nil)
