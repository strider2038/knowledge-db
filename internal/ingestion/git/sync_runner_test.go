package git_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	igit "github.com/strider2038/knowledge-db/internal/ingestion/git"
)

type noopGitCommitter struct{}

func (noopGitCommitter) CommitNode(context.Context, string, string) error { return nil }
func (noopGitCommitter) CommitAll(context.Context, string) error          { return nil }
func (noopGitCommitter) Status(context.Context) (*igit.GitStatus, error)  { return &igit.GitStatus{}, nil }
func (noopGitCommitter) DiffStat(context.Context) (string, error)         { return "", nil }
func (noopGitCommitter) Sync(context.Context) error                       { return nil }

func TestGitSyncRunner_WhenSyncSucceeds_ExpectOnSyncedCallback(t *testing.T) {
	t.Parallel()
	synced := make(chan struct{}, 1)
	runner := igit.NewGitSyncRunner(noopGitCommitter{}, 20*time.Millisecond, func(context.Context) {
		select {
		case synced <- struct{}{}:
		default:
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	select {
	case <-synced:
	case <-time.After(2 * time.Second):
		t.Fatal("onSynced callback was not called")
	}

	cancel()
	require.NoError(t, <-done)
}
