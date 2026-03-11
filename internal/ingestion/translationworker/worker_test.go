package translationworker_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationqueue"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationworker"
	"github.com/strider2038/knowledge-db/internal/kb"
)

type mockTranslator struct {
	mu    sync.Mutex
	calls int
}

func (m *mockTranslator) Translate(_ context.Context, _ string) (string, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	return "", nil
}

func (m *mockTranslator) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.calls
}

func TestWorker_WhenTranslationExists_ExpectIdempotent(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	fs := afero.NewMemMapFs()
	basePath := "/data"
	themePath := "go"
	slug := "test-article"
	_ = fs.MkdirAll(filepath.Join(basePath, themePath), 0o755)

	articleContent := `---
keywords: [go]
type: article
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
title: "Test"
---

Content in English.`
	translationContent := `---
translation_of: test-article
lang: ru
---

Контент на русском.`

	_ = afero.WriteFile(fs, filepath.Join(basePath, themePath, slug+".md"), []byte(articleContent), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(basePath, themePath, slug+".ru.md"), []byte(translationContent), 0o644)

	store := kb.NewStore(fs)
	queue := translationqueue.New(10)
	mockTr := &mockTranslator{}
	committer := &git.NoopGitCommitter{}

	worker := translationworker.New(queue, store, mockTr, committer, basePath)

	go func() {
		_ = worker.Run(ctx)
	}()

	_, _ = queue.Enqueue(themePath, slug)

	require.Eventually(t, func() bool {
		s, _ := queue.GetStatus(themePath, slug)

		return s == translationqueue.StatusDone
	}, 2*time.Second, 50*time.Millisecond)

	assert.Equal(t, 0, mockTr.callCount(), "translator must not be called when translation already exists")
}
