package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/pkg/tracing"
)

func TestRequestLoggingMiddleware_WhenHandlerReturnsStatus_CapturesStatus(t *testing.T) {
	t.Parallel()

	handler := tracing.Middleware(
		api.LoggingMiddleware(
			api.RequestLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			})),
		),
	)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestRequestLoggingMiddleware_WhenHandlerSucceeds_ReturnsOK(t *testing.T) {
	t.Parallel()

	handler := tracing.Middleware(
		api.LoggingMiddleware(
			api.RequestLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("ok"))
			})),
		),
	)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %q", rec.Body.String())
	}
}
