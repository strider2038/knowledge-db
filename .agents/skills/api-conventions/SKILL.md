---
name: api-conventions
description: HTTP API conventions for knowledge-db — REST routes, snake_case JSON, error responses, status mapping. Use when designing or implementing api handlers or web API client types.
---

# HTTP API conventions (knowledge-db)

Source of truth for routes: `internal/api/router.go`.

This is **not** a miniapp Rich API (`POST /api/entities/find`). knowledge-db uses REST-style methods and path parameters.

## Methods and routing

- **Go 1.22+** patterns on `http.ServeMux`: `"GET /api/nodes/{path...}"`
- **Path params** for node paths and resource ids (`{path...}`, `{id}`)
- **Sub-actions** as path suffixes where needed, e.g. `POST /api/nodes/{path...}/move` is registered via `MoveNode` (path value strips `/move`)

Common groups:

| Area | Examples |
|------|----------|
| Health | `GET /healthz`, `GET /readyz` |
| Auth | `POST /api/auth/login`, `GET /api/auth/session`, OAuth start/callback |
| Nodes | `GET/PATCH/DELETE /api/nodes/{path...}`, `POST` for move |
| Discovery | `GET /api/tree`, `GET/POST /api/search`, `GET /api/nodes` |
| Ingest | `POST /api/ingest` |
| Git | `GET /api/git/status`, `POST /api/git/commit`, `POST /api/git/sync` |
| Chat | `GET/POST /api/chats`, `POST /api/chat` |
| Jobs | `POST /api/jobs`, `GET /api/jobs/{id}`, logs subpaths |
| Index | `POST /api/index/rebuild`, `GET /api/index/status` |

## JSON naming

- Request and response fields use **snake_case** in JSON: `target_path`, `source_url`, `has_changes`, `manual_processed`
- Match tags on Go structs: `` `json:"target_path"` ``

## Response helpers

```go
writeJSON(w, payload)           // 200 + application/json
writeError(w, code, msg string) // {"error":"<msg>"} + status code
```

Keep error messages short and safe for UI display; log details with `clog.Errorf` server-side.

## Status mapping (typical)

| Condition | Status |
|-----------|--------|
| `errors.Is(err, kb.ErrNodeNotFound)` | 404 |
| `errors.Is(err, kb.ErrConflict)` | 409 |
| `errors.Is(err, kb.ErrInvalidPath)` or bad JSON/body | 400 |
| Git disabled / feature unavailable | 503 |
| Upstream fetch/LLM failure (ingest/refresh) | 502 |
| Unexpected internal error | 500 |

Auth failures use auth middleware / handler logic (401) when auth is enabled.

## Request bodies

- `Content-Type: application/json` for POST/PATCH with body
- Empty body allowed only where handler explicitly accepts it (e.g. git commit with optional message)
- Partial updates: `PATCH /api/nodes/{path...}` — only supported frontmatter fields (`manual_processed`, `title`, `keywords`, `labels`)

## Path parameters

- Node paths are logical KB paths (`topic/subtopic/node-name`), not filesystem absolute paths
- Reject traversal (`..`) — handlers return 400
- Trailing slash edge cases: test with `httptest` when behavior is ambiguous

## Frontend contract

- Web client: `web/src/services/api.ts`
- Base URL from `VITE_API_URL`
- Mirror snake_case in TypeScript types

## Adding a new endpoint

1. Add handler method on `api.Handler` or `AuthHandler`
2. Register route in `NewMux`
3. Document JSON shape (snake_case)
4. Add `api_test` with `apitest` — see [golang-tests](../golang-tests/SKILL.md)
5. Update OpenSpec if the behavior is specified there

## Checklist

- [ ] Route uses appropriate HTTP method (GET for reads, POST for actions, PATCH for partial update)
- [ ] JSON fields snake_case
- [ ] Domain errors mapped via `errors.Is` to correct status
- [ ] Errors logged with `clog.Errorf` before 5xx
- [ ] Test in `internal/api/*_test.go`
