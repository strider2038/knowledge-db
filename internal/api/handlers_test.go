package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/strider2038/knowledge-db/internal/ingestion"
)

func setupTestHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	tmp := t.TempDir()
	// Создаём валидный узел для тестов
	nodePath := filepath.Join(tmp, "topic", "node1")
	_ = os.MkdirAll(nodePath, 0o755)
	_ = os.WriteFile(filepath.Join(nodePath, "annotation.md"), []byte("Annotation"), 0o644)
	_ = os.WriteFile(filepath.Join(nodePath, "content.md"), []byte("Content"), 0o644)
	_ = os.WriteFile(filepath.Join(nodePath, "metadata.json"), []byte(`{"keywords":["a"],"created":"2024-01-01T00:00:00Z","updated":"2024-01-01T00:00:00Z"}`), 0o644)
	h := NewHandler(tmp, &ingestion.StubIngester{})
	mux := NewMux(h)
	return mux, tmp
}

func TestGetNode_WhenNotFound_Expect404(t *testing.T) {
	t.Parallel()
	handler, _ := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes/missing/path")

	resp.IsNotFound()
}

func TestGetNode_WhenValidPath_ExpectOK(t *testing.T) {
	t.Parallel()
	handler, _ := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes/topic/node1")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("path").IsString().EqualTo("topic/node1")
		json.Node("annotation").IsString().EqualTo("Annotation")
		json.Node("content").IsString().EqualTo("Content")
	})
}

func TestGetTree_WhenValidBase_ExpectOK(t *testing.T) {
	t.Parallel()
	handler, _ := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/tree")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("children").IsArray()
	})
}

func TestListNodes_WhenValidPath_ExpectOK(t *testing.T) {
	t.Parallel()
	handler, _ := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes?path=topic")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray()
	})
}

func TestSearch_WhenQuery_ExpectOK(t *testing.T) {
	t.Parallel()
	handler, _ := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/search?q=test")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray()
	})
}

func TestIngest_WhenEmptyText_Expect400(t *testing.T) {
	t.Parallel()
	handler, _ := setupTestHandler(t)

	resp := apitest.HandlePOST(t, handler, "/api/ingest", strings.NewReader(`{"text":""}`),
		apitest.WithContentType("application/json"))

	resp.IsBadRequest()
}

func TestIngest_WhenStub_Expect501(t *testing.T) {
	t.Parallel()
	handler, _ := setupTestHandler(t)

	resp := apitest.HandlePOST(t, handler, "/api/ingest", strings.NewReader(`{"text":"hello"}`),
		apitest.WithContentType("application/json"))

	resp.HasCode(http.StatusNotImplemented)
}
