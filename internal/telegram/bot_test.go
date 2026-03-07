package telegram //nolint:testpackage // internal methods handleUpdate and buildConfirmation require package-level access

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

	ingestCalled := false
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
	var capturedMessages []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/botten/sendMessage" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if text, ok := body["text"].(string); ok {
				capturedMessages = append(capturedMessages, text)
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	bot := &Bot{
		token:    "token",
		ownerID:  12345,
		dataPath: "/data",
		ingester: mock,
	}
	bot.token = "en"

	// Override the ingester to track calls
	origIngester := bot.ingester
	bot.ingester = &callTrackingIngester{inner: origIngester, called: &ingestCalled}

	u := update{
		UpdateID: 1,
		Message: &struct {
			Text    string `json:"text"`
			Caption string `json:"caption"`
			Chat    *struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			From *struct {
				ID int64 `json:"id"`
			} `json:"from"`
		}{
			Text: "test message",
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	assert.True(t, ingestCalled)
}

func TestHandleUpdate_WhenUnauthorizedUser_ExpectIngestNotCalled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ingestCalled := false
	mock := &callTrackingIngester{
		inner:  &mockIngester{},
		called: &ingestCalled,
	}

	bot := &Bot{
		token:    "token",
		ownerID:  99999,
		dataPath: "/data",
		ingester: mock,
	}

	u := update{
		UpdateID: 1,
		Message: &struct {
			Text    string `json:"text"`
			Caption string `json:"caption"`
			Chat    *struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			From *struct {
				ID int64 `json:"id"`
			} `json:"from"`
		}{
			Text: "test",
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	assert.False(t, ingestCalled)
}

func TestHandleUpdate_WhenNoOwnerIDSet_ExpectAllUsersAllowed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ingestCalled := false
	bot := &Bot{
		token:    "token",
		ownerID:  0,
		dataPath: "/data",
		ingester: &callTrackingIngester{
			inner: &mockIngester{
				node: &kb.Node{Path: "go/test", Metadata: map[string]any{}},
			},
			called: &ingestCalled,
		},
	}

	u := update{
		UpdateID: 1,
		Message: &struct {
			Text    string `json:"text"`
			Caption string `json:"caption"`
			Chat    *struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			From *struct {
				ID int64 `json:"id"`
			} `json:"from"`
		}{
			Text: "hello",
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 99999},
		},
	}

	bot.handleUpdate(ctx, u)

	assert.True(t, ingestCalled)
}

func TestHandleUpdate_WhenCaptionOnly_ExpectIngestCalled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ingestCalled := false
	var capturedText string
	mock := &mockIngester{
		node: &kb.Node{
			Path:       "go/notes/forwarded-caption",
			Annotation: "forwarded",
			Metadata:   map[string]any{"type": "note"},
		},
	}
	bot := &Bot{
		token:    "token",
		ownerID:  12345,
		dataPath: "/data",
		ingester: &callTrackingIngesterWithCapture{
			inner:        mock,
			called:       &ingestCalled,
			capturedText: &capturedText,
		},
	}

	u := update{
		UpdateID: 1,
		Message: &struct {
			Text    string `json:"text"`
			Caption string `json:"caption"`
			Chat    *struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			From *struct {
				ID int64 `json:"id"`
			} `json:"from"`
		}{
			Text:    "",
			Caption: "forwarded media caption",
			Chat: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
			From: &struct {
				ID int64 `json:"id"`
			}{ID: 12345},
		},
	}

	bot.handleUpdate(ctx, u)

	assert.True(t, ingestCalled)
	assert.Equal(t, "forwarded media caption", capturedText)
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
	called *bool
}

func (c *callTrackingIngester) IngestText(ctx context.Context, text string) (*kb.Node, error) {
	*c.called = true

	return c.inner.IngestText(ctx, text)
}

func (c *callTrackingIngester) IngestURL(ctx context.Context, url string) (*kb.Node, error) {
	*c.called = true

	return c.inner.IngestURL(ctx, url)
}

type callTrackingIngesterWithCapture struct {
	inner        ingestion.Ingester
	called       *bool
	capturedText *string
}

func (c *callTrackingIngesterWithCapture) IngestText(ctx context.Context, text string) (*kb.Node, error) {
	*c.called = true
	*c.capturedText = text

	return c.inner.IngestText(ctx, text)
}

func (c *callTrackingIngesterWithCapture) IngestURL(ctx context.Context, url string) (*kb.Node, error) {
	*c.called = true

	return c.inner.IngestURL(ctx, url)
}
