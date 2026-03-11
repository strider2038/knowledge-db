package git_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gittool "github.com/strider2038/knowledge-db/internal/ingestion/git"
)

type concurrentTrackingCommitter struct {
	delay   time.Duration
	active  atomic.Int32
	maxSeen atomic.Int32
}

func (c *concurrentTrackingCommitter) CommitNode(_ context.Context, _, _ string) error {
	n := c.active.Add(1)
	if prev := c.maxSeen.Load(); n > prev {
		c.maxSeen.Store(n)
	}
	time.Sleep(c.delay)
	c.active.Add(-1)

	return nil
}

func (c *concurrentTrackingCommitter) Sync(_ context.Context) error { return nil }

func TestSerializedGitCommitter_WhenConcurrentCalls_ExpectSequential(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	inner := &concurrentTrackingCommitter{delay: 20 * time.Millisecond}
	committer := gittool.NewSerializedGitCommitter(inner)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := committer.CommitNode(ctx, "/path/node.md", "test")
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), inner.maxSeen.Load(),
		"at most one CommitNode must run at a time")
}
