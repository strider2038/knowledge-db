package git

import (
	"context"
	"sync"
)

// commitJob — задача для сериализованного выполнения CommitNode.
type commitJob struct {
	ctx      context.Context
	nodePath string
	message  string
	errCh    chan error
}

// SerializedGitCommitter — обёртка над GitCommitter, сериализующая вызовы CommitNode.
// Все CommitNode выполняются последовательно в одной горутине, что исключает
// гонки при одновременном git add/commit/push из pipeline и воркера перевода.
type SerializedGitCommitter struct {
	inner GitCommitter
	jobs  chan commitJob
	wg    sync.WaitGroup
}

// NewSerializedGitCommitter создаёт SerializedGitCommitter и запускает воркер.
func NewSerializedGitCommitter(inner GitCommitter) *SerializedGitCommitter {
	const queueSize = 64
	s := &SerializedGitCommitter{
		inner: inner,
		jobs:  make(chan commitJob, queueSize),
	}
	s.wg.Add(1)
	go s.worker()

	return s
}

func (s *SerializedGitCommitter) worker() {
	defer s.wg.Done()
	for job := range s.jobs {
		err := s.inner.CommitNode(job.ctx, job.nodePath, job.message)
		select {
		case job.errCh <- err:
		default:
			// Caller may have given up (context cancelled)
		}
	}
}

// CommitNode ставит задачу в очередь и ждёт её выполнения.
func (s *SerializedGitCommitter) CommitNode(ctx context.Context, nodePath, message string) error {
	errCh := make(chan error, 1)
	job := commitJob{ctx: ctx, nodePath: nodePath, message: message, errCh: errCh}

	select {
	case s.jobs <- job:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Sync делегирует вызов underlying committer.
func (s *SerializedGitCommitter) Sync(ctx context.Context) error {
	return s.inner.Sync(ctx)
}
