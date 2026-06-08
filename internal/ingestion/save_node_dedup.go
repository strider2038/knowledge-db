package ingestion

import (
	"context"
	"database/sql"
	stderrors "errors"
	"maps"
	"path/filepath"
	"strings"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/ingestion/llm"
	"github.com/strider2038/knowledge-db/internal/kb"
)

var errNoExistingNodeForIngest = errors.New("no existing node for ingest dedup")

func (p *PipelineIngester) resolveExistingNode(ctx context.Context, result *llm.ProcessResult) (*kb.NodeFile, error) {
	if nodeID := strings.TrimSpace(result.NodeID); nodeID != "" {
		node, err := p.store.GetNodeByID(ctx, p.basePath, nodeID)
		if err != nil {
			if stderrors.Is(err, kb.ErrNodeNotFound) {
				return nil, errNoExistingNodeForIngest
			}

			return nil, errors.Errorf("resolve existing node by id: %w", err)
		}

		return p.loadExistingNodeFile(ctx, node.Path, node.ID, "by id")
	}
	if p.indexStore == nil {
		return nil, errNoExistingNodeForIngest
	}
	sourceURL := strings.TrimSpace(result.SourceURL)
	if sourceURL == "" {
		return nil, errNoExistingNodeForIngest
	}
	norm := kb.NormalizeSourceURLForDedup(sourceURL)
	match, err := p.indexStore.FindBySourceURL(ctx, norm)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, errNoExistingNodeForIngest
		}

		return nil, errors.Errorf("resolve existing node: %w", err)
	}

	return p.loadExistingNodeFile(ctx, match.Path, match.NodeID, "by source_url")
}

func (p *PipelineIngester) loadExistingNodeFile(ctx context.Context, nodePath, nodeID, lookup string) (*kb.NodeFile, error) {
	file, err := p.store.GetNodeFile(ctx, p.basePath, nodePath)
	if err != nil {
		if stderrors.Is(err, kb.ErrNodeNotFound) {
			clog.Warn(ctx, "ingest: indexed node missing on disk, will create new",
				"path", nodePath, "node_id", nodeID, "lookup", lookup)
			p.deleteStaleIndexNode(ctx, nodeID)

			return nil, errNoExistingNodeForIngest
		}

		return nil, errors.Errorf("resolve existing node %s: %w", lookup, err)
	}

	return file, nil
}

func (p *PipelineIngester) deleteStaleIndexNode(ctx context.Context, nodeID string) {
	if p.indexStore == nil || strings.TrimSpace(nodeID) == "" {
		return
	}
	if err := p.indexStore.DeleteNodeByID(ctx, nodeID); err != nil {
		clog.Errorf(ctx, "ingest: delete stale index node: %w", err)
	}
}

func (p *PipelineIngester) updateExistingNode(ctx context.Context, existing *kb.NodeFile, result *llm.ProcessResult) (*kb.Node, error) {
	frontmatter := maps.Clone(existing.Frontmatter)
	p.applyResultToExistingFrontmatter(ctx, frontmatter, result)

	node, err := p.store.UpdateNode(ctx, p.basePath, existing.Path, kb.UpdateNodeParams{
		Frontmatter: frontmatter,
		Content:     result.Content,
	})
	if err != nil {
		return nil, errors.Errorf("update existing node: %w", err)
	}
	clog.Info(ctx, "ingest: node updated", "path", existing.Path, "id", node.ID)

	nodeMdPath := basePathJoin(p.basePath, existing.Path)
	if err := p.committer.CommitNode(ctx, nodeMdPath, "update: "+existing.Path); err != nil {
		clog.Errorf(ctx, "update existing node: git commit failed: %w", err)
	}

	return node, nil
}

func basePathJoin(basePath, nodePath string) string {
	return filepath.Join(basePath, filepath.FromSlash(nodePath)+".md")
}
