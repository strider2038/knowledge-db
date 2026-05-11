package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/chat"
	"github.com/strider2038/knowledge-db/internal/index"
	"github.com/strider2038/knowledge-db/internal/ingestion"
)

func setupTestHandlerWithIndex(t *testing.T) (http.Handler, *index.IndexStore) {
	t.Helper()

	tmp := t.TempDir()
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	chatStore, err := chat.NewStore(filepath.Join(tmp, "chat.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = chatStore.Close() })
	h.SetChatStore(chatStore)

	store, err := index.NewIndexStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	provider := &testEmbeddingProvider{vector: []float32{1, 0, 0}}
	worker := index.NewSyncWorker(store, provider, tmp, "model", 0)
	embeddingCfg := config.Embedding{
		Enabled:   true,
		APIKey:    "key",
		APIURL:    "http://localhost",
		Model:     "text-embedding-3-small",
		ChatModel: "gpt-4o",
	}
	h.SetIndexComponents(store, worker, provider, embeddingCfg)

	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux, store
}

type testEmbeddingProvider struct {
	vector []float32
}

func (p *testEmbeddingProvider) Embed(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = p.vector
	}

	return result, nil
}

func setupTestHandlerWithoutIndex(t *testing.T) http.Handler {
	t.Helper()

	tmp := t.TempDir()
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	chatStore, err := chat.NewStore(filepath.Join(tmp, "chat.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = chatStore.Close() })
	h.SetChatStore(chatStore)

	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func createChatSessionID(t *testing.T, handler http.Handler) string {
	t.Helper()
	resp := apitest.HandlePOST(t, handler, "/api/chats", strings.NewReader(`{"title":"t"}`))
	resp.IsOK()
	var data struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(resp.Recorder().Body.Bytes(), &data))
	require.NotEmpty(t, data.ID)
	return data.ID
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
		json.Node("keyword_index").IsString()
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

func TestPostIndexRebuild_WhenIndexAvailable_Expect202(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/index/rebuild", nil)

	resp.HasCode(202)
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("status").IsString().EqualTo("rebuild started")
	})
}

func TestPostChat_WhenIndexUnavailable_Expect503(t *testing.T) {
	t.Parallel()

	handler := setupTestHandlerWithoutIndex(t)
	sessionID := createChatSessionID(t, handler)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{"session_id":"`+sessionID+`","message":"hello"}`))

	resp.HasCode(503)
}

func TestPostChat_WhenEmptyMessage_Expect400(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)
	sessionID := createChatSessionID(t, handler)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{"session_id":"`+sessionID+`","message":""}`))

	resp.IsBadRequest()
}

func TestPostChat_WhenMissingMessage_Expect400(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)
	sessionID := createChatSessionID(t, handler)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{"session_id":"`+sessionID+`"}`))

	resp.IsBadRequest()
}

func TestPostChat_WhenMissingSessionID_Expect400(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{"message":"hello"}`))

	resp.IsBadRequest()
}

func TestPostChat_WhenUnknownSession_Expect404(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{"session_id":"missing-session","message":"hello"}`))

	resp.HasCode(404)
}

func TestPostChat_WhenSourcePathsExcludeMatches_ExpectInsufficientData(t *testing.T) {
	t.Parallel()

	handler, store := setupTestHandlerWithIndex(t)
	seedSearchIndex(t, store)
	sessionID := createChatSessionID(t, handler)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{
		"session_id":"`+sessionID+`",
		"message":"sqlite",
		"source_paths":["missing/path"]
	}`))

	resp.IsOK()
	body := resp.Recorder().Body.String()
	assert.Contains(t, body, `"sources": []`)
	assert.Contains(t, body, "Недостаточно данных")
}

func TestPostChat_WhenRAGModeWithoutContext_ExpectKnowledgeBaseFallback(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)
	sessionID := createChatSessionID(t, handler)

	resp := apitest.HandlePOST(t, handler, "/api/chat", strings.NewReader(`{
		"session_id":"`+sessionID+`",
		"message":"Что есть в базе про RAG?"
	}`))

	resp.IsOK()
	body := resp.Recorder().Body.String()
	assert.Contains(t, body, `"sources": []`)
	assert.Contains(t, body, "В базе знаний не найдено релевантной информации по запросу.")
}

