package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func setupTestHandlerWithNode(t *testing.T) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	themeDir := filepath.Join(tmp, "topic")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	content := `---
keywords: [test]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Test annotation"
---

Content here`
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "my-node.md"), []byte(content), 0o644))
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	return mux
}

func TestDeleteNode_WhenExists_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithNode(t)

	resp := apitest.HandleDELETE(t, handler, "/api/nodes/topic/my-node")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("path").IsString().EqualTo("topic/my-node")
		json.Node("deleted").IsTrue()
	})
}

func TestDeleteNode_WhenNotFound_Expect404(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithNode(t)

	resp := apitest.HandleDELETE(t, handler, "/api/nodes/nonexistent/path")

	resp.IsNotFound()
}

func TestDeleteNode_WhenEmptyPath_Expect400(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithNode(t)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/nodes/", nil)
	resp := apitest.HandleRequest(t, handler, req)

	resp.IsBadRequest()
}

func TestMoveNode_WhenValidTarget_ExpectOK(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithNode(t)

	resp := apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/move",
		strings.NewReader(`{"target_path":"new-topic/my-node"}`),
		apitest.WithJSONContentType())

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("path").IsString().EqualTo("new-topic/my-node")
	})
}

func TestMoveNode_WhenConflict_Expect409(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "source"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "target"), 0o755))
	content := "---\nkeywords: []\n---\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "source", "node.md"), []byte(content), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "target", "node.md"), []byte(content), 0o644))
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	resp := apitest.HandlePOST(t, mux, "/api/nodes/source/node/move",
		strings.NewReader(`{"target_path":"target/node"}`),
		apitest.WithJSONContentType())

	resp.HasCode(http.StatusConflict)
}

func TestMoveNode_WhenNotFound_Expect404(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithNode(t)

	resp := apitest.HandlePOST(t, handler, "/api/nodes/nonexistent/path/move",
		strings.NewReader(`{"target_path":"target/node"}`),
		apitest.WithJSONContentType())

	resp.IsNotFound()
}

func TestMoveNode_WhenEmptyTargetPath_Expect400(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithNode(t)

	resp := apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/move",
		strings.NewReader(`{"target_path":""}`),
		apitest.WithJSONContentType())

	resp.IsBadRequest()
}

func TestMoveNode_WhenPathTraversal_Expect400(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithNode(t)

	resp := apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/move",
		strings.NewReader(`{"target_path":"../etc/passwd"}`),
		apitest.WithJSONContentType())

	resp.IsBadRequest()
}
