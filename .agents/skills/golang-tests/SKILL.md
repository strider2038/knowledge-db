---
name: golang-tests
description: Go testing for knowledge-db — API tests (muonsoft/api-testing), unit tests with testify and afero. Use when writing tests for internal/ and cmd/.
---

# Go testing (knowledge-db)

Cover API handlers, `internal/kb`, `internal/ingestion`, and related packages.

Detailed API examples: [references/api-tests.md](references/api-tests.md).

## One scenario per test (AAA)

Use explicit sections:

```go
func TestDeleteNode_WhenExists_ExpectOK(t *testing.T) {
    t.Parallel()
    // Arrange
    handler := setupTestHandlerWithNode(t)

    // Act
    resp := apitest.HandleDELETE(t, handler, "/api/nodes/topic/my-node")

    // Assert
    resp.IsOK()
    resp.HasJSON(func(json *assertjson.AssertJSON) {
        json.Node("path").IsString().EqualTo("topic/my-node")
        json.Node("deleted").IsTrue()
    })
}
```

## API tests (muonsoft/api-testing)

Packages: `github.com/muonsoft/api-testing/apitest`, `assertjson`.

```go
resp := apitest.HandleGET(t, mux, "/api/git/status")
resp.IsOK()

resp := apitest.HandlePOST(t, mux, "/api/git/commit",
    strings.NewReader(`{"message":"sync"}`),
    apitest.WithJSONContentType(),
)
resp.HasCode(503)
```

**assertjson paths:** variadic `Node("key", 0, "nested")` — not legacy `/key/0/nested`.

Custom requests: `httptest.NewRequest` + `apitest.HandleRequest(t, handler, req)`.

Setup pattern:

```go
h := api.NewHandler(dataPath, &ingestion.StubIngester{})
h.SetGitCommitter(mock, nil, gitDisabled)
mux, err := api.NewMux(h, nil)
require.NoError(t, err)
```

Use `package api_test` (black-box) for handler tests.

## Naming

```text
Test<Entity>_<Action>_When<Condition>_Expect<Result>
```

Examples: `TestGetGitStatus_WhenGitDisabled_Expect503`, `TestMoveNode_WhenConflict_Expect409`.

## testify

| Use | Package |
|-----|---------|
| Must stop test | `require.NoError`, `require.Error` |
| Continue on failure | `assert.Equal`, `assert.True`, `assert.ErrorIs` |

Prefer `assert.ErrorIs(t, err, target)` over `assert.True(t, errors.Is(...))`.

## Helpers

- Accept `testing.TB`, call `tb.Helper()` at start.
- On setup failure: `tb.Fatalf` — **no panic** in test helpers.

## Filesystem tests

### API / integration-style: `t.TempDir`

Many `internal/api` tests seed markdown under `t.TempDir()` and pass the path to `api.NewHandler`. This matches real on-disk layout.

### Unit tests for `kb.Store`: afero

```go
fs := afero.NewMemMapFs()
store := kb.NewStore(fs)
```

Use absolute paths with MemMapFs (`/` as base). See `internal/kb` tests for `seedMemFS` patterns.

## Mocks

- Small interfaces (`igit.GitCommitter`, `ingestion.Ingester`) — manual mocks in `*_test.go`.
- Return errors with `errors.Errorf` from muonsoft/errors for anonymous failures.

## What we do not use here

- Postgres / `//go:build integration` repository suites
- testify/suite + DI test containers
- OAuth cookie / company-scoped access tests

## Checklist

- [ ] API endpoints touched have tests
- [ ] AAA structure with `t.Parallel()` where safe
- [ ] `TestX_WhenY_ExpectZ` naming
- [ ] testify `require` / `assert`, not bare `t.Fatal` except in helpers
- [ ] JSON assertions via `assertjson` + `HasJSON`
- [ ] kb store unit tests use afero when testing `Store` directly
