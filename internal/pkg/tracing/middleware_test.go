package tracing_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/pkg/tracing"
)

func TestMiddleware_WhenHeaderPresent_UsesRequestID(t *testing.T) {
	t.Parallel()

	requestID := uuid.Must(uuid.NewV4())
	var capturedID uuid.UUID
	handler := tracing.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = tracing.RequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)
	req.Header.Set(tracing.Header, requestID.String())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, requestID, capturedID)
}

func TestMiddleware_WhenHeaderMissing_GeneratesRequestID(t *testing.T) {
	t.Parallel()

	var capturedID uuid.UUID
	handler := tracing.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = tracing.RequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req, err2 := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err2)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEqual(t, uuid.Nil, capturedID)
}

func TestMiddleware_WhenHeaderInvalid_GeneratesRequestID(t *testing.T) {
	t.Parallel()

	var capturedID uuid.UUID
	handler := tracing.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = tracing.RequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req, err2 := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err2)
	req.Header.Set(tracing.Header, "not-a-valid-uuid")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEqual(t, uuid.Nil, capturedID)
}
