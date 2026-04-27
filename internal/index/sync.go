package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	"github.com/pior/runnable"
	"github.com/spf13/afero"

	"github.com/strider2038/knowledge-db/internal/kb"
)

// SyncEvent — событие синхронизации индекса.
type SyncEvent interface {
	syncEventType()
}

// SingleNodeEvent — индексация одной ноды.
type SingleNodeEvent struct {
	Path string
}

func (SingleNodeEvent) syncEventType() {}

// GitSyncDiffEvent — diff после git pull (пока заглушка, использует FullReconcile).
type GitSyncDiffEvent struct{}

func (GitSyncDiffEvent) syncEventType() {}

// FullReconcileEvent — полная сверка индекса с FS.
type FullReconcileEvent struct{}

func (FullReconcileEvent) syncEventType() {}

// ManualRebuildEvent — полная перестройка индекса.
type ManualRebuildEvent struct{}

func (ManualRebuildEvent) syncEventType() {}

// SyncWorker синхронизирует индекс с git-репозиторием.
type SyncWorker struct {
	store     *IndexStore
	provider  EmbeddingProvider
	kbStore   *kb.Store
	dataPath  string
	model     string
	events    chan SyncEvent
	rateLimit time.Duration
}

// NewSyncWorker создаёт SyncWorker.
func NewSyncWorker(store *IndexStore, provider EmbeddingProvider, dataPath, model string, rateLimit time.Duration) *SyncWorker {
	return &SyncWorker{
		store:     store,
		provider:  provider,
		kbStore:   kb.NewStore(afero.NewOsFs()),
		dataPath:  dataPath,
		model:     model,
		events:    make(chan SyncEvent, 100),
		rateLimit: rateLimit,
	}
}

// Send отправляет событие в очередь синхронизации.
func (w *SyncWorker) Send(event SyncEvent) {
	select {
	case w.events <- event:
		clog.Debug(context.Background(), "sync: event queued", "event", fmt.Sprintf("%T", event))
	default:
		clog.Info(context.Background(), "sync: event dropped (queue full)", "event", fmt.Sprintf("%T", event))
	}
}

// Run запускает воркер синхронизации до отмены контекста.
func (w *SyncWorker) Run(ctx context.Context) error {
	logger := clog.FromContext(ctx)
	logger.Info("index sync worker: started")
	defer logger.Info("index sync worker: stopped")

	logger.Info("sync: performing initial full reconcile")
	w.fullReconcile(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-w.events:
			w.handleEvent(ctx, event)
		}
	}
}

func (w *SyncWorker) handleEvent(ctx context.Context, event SyncEvent) {
	logger := clog.FromContext(ctx)
	logger.Debug("sync: event received", "event", fmt.Sprintf("%T", event))

	switch e := event.(type) {
	case SingleNodeEvent:
		logger.Debug("sync: processing single node", "path", e.Path)
		w.processSingleNode(ctx, e.Path)
	case GitSyncDiffEvent:
		logger.Info("sync: git diff event, starting full reconcile")
		w.fullReconcile(ctx)
	case FullReconcileEvent:
		logger.Info("sync: full reconcile event")
		w.fullReconcile(ctx)
	case ManualRebuildEvent:
		logger.Info("sync: manual rebuild event")
		w.manualRebuild(ctx)
	}
}

func (w *SyncWorker) processSingleNode(ctx context.Context, path string) {
	logger := clog.FromContext(ctx)
	logger.Debug("sync: processing node", "path", path)

	node, err := w.kbStore.GetNode(ctx, w.dataPath, path)
	if err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			logger.Info("sync: node deleted from index", "path", path)
			if err := w.store.DeleteNode(ctx, path); err != nil {
				clog.Errorf(ctx, "sync: delete node from index: %w", err)
			}

			return
		}
		clog.Errorf(ctx, "sync: get node %s: %w", path, err)

		return
	}

	contentHash := computeContentHash(node)
	bodyHash := computeBodyHash(node.Content)

	existing, err := w.store.GetNodeByPath(ctx, path)
	if err == nil && existing.ContentHash == contentHash && existing.BodyHash == bodyHash {
		logger.Debug("sync: node unchanged, skipping", "path", path)

		return
	}

	logger.Debug("sync: node changed or new, indexing", "path", path)

	nodeType, _ := node.Metadata["type"].(string)
	embeddingText := buildNodeEmbeddingText(node, nodeType)
	vectors, err := w.provider.Embed(ctx, []string{embeddingText})
	if err != nil {
		clog.Errorf(ctx, "sync: embed node %s: %w", path, err)

		return
	}

	embID, err := w.store.InsertEmbedding(ctx, vectors[0], w.model)
	if err != nil {
		clog.Errorf(ctx, "sync: insert embedding for %s: %w", path, err)

		return
	}

	if err := w.store.UpsertNode(ctx, path, contentHash, bodyHash, embID); err != nil {
		clog.Errorf(ctx, "sync: upsert node %s: %w", path, err)

		return
	}

	logger.Info("sync: node indexed", "path", path, "type", nodeType)

	if nodeType == "article" && strings.TrimSpace(node.Content) != "" {
		clog.Debug(ctx, "sync: processing article chunks", "path", path)
		w.processChunks(ctx, path, node.Content)
	}

	w.rateLimitWait(ctx)
}

