package ingestion

import (
	"context"
	"strings"
)

// NodesChangedNotifier is called after pipeline persistence changes node files on disk.
type NodesChangedNotifier func(ctx context.Context, paths ...string)

func (p *PipelineIngester) SetNodesChangedNotifier(notifier NodesChangedNotifier) {
	p.nodesChanged = notifier
}

func (p *PipelineIngester) notifyNodesChanged(ctx context.Context, paths ...string) {
	if p.nodesChanged == nil {
		return
	}
	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	if len(unique) == 0 {
		return
	}
	p.nodesChanged(ctx, unique...)
}
