# API tests — knowledge-db reference

Reference implementations:

- `internal/api/node_management_test.go`
- `internal/api/git_handlers_test.go`

## Package and imports

```go
package api_test

import (
    "net/http"
    "strings"
    "testing"

    "github.com/muonsoft/api-testing/apitest"
    "github.com/muonsoft/api-testing/assertjson"
    "github.com/stretchr/testify/require"
    "github.com/strider2038/knowledge-db/internal/api"
    "github.com/strider2038/knowledge-db/internal/ingestion"
)
```

## Handler setup with filesystem

```go
func setupTestHandlerWithNode(t *testing.T) http.Handler {
    t.Helper()
    tmp := t.TempDir()
    themeDir := filepath.Join(tmp, "topic")
    require.NoError(t, os.MkdirAll(themeDir, 0o755))
    require.NoError(t, os.WriteFile(
        filepath.Join(themeDir, "my-node.md"),
        []byte(`---
keywords: [test]
type: note
title: My Node
---
Content`),
        0o644,
    ))
    h := api.NewHandler(tmp, &ingestion.StubIngester{})
    mux, err := api.NewMux(h, nil)
    require.NoError(t, err)
    return mux
}
```

## DELETE node

```go
resp := apitest.HandleDELETE(t, handler, "/api/nodes/topic/my-node")
resp.IsOK()
resp.HasJSON(func(json *assertjson.AssertJSON) {
    json.Node("path").IsString().EqualTo("topic/my-node")
    json.Node("deleted").IsTrue()
})

resp := apitest.HandleDELETE(t, handler, "/api/nodes/nonexistent/path")
resp.IsNotFound()
```

## POST with JSON body (snake_case)

```go
resp := apitest.HandlePOST(t, handler, "/api/nodes/topic/my-node/move",
    strings.NewReader(`{"target_path":"new-topic/my-node"}`),
    apitest.WithJSONContentType(),
)
resp.IsOK()
```

Status checks:

```go
resp.IsBadRequest()
resp.IsNotFound()
resp.HasCode(http.StatusConflict)
resp.HasCode(503)
```

## Mock git committer

```go
type mockGitCommitter struct {
    status    *igit.GitStatus
    statusErr error
    commitErr error
}

func setupGitHandler(t *testing.T, committer igit.GitCommitter, gitDisabled bool) http.Handler {
    t.Helper()
    h := api.NewHandler(t.TempDir(), &ingestion.StubIngester{})
    h.SetGitCommitter(committer, nil, gitDisabled)
    mux, err := api.NewMux(h, nil)
    require.NoError(t, err)
    return mux
}

func TestGetGitStatus_WhenChanges_ExpectOK(t *testing.T) {
    t.Parallel()
    mux := setupGitHandler(t, &mockGitCommitter{
        status: &igit.GitStatus{HasChanges: true, ChangedFiles: 3},
    }, false)

    resp := apitest.HandleGET(t, mux, "/api/git/status")
    resp.IsOK()
    resp.HasJSON(func(json *assertjson.AssertJSON) {
        json.Node("has_changes").IsTrue()
        json.Node("changed_files").IsInteger().EqualTo(3)
    })
}
```

## Custom request (edge URL)

When path params are awkward to hit via helper URLs:

```go
req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/api/nodes/", nil)
resp := apitest.HandleRequest(t, handler, req)
resp.IsBadRequest()
```

## Index worker integration

For move + reindex, start `index.SyncWorker` in a goroutine, use `require.Eventually` to assert sqlite index state. See `TestMoveNode_WhenIndexWorkerConfigured_ExpectReindexOldAndNewPaths` in `node_management_test.go`.

## apitest helpers summary

| Helper | Use |
|--------|-----|
| `HandleGET(t, handler, path)` | GET |
| `HandlePOST(t, handler, path, body, opts...)` | POST |
| `HandleDELETE(t, handler, path)` | DELETE |
| `HandleRequest(t, handler, req)` | Custom method/URL |
| `WithJSONContentType()` | `Content-Type: application/json` |
| `IsOK()`, `IsNotFound()`, `IsBadRequest()` | Status shortcuts |
| `HasJSON(func(*assertjson.AssertJSON))` | Body assertions |
