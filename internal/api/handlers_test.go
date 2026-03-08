package api_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muonsoft/api-testing/apitest"
	"github.com/muonsoft/api-testing/assertjson"
	"github.com/muonsoft/errors"
	"github.com/stretchr/testify/require"
	"github.com/strider2038/knowledge-db/internal/api"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

var errLLMUnavailable = errors.New("LLM unavailable")

type mockIngester struct {
	node *kb.Node
	err  error
}

func (m *mockIngester) IngestText(_ context.Context, _ ingestion.IngestRequest) (*kb.Node, error) {
	return m.node, m.err
}

func (m *mockIngester) IngestURL(_ context.Context, _ string) (*kb.Node, error) {
	return m.node, m.err
}

func setupTestHandlerWithIngester(t *testing.T, ingester ingestion.Ingester) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	h := api.NewHandler(tmp, ingester)
	mux, err := api.NewMux(h)
	require.NoError(t, err)

	return mux
}

func setupTestHandler(t *testing.T) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	// Создаём валидный узел для тестов (topic/node1.md с frontmatter)
	themeDir := filepath.Join(tmp, "topic")
	_ = os.MkdirAll(themeDir, 0o755)
	node1Content := `---
keywords: [a]
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Annotation"
---

Content`
	_ = os.WriteFile(filepath.Join(themeDir, "node1.md"), []byte(node1Content), 0o644)
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

func setupTestHandlerForRecursive(t *testing.T) http.Handler {
	t.Helper()
	tmp := t.TempDir()
	for path, content := range map[string]string{
		"programming/node1.md": `---
keywords: [go, golang]
type: article
title: Go Basics
created: "2024-01-01T00:00:00Z"
updated: "2024-01-01T00:00:00Z"
annotation: "Article about Go"
source_url: "https://example.com/go"
---

Content`,
		"programming/scaling/node2.md": `---
keywords: [load, balancing]
type: link
title: Load Balancing
created: "2024-01-02T00:00:00Z"
updated: "2024-01-02T00:00:00Z"
annotation: "Link about load balancing"
source_url: "https://example.com/lb"
---

Content`,
		"programming/scaling/node3.md": `---
keywords: [kubernetes]
type: note
title: K8s Notes
created: "2024-01-03T00:00:00Z"
updated: "2024-01-03T00:00:00Z"
annotation: "Personal notes"
---

Content`,
		"ai/node4.md": `---
keywords: [ml, ai]
type: article
title: AI Overview
created: "2024-01-04T00:00:00Z"
updated: "2024-01-04T00:00:00Z"
annotation: "Machine learning intro"
---

Content`,
	} {
		fullPath := filepath.Join(tmp, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
	}
	h := api.NewHandler(tmp, &ingestion.StubIngester{})
	mux, err := api.NewMux(h)
	require.NoError(t, err)

	return mux
}

func TestListNodes_WhenRecursiveTrue_ExpectNodesFromSubtree(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerForRecursive(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes?path=programming&recursive=true")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray().WithLength(3)
		json.Node("total").IsInteger().EqualTo(3)
	})
}

func TestListNodes_WhenQueryQ_ExpectFilteredBySearch(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerForRecursive(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes?path=&recursive=true&q=go")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray().WithLength(1)
		json.Node("total").IsInteger().EqualTo(1)
		json.Node("nodes", 0, "title").IsString().EqualTo("Go Basics")
	})
}

func TestListNodes_WhenTypeFilter_ExpectFilteredByType(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerForRecursive(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes?path=&recursive=true&type=article,link")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray().WithLength(3)
		json.Node("total").IsInteger().EqualTo(3)
	})
}

func TestListNodes_WhenLimitOffset_ExpectPaginated(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerForRecursive(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes?path=&recursive=true&limit=2&offset=1")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray().WithLength(2)
		json.Node("total").IsInteger().EqualTo(4)
	})
}

func TestListNodes_WhenSortOrder_ExpectSorted(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerForRecursive(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes?path=&recursive=true&sort=title&order=asc")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray().WithLength(4)
		json.Node("nodes", 0, "title").IsString().EqualTo("AI Overview")
	})
}

func TestListNodes_WhenRecursiveFalse_ExpectBackwardCompatible(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerForRecursive(t)

	resp := apitest.HandleGET(t, handler, "/api/nodes?path=programming")

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("nodes").IsArray().WithLength(1)
		json.Node("total").DoesNotExist()
		json.Node("nodes", 0, "name").IsString().EqualTo("node1")
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

func TestIngest_WhenSuccess_ExpectOKWithNode(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithIngester(t, &mockIngester{
		node: &kb.Node{
			Path:       "go/concurrency/goroutine-leak",
			Annotation: "Article about goroutine leaks",
			Content:    "# Goroutine Leaks",
			Metadata: map[string]any{
				"type":     "article",
				"keywords": []any{"goroutines"},
			},
		},
	})

	resp := apitest.HandlePOST(t, handler, "/api/ingest", strings.NewReader(`{"text":"https://habr.com/article"}`),
		apitest.WithContentType("application/json"))

	resp.IsOK()
	resp.HasJSON(func(json *assertjson.AssertJSON) {
		json.Node("path").IsString().EqualTo("go/concurrency/goroutine-leak")
		json.Node("annotation").IsString().EqualTo("Article about goroutine leaks")
	})
}

func TestIngest_WhenSourceMetadata_ExpectPassedToIngester(t *testing.T) {
	t.Parallel()
	var lastReq ingestion.IngestRequest
	handler := setupTestHandlerWithIngester(t, &captureIngestRequestIngester{
		node: &kb.Node{Path: "go/test", Metadata: map[string]any{}},
		capture: func(req ingestion.IngestRequest) {
			lastReq = req
		},
	})

	resp := apitest.HandlePOST(t, handler, "/api/ingest",
		strings.NewReader(`{"text":"note","source_url":"https://t.me/channel/1","source_author":"Author"}`),
		apitest.WithContentType("application/json"))

	resp.IsOK()
	require.Equal(t, "note", lastReq.Text)
	require.Equal(t, "https://t.me/channel/1", lastReq.SourceURL)
	require.Equal(t, "Author", lastReq.SourceAuthor)
}

type captureIngestRequestIngester struct {
	node    *kb.Node
	capture func(ingestion.IngestRequest)
}

func (c *captureIngestRequestIngester) IngestText(_ context.Context, req ingestion.IngestRequest) (*kb.Node, error) {
	c.capture(req)

	return c.node, nil
}

func (c *captureIngestRequestIngester) IngestURL(_ context.Context, _ string) (*kb.Node, error) {
	return c.node, nil
}

func TestIngest_WhenIngesterError_Expect500(t *testing.T) {
	t.Parallel()
	handler := setupTestHandlerWithIngester(t, &mockIngester{
		err: errLLMUnavailable,
	})

	resp := apitest.HandlePOST(t, handler, "/api/ingest", strings.NewReader(`{"text":"some text"}`),
		apitest.WithContentType("application/json"))

	resp.HasCode(http.StatusInternalServerError)
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

func TestSPA_WhenMissingAsset_Expect404(t *testing.T) {
	t.Parallel()
	handler := setupTestHandler(t)

	resp := apitest.HandleGET(t, handler, "/assets/nonexistent-chunk-123.js")

	resp.HasCode(http.StatusNotFound)
}
