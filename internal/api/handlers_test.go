package api_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/ingestion"
)

func setupTestHandler(t *testing.T) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	// Создаём валидный узел для тестов (node1.md с frontmatter)
	nodePath := filepath.Join(tmp, "topic", "node1")
	_ = os.MkdirAll(nodePath, 0o755)
	node1Content := `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Annotation"
---

Content`
	_ = os.WriteFile(filepath.Join(nodePath, "node1.md"), []byte(node1Content), 0o644)
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	mux, err := api.NewMux(h)
	require.NoError(t, err)

	return mux
}

func TestGetNode_WhenNotFound_Expect404(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes/missing/path")

	resp.IsNotFound()
}

func TestGetNode_WhenValidPath_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

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
	handler := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/tree")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("children").IsArray()
	})
}

func TestListNodes_WhenValidPath_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes?path=topic")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray()
	})
}

func TestSearch_WhenQuery_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/api/search?q=test")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray()
	})
}

func TestIngest_WhenEmptyText_Expect400(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

	resp := apitest.HandlePOST(t, handler, "/api/ingest", strings.NewReader(`{"text":""}`),
		apitest.WithContentType("application/json"))

	resp.IsBadRequest()
}

func TestIngest_WhenStub_Expect501(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

	resp := apitest.HandlePOST(t, handler, "/api/ingest", strings.NewReader(`{"text":"hello"}`),
		apitest.WithContentType("application/json"))

	resp.HasCode(http.StatusNotImplemented)
}

func TestSPA_WhenRoot_ExpectIndexHTML(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/")

	resp.IsOK()
	resp.HasContentType("text/html; charset=utf-8")
}

func TestSPA_WhenAddRoute_ExpectIndexHTML(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/add")

	resp.IsOK()
	resp.HasContentType("text/html; charset=utf-8")
}