func (w *SyncWorker) processChunks(ctx context.Context, nodePath, body string) {
	logger := clog.FromContext(ctx)
	textChunks := ChunkText(body)
	if len(textChunks) == 0 {
		return
	}

	clog.Info(ctx, "sync: embedding article chunks", "path", nodePath, "chunks", len(textChunks))

	texts := make([]string, len(textChunks))
	for i, c := range textChunks {
		texts[i] = c.Heading + "\n" + c.Content
	}

	vectors, err := w.provider.Embed(ctx, texts)
	if err != nil {
		clog.Errorf(ctx, "sync: embed chunks for %s: %w", nodePath, err)

		return
	}

	chunks := make([]Chunk, len(textChunks))
	for i, tc := range textChunks {
		embID, err := w.store.InsertEmbedding(ctx, vectors[i], w.model)
		if err != nil {
			clog.Errorf(ctx, "sync: insert chunk embedding for %s: %w", nodePath, err)

			return
		}
		chunks[i] = Chunk{
			NodePath:    nodePath,
			ChunkIndex:  i,
			Heading:     tc.Heading,
			Content:     tc.Content,
			EmbeddingID: embID,
		}
	}

	if err := w.store.UpsertChunks(ctx, nodePath, chunks); err != nil {
		clog.Errorf(ctx, "sync: upsert chunks for %s: %w", nodePath, err)
	}

	logger.Info("sync: article chunks indexed", "path", nodePath, "chunks", len(chunks))
}

func (w *SyncWorker) fullReconcile(ctx context.Context) {
	logger := clog.FromContext(ctx)
	logger.Info("sync: full reconcile started")

	startTime := time.Now()
	allNodes, err := w.kbStore.ListAllNodes(ctx, w.dataPath)
	if err != nil {
		clog.Errorf(ctx, "sync: list all nodes: %w", err)

		return
	}

	indexed, err := w.store.ListAllIndexed(ctx)
	if err != nil {
		clog.Errorf(ctx, "sync: list indexed nodes: %w", err)

		return
	}

	indexedSet := make(map[string]IndexedNode, len(indexed))
	for _, n := range indexed {
		indexedSet[n.Path] = n
	}

	fsSet := make(map[string]bool, len(allNodes))
	indexedCount := 0
	for _, n := range allNodes {
		select {
		case <-ctx.Done():
			logger.Warn("sync: full reconcile cancelled by context")
			return
		default:
		}

		fsSet[n.Path] = true
		w.processSingleNode(ctx, n.Path)
		w.rateLimitWait(ctx)
		indexedCount++
	}

	for path := range indexedSet {
		select {
		case <-ctx.Done():
			logger.Warn("sync: full reconcile cancelled by context (stale deletion phase)")
			return
		default:
		}

		if !fsSet[path] {
			logger.Debug("sync: deleting stale node", "path", path)
			if err := w.store.DeleteNode(ctx, path); err != nil {
				clog.Errorf(ctx, "sync: delete stale node %s: %w", path, err)
			}
		}
	}

	logger.Info("sync: full reconcile complete",
		"total_nodes", len(allNodes),
		"stale_deleted", len(indexedSet)-indexedCount,
		"duration_ms", time.Since(startTime).Milliseconds(),
	)
}

func (w *SyncWorker) manualRebuild(ctx context.Context) {
	logger := clog.FromContext(ctx)
	logger.Info("sync: manual rebuild started")

	startTime := time.Now()
	if err := w.store.ClearAll(ctx); err != nil {
		clog.Errorf(ctx, "sync: clear index: %w", err)

		return
	}

	logger.Debug("sync: index cleared, starting full reconcile")

	w.fullReconcile(ctx)

	logger.Info("sync: manual rebuild complete", "duration_ms", time.Since(startTime).Milliseconds())
}

func (w *SyncWorker) rateLimitWait(ctx context.Context) {
	if w.rateLimit <= 0 {
		return
	}

	select {
	case <-ctx.Done():
	case <-time.After(w.rateLimit):
	}
}

func computeContentHash(node *kb.Node) string {
	title, _ := node.Metadata["title"].(string)
	annotation, _ := node.Metadata["annotation"].(string)
	keywords := extractKeywords(node.Metadata)
	nodeType, _ := node.Metadata["type"].(string)

	data := fmt.Sprintf("%s|%s|%s|%s", title, annotation, strings.Join(keywords, ","), nodeType)
	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}

func computeBodyHash(content string) string {
	hash := sha256.Sum256([]byte(content))

	return hex.EncodeToString(hash[:])
}

func buildNodeEmbeddingText(node *kb.Node, nodeType string) string {
	title, _ := node.Metadata["title"].(string)
	annotation, _ := node.Metadata["annotation"].(string)
	keywords := extractKeywords(node.Metadata)

	parts := []string{title, annotation}
	parts = append(parts, keywords...)

	if nodeType == "note" && strings.TrimSpace(node.Content) != "" {
		parts = append(parts, node.Content)
	}

	return strings.Join(parts, " ")
}

func extractKeywords(meta map[string]any) []string {
	raw, ok := meta["keywords"]
	if !ok {
		return nil
	}

	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}

		return result
	default:
		return nil
	}
}

// Ensure SyncWorker implements runnable.Runnable.
var _ runnable.Runnable = (*SyncWorker)(nil)
