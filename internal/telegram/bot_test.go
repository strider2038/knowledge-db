package telegram //nolint:testpackage // internal methods handleUpdate and buildConfirmation require package-level access

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

type mockIngester struct {
	node *kb.Node
	err  error
}

func (m *mockIngester) IngestText(_ context.Context, _ string) (*kb.Node, error) {
	return m.node, m.err
}

func (m *mockIngester) IngestURL(_ context.Context, _ string) (*kb.Node, error) {
	return m.node, m.err
}

func TestHandleUpdate_WhenAuthorizedUser_ExpectIngestCalled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var ingestCalled atomic.Bool
	mock := &mockIngester{
		node: &kb.Node{
			Path:       "go/concurrency/goroutine-basics",
			Annotation: "test",
			Metadata: map[string]any{
				"type":     "note",
				"keywords": []any{"go"},
			},
		},
	}

	bot := NewBot("en", 12345, mock)
	bot.buffer.ttl = time.Millisecond

	origIngester := bot.ingester
	bot.ingester = &callTrackingIngester{inner: origIngester, called: &ingestCalled}

	u := update{
		UpdateID: 1,
		Message: &message{
			MessageID: 1,
			Text:      "test message",
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	assert.Eventually(t, ingestCalled.Load, 100*time.Millisecond, time.Millisecond)
}

func TestHandleUpdate_WhenUnauthorizedUser_ExpectIngestNotCalled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var ingestCalled atomic.Bool
	mock := &callTrackingIngester{
		inner:  &mockIngester{},
		called: &ingestCalled,
	}

	bot := NewBot("token", 99999, mock)

	u := update{
		UpdateID: 1,
		Message: &message{
			MessageID: 1,
			Text:      "test",
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	assert.False(t, ingestCalled.Load())
}

func TestHandleUpdate_WhenNoOwnerIDSet_ExpectAllUsersAllowed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var ingestCalled atomic.Bool
	bot := NewBot("token", 0, &callTrackingIngester{
		inner:  &mockIngester{node: &kb.Node{Path: "go/test", Metadata: map[string]any{}}},
		called: &ingestCalled,
	})
	bot.buffer.ttl = time.Millisecond

	u := update{
		UpdateID: 1,
		Message: &message{
			MessageID: 1,
			Text:      "hello",
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 99999},
		},
	}

	bot.handleUpdate(ctx, u)

	assert.Eventually(t, ingestCalled.Load, 100*time.Millisecond, time.Millisecond)
}

