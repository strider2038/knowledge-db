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

func (m *mockIngester) IngestText(_ context.Context, _ ingestion.IngestRequest) (*kb.Node, error) {
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

	bot := NewBot("en", 12345, mock, "")
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

	bot := NewBot("token", 99999, mock, "")

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
	}, "")
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
	}, "")
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
	}, "")

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
	}, "")

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
	}, "")

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
	}, "")
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
	}, "")

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

func TestEntitiesToMarkdown_WhenBold_ExpectMarkdown(t *testing.T) {
	t.Parallel()
	text := "Hello world"
	entities := []messageEntity{{Type: "bold", Offset: 0, Length: 5}} // "Hello" = 5 chars
	result := entitiesToMarkdown(text, entities)
	assert.Equal(t, "**Hello** world", result)
}

func TestEntitiesToMarkdown_WhenItalic_ExpectMarkdown(t *testing.T) {
	t.Parallel()
	text := "Hello world"
	entities := []messageEntity{{Type: "italic", Offset: 6, Length: 5}}
	result := entitiesToMarkdown(text, entities)
	assert.Equal(t, "Hello *world*", result)
}

func TestEntitiesToMarkdown_WhenCode_ExpectBackticks(t *testing.T) {
	t.Parallel()
	text := "Use the fmt package"
	entities := []messageEntity{{Type: "code", Offset: 8, Length: 3}}
	result := entitiesToMarkdown(text, entities)
	assert.Equal(t, "Use the `fmt` package", result)
}

func TestEntitiesToMarkdown_WhenTextLink_ExpectMarkdownLink(t *testing.T) {
	t.Parallel()
	text := "Click here"
	entities := []messageEntity{{Type: "text_link", Offset: 0, Length: 10, URL: "https://example.com"}}
	result := entitiesToMarkdown(text, entities)
	assert.Equal(t, "[Click here](https://example.com)", result)
}

func TestEntitiesToMarkdown_WhenMultipleEntities_ExpectAllConverted(t *testing.T) {
	t.Parallel()
	text := "Bold and italic"
	entities := []messageEntity{
		{Type: "bold", Offset: 0, Length: 4},
		{Type: "italic", Offset: 9, Length: 6},
	}
	result := entitiesToMarkdown(text, entities)
	assert.Equal(t, "**Bold** and *italic*", result)
}

func TestEntitiesToMarkdown_WhenNoEntities_ExpectOriginalText(t *testing.T) {
	t.Parallel()
	text := "Plain text"
	result := entitiesToMarkdown(text, nil)
	assert.Equal(t, "Plain text", result)
}

func TestEntitiesToMarkdown_WhenStrikethrough_ExpectGFM(t *testing.T) {
	t.Parallel()
	text := "deleted text"
	entities := []messageEntity{{Type: "strikethrough", Offset: 0, Length: 12}}
	result := entitiesToMarkdown(text, entities)
	assert.Equal(t, "~~deleted text~~", result)
}

func TestEntitiesToMarkdown_WhenBlockquote_ExpectQuotePrefix(t *testing.T) {
	t.Parallel()
	text := "Quoted line"
	entities := []messageEntity{{Type: "blockquote", Offset: 0, Length: 11}}
	result := entitiesToMarkdown(text, entities)
	assert.Equal(t, "> Quoted line", result)
}

func TestExtractTextWithFormatting_WhenBoldInMessage_ExpectMarkdown(t *testing.T) {
	t.Parallel()
	msg := &message{
		Text:     "Important note",
		Entities: []messageEntity{{Type: "bold", Offset: 0, Length: 9}}, // "Important" = 9 chars
	}
	result := extractTextWithFormatting(msg)
	assert.Equal(t, "**Important** note", result)
}

func TestExtractTextWithFormatting_WhenCaptionWithEntities_ExpectMarkdown(t *testing.T) {
	t.Parallel()
	msg := &message{
		Caption:         "Photo with caption",                                     // plain text from Telegram
		CaptionEntities: []messageEntity{{Type: "italic", Offset: 11, Length: 7}}, // "caption" = 7 chars
	}
	result := extractTextWithFormatting(msg)
	assert.Equal(t, "Photo with *caption*", result)
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
	assert.NotContains(t, msg, "Открыть на сайте")
}

func TestWebNodePageURL_WhenBaseAndPath_ExpectEscapedSegments(t *testing.T) {
	t.Parallel()
	// Сегменты пути — как в браузере: `/` между темами, спецсимволы внутри сегмента экранируются.
	assert.Equal(t, "https://kb.example/node/go/concurrency/note", webNodePageURL("https://kb.example", "go/concurrency/note"))
}

