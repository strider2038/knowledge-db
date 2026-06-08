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

// NodeMovedEvent — обновление path в индексе без потери embeddings.
type NodeMovedEvent struct {
	NodeID  string
	OldPath string
	NewPath string
}

func (NodeMovedEvent) syncEventType() {}

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
	store            Store
	provider         EmbeddingProvider
	kbStore          *kb.Store
	dataPath         string
	model            string
	events           chan SyncEvent
	rateLimit        time.Duration
	periodicInterval time.Duration
	fullReconcileFn  func(context.Context)
}

func NewSyncWorker(store Store, provider EmbeddingProvider, dataPath, model string, rateLimit time.Duration) *SyncWorker {
	return &SyncWorker{
		store:            store,
		provider:         provider,
		kbStore:          kb.NewStore(afero.NewOsFs()),
		dataPath:         dataPath,
		model:            model,
		events:           make(chan SyncEvent, 100),
		rateLimit:        rateLimit,
		periodicInterval: 24 * time.Hour,
	}
}

// Send отправляет событие в очередь синхронизации.
func (w *SyncWorker) Send(ctx context.Context, event SyncEvent) {
	select {
	case w.events <- event:
		clog.Debug(ctx, "sync: event queued", "event", fmt.Sprintf("%T", event))
	default:
		clog.Info(ctx, "sync: event dropped (queue full)", "event", fmt.Sprintf("%T", event))
	}
}

// Run запускает воркер синхронизации до отмены контекста.
func (w *SyncWorker) Run(ctx context.Context) error {
	logger := clog.FromContext(ctx)
	logger.Info("index sync worker: started")
	defer logger.Info("index sync worker: stopped")

	logger.Info("sync: performing initial full reconcile")
	w.runFullReconcile(ctx)

	var periodicTicker *time.Ticker
	if w.periodicInterval > 0 {
		periodicTicker = time.NewTicker(w.periodicInterval)
		defer periodicTicker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-w.events:
			w.handleEvent(ctx, event)
		case <-tickerChan(periodicTicker):
			logger.Info("sync: periodic reconcile event")
			w.runFullReconcile(ctx)
		}
	}
}

func (w *SyncWorker) ProcessSingleNodeForTest(ctx context.Context, path string) {
	w.processSingleNode(ctx, path)
}

// ManualRebuild очищает индекс и выполняет полную переиндексацию всех нод из FS.
func (w *SyncWorker) ManualRebuild(ctx context.Context) error {
	logger := clog.FromContext(ctx)
	logger.Info("sync: manual rebuild started")

	startTime := time.Now()
	if err := w.store.ClearAll(ctx); err != nil {
		return errors.Errorf("sync: clear index: %w", err)
	}

	logger.Debug("sync: index cleared, starting full reconcile")

	w.runFullReconcile(ctx)

	logger.Info("sync: manual rebuild complete", "duration_ms", time.Since(startTime).Milliseconds())

	return nil
}

