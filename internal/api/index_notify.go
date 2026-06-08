package api

import (
	"context"
	"strings"

	"github.com/strider2038/knowledge-db/internal/index"
)

func (h *Handler) notifyIndexNodesChanged(ctx context.Context, paths ...string) {
	if h.syncWorker == nil {
		return
	}
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		h.syncWorker.Send(ctx, index.SingleNodeEvent{Path: path})
	}
}

func (h *Handler) notifyIndexGitSyncReconcile(ctx context.Context) {
	if h.syncWorker == nil {
		return
	}
	h.syncWorker.Send(ctx, index.GitSyncDiffEvent{})
}
