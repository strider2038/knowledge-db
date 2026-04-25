package git

import "context"

// NoopGitCommitter — GitCommitter, который ничего не делает (git отключён).
type NoopGitCommitter struct{}

// CommitNode — no-op, возвращает nil.
func (NoopGitCommitter) CommitNode(_ context.Context, _, _ string) error { return nil }

// Sync — no-op, возвращает nil.
func (NoopGitCommitter) Sync(_ context.Context) error { return nil }

// Status — no-op, возвращает статус без изменений.
func (NoopGitCommitter) Status(_ context.Context) (*GitStatus, error) {
	return &GitStatus{HasChanges: false, ChangedFiles: 0}, nil
}

// CommitAll — no-op, возвращает nil.
func (NoopGitCommitter) CommitAll(_ context.Context, _ string) error { return nil }

// DiffStat — no-op, возвращает пустую строку.
func (NoopGitCommitter) DiffStat(_ context.Context) (string, error) { return "", nil }