func (w *SyncWorker) handleEvent(ctx context.Context, event SyncEvent) {
	logger := clog.FromContext(ctx)
	logger.Debug("sync: event received", "event", fmt.Sprintf("%T", event))

	switch e := event.(type) {
	case SingleNodeEvent:
		logger.Debug("sync: processing single node", "path", e.Path)
		w.processSingleNode(ctx, e.Path)
	case NodeMovedEvent:
		logger.Info("sync: node moved", "node_id", e.NodeID, "old_path", e.OldPath, "new_path", e.NewPath)
		if err := w.store.UpdateNodePath(ctx, e.NodeID, e.NewPath); err != nil {
			clog.Errorf(ctx, "sync: update node path: %w", err)
		}
	case GitSyncDiffEvent:
		logger.Info("sync: git diff event, starting full reconcile")
		w.runFullReconcile(ctx)
	case FullReconcileEvent:
		logger.Info("sync: full reconcile event")
		w.runFullReconcile(ctx)
	case ManualRebuildEvent:
		logger.Info("sync: manual rebuild event")
		if err := w.ManualRebuild(ctx); err != nil {
			clog.Errorf(ctx, "sync: manual rebuild: %w", err)
		}
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

	nodeID := kb.NodeIDFromMetadata(node.Metadata)
	if nodeID == "" {
		logger.Warn("sync: skip node without id", "path", path)

		return
	}

	contentHash := computeContentHash(node)
	bodyHash := computeBodyHash(node.Content)

	existing, err := w.store.GetNodeByID(ctx, nodeID)
	if err == nil && existing.ContentHash == contentHash && existing.BodyHash == bodyHash && existing.Path == path {
		logger.Debug("sync: node unchanged, skipping", "path", path, "node_id", nodeID)

		return
	}

	logger.Debug("sync: node changed or new, indexing", "path", path, "node_id", nodeID)

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

	if err := w.store.UpsertNode(ctx, nodeID, path, contentHash, bodyHash, embID); err != nil {
		clog.Errorf(ctx, "sync: upsert node %s: %w", path, err)

		return
	}
	searchDoc := buildNodeSearchDocument(node, nodeType)
	searchDoc.NodeID = nodeID
	if err := w.store.UpsertNodeSearch(ctx, searchDoc); err != nil {
		clog.Errorf(ctx, "sync: upsert node search %s: %w", path, err)

		return
	}
	if sourceURL, _ := node.Metadata["source_url"].(string); sourceURL != "" {
		if err := w.store.UpsertNodeSourceURL(ctx, nodeID, kb.NormalizeSourceURLForDedup(sourceURL)); err != nil {
			clog.Errorf(ctx, "sync: upsert node source url %s: %w", path, err)
		}
	}

	logger.Info("sync: node indexed", "path", path, "node_id", nodeID, "type", nodeType)

	if shouldChunkBody(node, nodeType) {
		clog.Debug(ctx, "sync: processing body chunks", "path", path)
		w.processChunks(ctx, nodeID, path, node.Content)
	} else if err := w.store.DeleteChunks(ctx, nodeID, path); err != nil {
		clog.Errorf(ctx, "sync: delete chunks for %s: %w", path, err)
	}

	w.rateLimitWait(ctx)
}

func shouldChunkBody(node *kb.Node, nodeType string) bool {
	return (nodeType == "article" || shouldIndexBody(node, nodeType)) && strings.TrimSpace(node.Content) != ""
}

func (w *SyncWorker) processChunks(ctx context.Context, nodeID, nodePath, body string) {
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
			NodeID:      nodeID,
			NodePath:    nodePath,
			ChunkIndex:  i,
			Heading:     tc.Heading,
			Content:     tc.Content,
			EmbeddingID: embID,
		}
	}

	if err := w.store.UpsertChunks(ctx, nodeID, nodePath, chunks); err != nil {
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

func (w *SyncWorker) runFullReconcile(ctx context.Context) {
	if w.fullReconcileFn != nil {
		w.fullReconcileFn(ctx)

		return
	}
	w.fullReconcile(ctx)
}

func tickerChan(t *time.Ticker) <-chan time.Time {
	if t == nil {
		return nil
	}

	return t.C
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
	sourceKind, _ := node.Metadata["source_kind"].(string)
	contentProfile, _ := node.Metadata["content_profile"].(string)

	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s", title, annotation, strings.Join(keywords, ","), nodeType, sourceKind, contentProfile)
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

	if shouldIndexBody(node, nodeType) {
		parts = append(parts, node.Content)
	}

	return strings.Join(parts, " ")
}

func buildNodeSearchDocument(node *kb.Node, nodeType string) NodeSearchDocument {
	title, _ := node.Metadata["title"].(string)
	annotation, _ := node.Metadata["annotation"].(string)
	sourceURL, _ := node.Metadata["source_url"].(string)
	sourceKind, _ := node.Metadata["source_kind"].(string)
	contentProfile, _ := node.Metadata["content_profile"].(string)
	body := ""
	if shouldIndexBody(node, nodeType) {
		body = node.Content
	}

	return NodeSearchDocument{
		NodeID:          kb.NodeIDFromMetadata(node.Metadata),
		Path:            node.Path,
		Title:           title,
		Type:            nodeType,
		Aliases:         extractStringList(node.Metadata, "aliases"),
		Annotation:      annotation,
		Keywords:        extractKeywords(node.Metadata),
		SourceURL:       sourceURL,
		SourceKind:      sourceKind,
		ContentProfile:  contentProfile,
		ManualProcessed: kb.ManualProcessedEffective(node.Metadata),
		Body:            body,
	}
}

func shouldIndexBody(node *kb.Node, nodeType string) bool {
	if strings.TrimSpace(node.Content) == "" {
		return false
	}
	if nodeType == "note" {
		return true
	}
	if nodeType != "link" {
		return false
	}
	contentProfile, _ := node.Metadata["content_profile"].(string)

	return contentProfile != "" && contentProfile != string(kb.ContentProfileLinkBookmark)
}

func extractKeywords(meta map[string]any) []string {
	return extractStringList(meta, "keywords")
}

func extractStringList(meta map[string]any, key string) []string {
	raw, ok := meta[key]
	if !ok {
		return nil
	}

	switch v := raw.(type) {
	case []string:
		return v
	case []any:
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
