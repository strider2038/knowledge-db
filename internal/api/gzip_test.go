package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/strider2038/knowledge-db/internal/api"
)

func TestGzip_WhenAcceptEncodingGzip_CompressesResponse(t *testing.T) {
	t.Parallel()

	handler := api.Gzip(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Repeat("x", 1000)))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	// Сжатый ответ должен быть меньше 1000 байт
	if rec.Body.Len() >= 1000 {
		t.Errorf("expected compressed body < 1000 bytes, got %d", rec.Body.Len())
	}
}

func TestGzip_WhenNoAcceptEncoding_PassesThrough(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("x", 100)
	handler := api.Gzip(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("expected no Content-Encoding, got %q", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.String() != body {
		t.Errorf("expected uncompressed body %d bytes, got %d", len(body), rec.Body.Len())
	}
}

func TestGzip_WhenRangeHeader_PassesThrough(t *testing.T) {
	t.Parallel()

	body := "partial content"
	handler := api.Gzip(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Range", "bytes=0-10")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Errorf("expected no Content-Encoding for Range request, got %q", rec.Header().Get("Content-Encoding"))
	}
}