func TestHandleUpdate_WhenCaptionOnly_ExpectIngestCalled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var ingestCalled atomic.Bool
	var capturedText atomic.Value
	mock := &mockIngester{
		node: &kb.Node{
			Path:       "go/notes/forwarded-caption",
			Annotation: "forwarded",
			Metadata:   map[string]any{"type": "note"},
		},
	}
	bot := NewBot("token", 12345, &callTrackingIngesterWithCapture{
		inner:        mock,
		called:       &ingestCalled,
		capturedText: &capturedText,
	})
	bot.buffer.ttl = time.Millisecond

	u := update{
		UpdateID: 1,
		Message: &message{
			MessageID: 1,
			Caption:   "forwarded media caption",
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	assert.Eventually(t, ingestCalled.Load, 100*time.Millisecond, time.Millisecond)
	assert.Eventually(t, func() bool {
		v := capturedText.Load()

		s, ok := v.(string)

		return ok && s == "forwarded media caption"
	}, 100*time.Millisecond, time.Millisecond)
}

func TestHandleUpdate_WhenCommentThenForward_ExpectMergedIngest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedText atomic.Value
	var ingestCalled atomic.Bool
	bot := NewBot("token", 12345, &callTrackingIngesterWithCapture{
		inner:        &mockIngester{node: &kb.Node{Path: "ai/notes/test", Metadata: map[string]any{}}},
		capturedText: &capturedText,
		called:       &ingestCalled,
	})

	// Шаг 1: пользователь пишет комментарий как обычный текст
	commentUpdate := update{
		UpdateID: 1,
		Message: &message{
			MessageID: 1,
			Text:      "Сохрани как заметку со ссылкой",
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}
	bot.handleUpdate(ctx, commentUpdate)

	// Шаг 2: следом пересылает сообщение — бот должен объединить
	forwardOrigin := json.RawMessage(`{"type":"user"}`)
	fwdUpdate := update{
		UpdateID: 2,
		Message: &message{
			MessageID:     2,
			Text:          "Профессор Кнут... https://cs.stanford.edu/~knuth/papers/claude-cycles.pdf",
			ForwardOrigin: forwardOrigin,
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}
	bot.handleUpdate(ctx, fwdUpdate)

	expected := "Инструкции пользователя: Сохрани как заметку со ссылкой\nПересланное сообщение: Профессор Кнут... https://cs.stanford.edu/~knuth/papers/claude-cycles.pdf"
	assert.Eventually(t, func() bool {
		v := capturedText.Load()

		s, ok := v.(string)

		return ok && s == expected
	}, 100*time.Millisecond, time.Millisecond)
}

func TestHandleUpdate_WhenReplyToForwardedMessage_ExpectMergedIngest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedText atomic.Value
	var ingestCalled atomic.Bool
	bot := NewBot("token", 12345, &callTrackingIngesterWithCapture{
		inner:        &mockIngester{node: &kb.Node{Path: "ai/notes/test", Metadata: map[string]any{}}},
		capturedText: &capturedText,
		called:       &ingestCalled,
	})

	forwarded := &message{
		MessageID: 10,
		Text:      "https://habr.com/article",
		From: &struct {
			ID int64 `json:"id"`
		}{ID: 12345},
	}
	reply := update{
		UpdateID: 2,
		Message: &message{
			MessageID:      11,
			Text:           "сохрани в ai/notes",
			ReplyToMessage: forwarded,
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, reply)

	v := capturedText.Load()
	require.NotNil(t, v)
	s, ok := v.(string)
	require.True(t, ok)
	assert.Equal(t, "Инструкции пользователя: сохрани в ai/notes\nПересланное сообщение: https://habr.com/article", s)
}

func TestHandleUpdate_WhenForwardedWithoutReply_ExpectBufferedThenFlushed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var ingestCalled atomic.Bool
	bot := NewBot("token", 12345, &callTrackingIngester{
		inner:  &mockIngester{node: &kb.Node{Path: "go/test", Metadata: map[string]any{}}},
		called: &ingestCalled,
	})

	forwardOrigin := json.RawMessage(`{"type":"user"}`)
	u := update{
		UpdateID: 1,
		Message: &message{
			MessageID:     20,
			Text:          "forwarded text",
			ForwardOrigin: forwardOrigin,
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	// сразу после handleUpdate ingestion ещё не вызван — сообщение в буфере
	assert.False(t, ingestCalled.Load(), "ingest should not be called immediately, message is buffered")
}

func TestHandleUpdate_WhenForwardedWithoutReply_ExpectFlushedAfterTTL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedText atomic.Value
	var ingestCalled atomic.Bool
	bot := NewBot("token", 12345, &callTrackingIngesterWithCapture{
		inner:        &mockIngester{node: &kb.Node{Path: "go/test", Metadata: map[string]any{}}},
		capturedText: &capturedText,
		called:       &ingestCalled,
	})
	bot.buffer.ttl = time.Millisecond

	forwardOrigin := json.RawMessage(`{"type":"user"}`)
	u := update{
		UpdateID: 1,
		Message: &message{
			MessageID:     21,
			Text:          "forwarded after ttl",
			ForwardOrigin: forwardOrigin,
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	// Таймер вызывает processIngest асинхронно; sendMessage к API может занять время
	assert.Eventually(t, func() bool {
		v := capturedText.Load()

		s, ok := v.(string)

		return ok && s == "forwarded after ttl"
	}, 15*time.Second, 50*time.Millisecond)
}

func TestHandleUpdate_WhenReplyArrivesDuringBuffer_ExpectSingleIngest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedTexts []string
	var mu sync.Mutex
	bot := NewBot("token", 12345, &captureAllIngester{
		node: &kb.Node{Path: "go/test", Metadata: map[string]any{}},
		capture: func(text string) {
			mu.Lock()
			defer mu.Unlock()
			capturedTexts = append(capturedTexts, text)
		},
	})

	forwardOrigin := json.RawMessage(`{"type":"user"}`)
	fwd := update{
		UpdateID: 1,
		Message: &message{
			MessageID:     30,
			Text:          "https://example.com",
			ForwardOrigin: forwardOrigin,
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}
	bot.handleUpdate(ctx, fwd)

	// сразу следует reply — бот должен удалить из буфера и обработать как пару
	replyUpdate := update{
		UpdateID: 2,
		Message: &message{
			MessageID: 31,
			Text:      "сохрани",
			ReplyToMessage: &message{
				MessageID: 30,
				Text:      "https://example.com",
			},
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}
	bot.handleUpdate(ctx, replyUpdate)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, capturedTexts, 1, "exactly one ingest call expected")
	assert.Equal(t, "Инструкции пользователя: сохрани\nПересланное сообщение: https://example.com", capturedTexts[0])
}

func TestCombineForwardWithComment_ExpectLabels(t *testing.T) {
	t.Parallel()
	result := combineForwardWithComment("сохрани в go/tips", "https://go.dev/blog/article")
	assert.Equal(t, "Инструкции пользователя: сохрани в go/tips\nПересланное сообщение: https://go.dev/blog/article", result)
}

func TestBuildConfirmation_WhenNodeWithKeywords_ExpectFormattedMessage(t *testing.T) {
	t.Parallel()
	bot := &Bot{}
	node := &kb.Node{
		Path:       "go/concurrency/goroutine-leak",
		Annotation: "Article about goroutine leaks",
		Metadata: map[string]any{
			"type":     "article",
			"keywords": []any{"goroutines", "memory"},
		},
	}

	msg := bot.buildConfirmation(node)

	require.Contains(t, msg, "go/concurrency/goroutine-leak")
	assert.Contains(t, msg, "article")
	assert.Contains(t, msg, "goroutines")
}

type callTrackingIngester struct {
	inner  ingestion.Ingester
	called *atomic.Bool
}

func (c *callTrackingIngester) IngestText(ctx context.Context, text string) (*kb.Node, error) {
	c.called.Store(true)

	return c.inner.IngestText(ctx, text)
}

func (c *callTrackingIngester) IngestURL(ctx context.Context, url string) (*kb.Node, error) {
	c.called.Store(true)

	return c.inner.IngestURL(ctx, url)
}

type callTrackingIngesterWithCapture struct {
	inner        ingestion.Ingester
	called       *atomic.Bool
	capturedText *atomic.Value
}

func (c *callTrackingIngesterWithCapture) IngestText(ctx context.Context, text string) (*kb.Node, error) {
	c.called.Store(true)
	c.capturedText.Store(text)

	return c.inner.IngestText(ctx, text)
}

func (c *callTrackingIngesterWithCapture) IngestURL(ctx context.Context, url string) (*kb.Node, error) {
	c.called.Store(true)

	return c.inner.IngestURL(ctx, url)
}

type captureAllIngester struct {
	node    *kb.Node
	capture func(text string)
}

func (c *captureAllIngester) IngestText(_ context.Context, text string) (*kb.Node, error) {
	c.capture(text)

	return c.node, nil
}

func (c *captureAllIngester) IngestURL(_ context.Context, _ string) (*kb.Node, error) {
	return c.node, nil
}