func TestPostSearch_WhenSuccess_ExpectResults(t *testing.T) {
	t.Parallel()

	handler, store := setupTestHandlerWithIndex(t)
	seedSearchIndex(t, store)

	resp := apitest.HandlePOST(t, handler, "/api/search", strings.NewReader(`{"query":"sqlite","limit":5}`))

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("query").IsString().EqualTo("sqlite")
		json.Node("mode").IsString().EqualTo("search")
		json.Node("total").IsNumber().EqualTo(2)
		json.Node("results", 0, "path").IsString().EqualTo("articles/sqlite")
		json.Node("results", 0, "fragments", 0, "snippet").IsString()
		json.Node("meta", "keyword_index").IsString()
	})
}

func TestPostSearch_WhenEmptyQuery_Expect400(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/search", strings.NewReader(`{"query":""}`))

	resp.IsBadRequest()
}

func TestPostSearch_WhenTypeFilter_ExpectFiltered(t *testing.T) {
	t.Parallel()

	handler, store := setupTestHandlerWithIndex(t)
	seedSearchIndex(t, store)

	resp := apitest.HandlePOST(t, handler, "/api/search", strings.NewReader(`{"query":"local","type":["note"],"limit":5}`))

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("total").IsNumber().EqualTo(1)
		json.Node("results", 0, "path").IsString().EqualTo("notes/local")
		json.Node("results", 0, "type").IsString().EqualTo("note")
	})
}

func TestPostSearch_WhenIndexUnavailable_Expect503(t *testing.T) {
	t.Parallel()

	handler := setupTestHandlerWithoutIndex(t)

	resp := apitest.HandlePOST(t, handler, "/api/search", strings.NewReader(`{"query":"sqlite"}`))

	resp.HasCode(503)
}

func TestChatsCRUD_WhenValidFlow_ExpectSuccess(t *testing.T) {
	t.Parallel()

	handler, _ := setupTestHandlerWithIndex(t)
	sessionID := createChatSessionID(t, handler)

	listResp := apitest.HandleGET(t, handler, "/api/chats")
	listResp.IsOK()
	assert.Contains(t, listResp.Recorder().Body.String(), sessionID)

	getResp := apitest.HandleGET(t, handler, "/api/chats/"+sessionID)
	getResp.IsOK()
	assert.Contains(t, getResp.Recorder().Body.String(), sessionID)

	patchResp := apitest.HandlePATCH(t, handler, "/api/chats/"+sessionID, strings.NewReader(`{"title":"Renamed"}`))
	patchResp.IsOK()
	assert.Contains(t, patchResp.Recorder().Body.String(), "Renamed")

	deleteResp := apitest.HandleDELETE(t, handler, "/api/chats/"+sessionID)
	deleteResp.IsOK()

	getMissing := apitest.HandleGET(t, handler, "/api/chats/"+sessionID)
	getMissing.HasCode(404)
}

func seedSearchIndex(t *testing.T, store *index.IndexStore) {
	t.Helper()

	ctx := context.Background()
	embID, err := store.InsertEmbedding(ctx, []float32{1, 0, 0}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, "articles/sqlite", "h1", "bh1", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, index.NodeSearchDocument{
		Path:       "articles/sqlite",
		Title:      "SQLite",
		Type:       "article",
		Annotation: "database retrieval",
		Keywords:   []string{"sqlite"},
	}))
	chunkEmbID, err := store.InsertEmbedding(ctx, []float32{1, 0, 0}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertChunks(ctx, "articles/sqlite", []index.Chunk{
		{NodePath: "articles/sqlite", ChunkIndex: 0, Heading: "Search", Content: "sqlite local retrieval chunk", EmbeddingID: chunkEmbID},
	}))

	embID, err = store.InsertEmbedding(ctx, []float32{0, 1, 0}, "model")
	require.NoError(t, err)
	require.NoError(t, store.UpsertNode(ctx, "notes/local", "h2", "bh2", embID))
	require.NoError(t, store.UpsertNodeSearch(ctx, index.NodeSearchDocument{
		Path:       "notes/local",
		Title:      "Local Note",
		Type:       "note",
		Annotation: "local workflow",
		Keywords:   []string{"local"},
		Body:       "local workflow body",
	}))
}
