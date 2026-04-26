package index

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIProvider_Embed_WhenSingleText_ExpectSingleVector(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req embeddingRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "text-embedding-3-small", req.Model)
		inputs, ok := req.Input.([]interface{})
		require.True(t, ok)
		require.Len(t, inputs, 1)
		assert.Equal(t, "hello world", inputs[0])

		resp := embeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{Embedding: []float32{0.1, 0.2, 0.3}, Index: 0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	provider := NewAPIProvider(server.URL, "test-key", "text-embedding-3-small")
	result, err := provider.Embed(context.Background(), []string{"hello world"})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, result[0])
}

func TestAPIProvider_Embed_WhenMultipleTexts_ExpectMultipleVectors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := embeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{Embedding: []float32{0.1, 0.2}, Index: 0},
				{Embedding: []float32{0.3, 0.4}, Index: 1},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	provider := NewAPIProvider(server.URL, "key", "model")
	result, err := provider.Embed(context.Background(), []string{"a", "b"})

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, []float32{0.1, 0.2}, result[0])
	assert.Equal(t, []float32{0.3, 0.4}, result[1])
}

func TestAPIProvider_Embed_WhenAPIError_ExpectError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := apiError{}
		resp.Error.Message = "invalid api key"
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	provider := NewAPIProvider(server.URL, "bad-key", "model")
	_, err := provider.Embed(context.Background(), []string{"text"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "embedding API error")
	assert.Contains(t, err.Error(), "invalid api key")
}

func TestAPIProvider_Embed_WhenCountMismatch_ExpectError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := embeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{Embedding: []float32{0.1}, Index: 0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer server.Close()

	provider := NewAPIProvider(server.URL, "key", "model")
	_, err := provider.Embed(context.Background(), []string{"a", "b"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "returned 1 results, expected 2")
}

func TestAPIProvider_Embed_WhenContextCancelled_ExpectError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := NewAPIProvider("http://localhost", "key", "model")
	_, err := provider.Embed(ctx, []string{"text"})

	require.Error(t, err)
}