func TestBuildConfirmation_WhenWebBaseURLSet_ExpectOpenLink(t *testing.T) {
	t.Parallel()
	bot := &Bot{webPublicBaseURL: "https://kb.example"}
	node := &kb.Node{
		Path:     "topic/a & b",
		Metadata: map[string]any{"type": "note"},
	}
	msg := bot.buildConfirmation(node)
	assert.Contains(t, msg, `href="https://kb.example/node/topic/a%20%26%20b"`)
	assert.Contains(t, msg, "Открыть на сайте")
	assert.Contains(t, msg, "topic/a &amp; b")
}

type callTrackingIngester struct {
	inner  ingestion.Ingester
	called *atomic.Bool
}

func (c *callTrackingIngester) IngestText(ctx context.Context, req ingestion.IngestRequest) (*kb.Node, error) {
	c.called.Store(true)

	return c.inner.IngestText(ctx, req)
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

func (c *callTrackingIngesterWithCapture) IngestText(ctx context.Context, req ingestion.IngestRequest) (*kb.Node, error) {
	c.called.Store(true)
	c.capturedText.Store(req.Text)

	return c.inner.IngestText(ctx, req)
}

func (c *callTrackingIngesterWithCapture) IngestURL(ctx context.Context, url string) (*kb.Node, error) {
	c.called.Store(true)

	return c.inner.IngestURL(ctx, url)
}

type captureAllIngester struct {
	node    *kb.Node
	capture func(text string)
}

func (c *captureAllIngester) IngestText(_ context.Context, req ingestion.IngestRequest) (*kb.Node, error) {
	c.capture(req.Text)

	return c.node, nil
}

func (c *captureAllIngester) IngestURL(_ context.Context, _ string) (*kb.Node, error) {
	return c.node, nil
}

func TestParseForwardOrigin_WhenChannel_ExpectURLAndAuthor(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"type":"channel","date":1234567890,"chat":{"id":-1001234567890,"username":"testchannel","title":"Test Channel"},"message_id":42}`)
	url, author := parseForwardOrigin(raw)

	assert.Equal(t, "https://t.me/testchannel/42", url)
	assert.Equal(t, "Test Channel", author)
}

func TestParseForwardOrigin_WhenChannelWithoutUsername_ExpectCLink(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"type":"channel","date":1234567890,"chat":{"id":-1001234567890,"title":"Private Channel"},"message_id":99}`)
	url, author := parseForwardOrigin(raw)

	assert.Equal(t, "https://t.me/c/1234567890/99", url)
	assert.Equal(t, "Private Channel", author)
}

func TestParseForwardOrigin_WhenUser_ExpectAuthorOnly(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"type":"user","date":1234567890,"sender_user":{"id":123,"username":"johndoe","first_name":"John","last_name":"Doe"}}`)
	url, author := parseForwardOrigin(raw)

	assert.Empty(t, url)
	assert.Equal(t, "@johndoe", author)
}

func TestParseForwardOrigin_WhenHiddenUser_ExpectAuthorOnly(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"type":"hidden_user","date":1234567890,"sender_user_name":"Anonymous"}`)
	url, author := parseForwardOrigin(raw)

	assert.Empty(t, url)
	assert.Equal(t, "Anonymous", author)
}

func TestHandleUpdate_WhenForwardedMessage_ExpectIngestWithSourceMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedReq ingestion.IngestRequest
	var captureMu sync.Mutex
	mock := &captureRequestIngester{
		node: &kb.Node{Path: "go/test", Metadata: map[string]any{}},
		capture: func(req ingestion.IngestRequest) {
			captureMu.Lock()
			capturedReq = req
			captureMu.Unlock()
		},
	}
	bot := NewBot("token", 12345, mock, "")
	bot.buffer.ttl = time.Millisecond

	u := update{
		UpdateID: 1,
		Message: &message{
			MessageID:     1,
			Text:          "Forwarded content",
			ForwardOrigin: json.RawMessage(`{"type":"channel","date":1234567890,"chat":{"id":-1001234567890,"username":"techchannel","title":"Tech Channel"},"message_id":10}`),
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	require.Eventually(t, func() bool {
		captureMu.Lock()
		defer captureMu.Unlock()

		return capturedReq.Text != ""
	}, 100*time.Millisecond, time.Millisecond)

	captureMu.Lock()
	defer captureMu.Unlock()
	assert.Equal(t, "Forwarded content", capturedReq.Text)
	assert.Equal(t, "https://t.me/techchannel/10", capturedReq.SourceURL)
	assert.Equal(t, "Tech Channel", capturedReq.SourceAuthor)
}

type captureRequestIngester struct {
	node    *kb.Node
	capture func(ingestion.IngestRequest)
}

func (c *captureRequestIngester) IngestText(_ context.Context, req ingestion.IngestRequest) (*kb.Node, error) {
	c.capture(req)

	return c.node, nil
}

func (c *captureRequestIngester) IngestURL(_ context.Context, _ string) (*kb.Node, error) {
	return c.node, nil
}
