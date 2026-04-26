package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/index"
)

func setupTestHandlerWithIndex(t *testing.T) (http.Handler, *index.IndexStore) {
	t.Helper()

	tmp := t.TempDir()
	h := api.NewHandler(tmp, &ingestion.StubIngester{})

	store, err := index.NewIndexStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	provider := index.NewAPIProvider("http://localhost", "key", "model")
	embeddingCfg := config.Embedding{
		Enabled:   true,
		APIKey:    "key",
		APIURL:    "http://localhost",
		Model:     "text-embedding-3-small",
		ChatModel: "gpt-4o",
	}
	h.SetIndexComponents(store, nil, provider, embeddingCfg)

	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux, store
}

func setupTestHandlerWithoutIndex(t *testing.T) http.Handler {
	t.Helper()

	tmp := t.TempDir()
	h := api.NewHandler(tmp, &ingestion.StubIngester{})

	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func TestGetIndexStatus_WhenIndexAvailable_ExpectOK(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)

	resp := apitest.HandleGET(t, handler, "/api/index/status")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("status").IsString().EqualTo("ready")
		json.Node("total_nodes").IsNumber().EqualTo(0)
		json.Node("total_chunks").IsNumber().EqualTo(0)
		json.Node("embedding_model").IsString().EqualTo("text-embedding-3-small")
	})
}

func TestGetIndexStatus_WhenIndexUnavailable_Expect503(t *testing.T) {
	t.Parallel()

	handler := setupTestHandlerWithoutIndex(t)

	resp := apitest.HandleGET(t, handler, "/api/index/status")

	resp.HasCode(503)
}

func TestPostIndexRebuild_WhenIndexUnavailable_Expect503(t *testing.T) {
	t.Parallel()

	handler := setupTestHandlerWithoutIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/index/rebuild", nil)

	resp.HasCode(503)
}

func TestPostChat_WhenIndexUnavailable_Expect503(t *testing.T) {
	t.Parallel()

	handler := setupTestHandlerWithoutIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{"message":"hello"}`))

	resp.HasCode(503)
}

func TestPostChat_WhenEmptyMessage_Expect400(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{"message":""}`))

	resp.IsBadRequest()
}

func TestPostChat_WhenMissingMessage_Expect400(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{}`))

	resp.IsBadRequest()
}
