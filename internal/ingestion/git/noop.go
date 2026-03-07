package git

import "context"

// NoopGitCommitter — GitCommitter, который ничего не делает (git отключён).
type NoopGitCommitter struct{}

// CommitNode — no-op, возвращает nil.
func (NoopGitCommitter) CommitNode(_ context.Context, _, _ string) error { return nil }

// Sync — no-op, возвращает nil.
func (NoopGitCommitter) Sync(_ context.Context) error { return nil }
