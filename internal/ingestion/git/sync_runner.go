package git

import (
	"context"
	"time"

	"github.com/muonsoft/clog"
)

// GitSyncRunner — runnable для периодической синхронизации с remote.
type GitSyncRunner struct {
	committer GitCommitter
	interval  time.Duration
}

// NewGitSyncRunner создаёт GitSyncRunner.
func NewGitSyncRunner(committer GitCommitter, interval time.Duration) *GitSyncRunner {
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	return &GitSyncRunner{committer: committer, interval: interval}
}

// Run выполняет периодический Sync с заданным интервалом.
// Завершается при отмене контекста.
func (r *GitSyncRunner) Run(ctx context.Context) error {
	logger := clog.FromContext(ctx)
	logger.Info("git sync runner: started")
	defer logger.Info("git sync runner: stopped")

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.committer.Sync(ctx); err != nil {
				clog.Errorf(ctx, "git sync: %w", err)
			} else {
				logger.Info("git sync: completed")
			}
		}
	}
}
