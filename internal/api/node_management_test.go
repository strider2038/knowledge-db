package api_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/index"
	indexSqlite "github.com/strider2038/knowledge-db/internal/index/sqlite"
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

func TestMoveNode_WhenIndexWorkerConfigured_ExpectReindexOldAndNewPaths(t *testing.T) {
	t.Parallel()
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

	store, err := indexSqlite.NewStore(filepath.Join(tmp, "index.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	worker := index.NewSyncWorker(store, &testEmbeddingProvider{vector: []float32{1, 0, 0}}, tmp, "model", 0)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	require.Eventually(t, func() bool {
		_, err := store.GetNodeByPath(context.Background(), "topic/my-node")

		return err == nil
	}, time.Second, 10*time.Millisecond)

	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	h.SetIndexComponents(store, worker, &testEmbeddingProvider{vector: []float32{1, 0, 0}}, config.Embedding{
		Enabled: true,
		APIKey:  "key",
		APIURL:  "http://localhost",
		Model:   "text-embedding-3-small",
	})
	mux, err := api.NewMux(h, nil)
	require.NoError(t, err)

	resp := apitest.HandlePOST(t, mux, "/api/nodes/topic/my-node/move",
		strings.NewReader(`{"target_path":"new-topic/my-node"}`),
		apitest.WithJSONContentType())

	resp.IsOK()
	require.Eventually(t, func() bool {
		_, oldErr := store.GetNodeByPath(context.Background(), "topic/my-node")
		newNode, newErr := store.GetNodeByPath(context.Background(), "new-topic/my-node")

		return errors.Is(oldErr, sql.ErrNoRows) && newErr == nil && newNode.Path == "new-topic/my-node"
	}, time.Second, 10*time.Millisecond)
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
