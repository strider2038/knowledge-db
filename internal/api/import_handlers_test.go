package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

func setupImportTestHandler(t *testing.T, ingester ingestion.Ingester) http.Handler {
	t.Helper()
	dataPath := t.TempDir()
	uploadsDir := t.TempDir()
	h := api.NewHandlerWithUploads(dataPath, uploadsDir, ingester)
	mux, err := api.NewMux(h)
	require.NoError(t, err)

	return mux
}

const validTelegramJSON = `{
	"id": 123,
	"name": "Test",
	"type": "personal_chat",
	"messages": [
		{"id": 1, "type": "message", "date_unixtime": "1557861184", "from": "Alice", "text": "First"},
		{"id": 2, "type": "message", "date_unixtime": "1557861185", "from": "Bob", "text": "Second"}
	]
}`

func TestImportTelegramCreate_WhenValidJSON_ExpectSession(t *testing.T) {
	t.Parallel()
	handler := setupImportTestHandler(t, &ingestion.StubIngester{})

	resp := apitest.HandlePOST(t, handler, "/api/import/telegram", strings.NewReader(validTelegramJSON),
		apitest.WithContentType("application/json"))

	resp.IsOK()
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("session_id").IsString()
		j.Node("total").IsNumber().EqualTo(2)
		j.Node("current_index").IsNumber().EqualTo(0)
		j.Node("current_item").IsObject()
		j.Node("current_item", "text").IsString().EqualTo("Second")
	})
}

func TestImportTelegramCreate_WhenInvalidJSON_Expect400(t *testing.T) {
	t.Parallel()
	handler := setupImportTestHandler(t, &ingestion.StubIngester{})

	resp := apitest.HandlePOST(t, handler, "/api/import/telegram", strings.NewReader("not json"),
		apitest.WithContentType("application/json"))

	resp.IsBadRequest()
}

func TestImportTelegramGetSession_WhenValid(t *testing.T) {
	t.Parallel()
	handler := setupImportTestHandler(t, &ingestion.StubIngester{})

	req := httptest.NewRequest(http.MethodPost, "/api/import/telegram", strings.NewReader(validTelegramJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var createData struct {
		SessionID string `json:"session_id"` //nolint:tagliatelle // API response snake_case
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createData))
	require.NotEmpty(t, createData.SessionID)

	getResp := apitest.HandleGET(t, handler, "/api/import/telegram/session/"+createData.SessionID)
	getResp.IsOK()
	getResp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("session_id").IsString().EqualTo(createData.SessionID)
		j.Node("total").IsNumber().EqualTo(2)
		j.Node("current_index").IsNumber().EqualTo(0)
		j.Node("processed_count").IsNumber().EqualTo(0)
		j.Node("rejected_count").IsNumber().EqualTo(0)
		j.Node("current_item").IsObject()
	})
}

func TestImportTelegramGetSession_WhenNotFound_Expect404(t *testing.T) {
	t.Parallel()
	handler := setupImportTestHandler(t, &ingestion.StubIngester{})

	resp := apitest.HandleGET(t, handler, "/api/import/telegram/session/unknown-id")
	resp.IsNotFound()
}

func TestImportTelegramAccept_WhenValid(t *testing.T) {
	t.Parallel()
	ingester := &mockIngester{
		node: &kb.Node{Path: "topic/node1", Metadata: map[string]any{}},
	}
	handler := setupImportTestHandler(t, ingester)

	req := httptest.NewRequest(http.MethodPost, "/api/import/telegram", strings.NewReader(validTelegramJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var createData struct {
		SessionID string `json:"session_id"` //nolint:tagliatelle // API response snake_case
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createData))

	acceptResp := apitest.HandlePOST(t, handler, "/api/import/telegram/session/"+createData.SessionID+"/accept",
		strings.NewReader(`{"type_hint":"article"}`),
		apitest.WithContentType("application/json"))
	acceptResp.IsOK()
	acceptResp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("node").IsObject()
		j.Node("node", "path").IsString().EqualTo("topic/node1")
		j.Node("next_item").IsObject()
		j.Node("next_item", "text").IsString().EqualTo("First")
	})
}

func TestImportTelegramReject_WhenValid(t *testing.T) {
	t.Parallel()
	handler := setupImportTestHandler(t, &ingestion.StubIngester{})

	req := httptest.NewRequest(http.MethodPost, "/api/import/telegram", strings.NewReader(validTelegramJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var createData struct {
		SessionID string `json:"session_id"` //nolint:tagliatelle // API response snake_case
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createData))

	rejectResp := apitest.HandlePOST(t, handler, "/api/import/telegram/session/"+createData.SessionID+"/reject",
		strings.NewReader(`{}`),
		apitest.WithContentType("application/json"))
	rejectResp.IsOK()
	rejectResp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("next_item").IsObject()
		j.Node("next_item", "text").IsString().EqualTo("First")
	})
}

func TestImportTelegramAccept_WhenSessionComplete_Expect409(t *testing.T) {
	t.Parallel()
	ingester := &mockIngester{
		node: &kb.Node{Path: "topic/node1", Metadata: map[string]any{}},
	}
	handler := setupImportTestHandler(t, ingester)

	req := httptest.NewRequest(http.MethodPost, "/api/import/telegram", strings.NewReader(validTelegramJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var createData struct {
		SessionID string `json:"session_id"` //nolint:tagliatelle // API response snake_case
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createData))

	// Accept both items
	for i := 0; i < 2; i++ {
		acceptResp := apitest.HandlePOST(t, handler, "/api/import/telegram/session/"+createData.SessionID+"/accept",
			strings.NewReader(`{}`),
			apitest.WithContentType("application/json"))
		acceptResp.IsOK()
	}

	// Third accept — сессия завершена, нет текущей записи
	resp := apitest.HandlePOST(t, handler, "/api/import/telegram/session/"+createData.SessionID+"/accept",
		strings.NewReader(`{}`),
		apitest.WithContentType("application/json"))
	resp.HasCode(http.StatusConflict)
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("error").IsString().EqualTo("no current item")
	})
}

func TestImportTelegram_WhenNotConfigured_Expect503(t *testing.T) {
	t.Parallel()
	dataPath := t.TempDir()
	h := api.NewHandler(dataPath, &ingestion.StubIngester{})
	mux, err := api.NewMux(h)
	require.NoError(t, err)

	resp := apitest.HandlePOST(t, mux, "/api/import/telegram", strings.NewReader(validTelegramJSON),
		apitest.WithContentType("application/json"))

	resp.HasCode(http.StatusServiceUnavailable)
	resp.HasJSON(func(j *assertjson.AssertJSON) {
		j.Node("error").IsString().EqualTo("import not configured")
	})
}
