---
name: api-conventions
description: HTTP API conventions for knowledge-db — hybrid POST-action mutations + REST GET reads, snake_case JSON, error responses, status mapping. Use when designing or implementing api handlers or web API client types.
---

# HTTP API conventions (knowledge-db)

Source of truth for routes: `internal/api/router.go`.

knowledge-db uses a **hybrid** style — a sanctioned carve-out from the house POST-action
convention (`api-conventions` in the hub), chosen deliberately for a knowledge base:

- **Mutations** follow the house **POST-action / RPC** style:
  `POST /api/<resource>/<action>` with the identifier/path in the **JSON body**
  (`path`, `id`, `target_path`). **No** `PUT`/`DELETE`/`PATCH` under `/api/`.
- **Reads** stay **REST, addressed by path** (`GET /api/nodes/{path...}`,
  `GET /api/nodes/by-id/{id}`, `GET /api/assets/{path...}`, status/log pollers): KB nodes have
  human-meaningful, shareable/bookmarkable URLs, and asset downloads need a plain `<img src>`
  URL — the one place path-addressing genuinely earns its keep.

This split is enforced by `internal/api/router_guard_test.go` (verbs-only: fails on
`PUT`/`DELETE`/`PATCH`; path params in `GET` are allowed).

## Methods and routing

- **Go 1.22+** patterns on `http.ServeMux`: `"GET /api/nodes/{path...}"`, `"POST /api/nodes/update"`
- **Path params** ONLY for `GET` reads (`{path...}`, `{id}`)
- **Mutations**: one route + one handler per action — no suffix-dispatch multiplexers.
  Extract `path`/`id` from the decoded JSON body; empty → 400.

Common groups:

| Area | Examples |
|------|----------|
| Health | `GET /healthz`, `GET /readyz` |
| Auth | `POST /api/auth/login`, `GET /api/auth/session`, OAuth start/callback |
| Nodes (read) | `GET /api/nodes/{path...}`, `GET /api/nodes/by-id/{id}`, `GET /api/nodes` |
| Nodes (mutate) | `POST /api/nodes/update\|delete\|move\|refresh-description\|normalize\|agent-edit\|dump-images` (`path` in body) |
| Discovery | `GET /api/tree`, `GET/POST /api/search` |
| Ingest | `POST /api/ingest` |
| Git | `GET /api/git/status`, `POST /api/git/commit`, `POST /api/git/sync` |
| Chat | `GET/POST /api/chats`, `GET /api/chats/{id}`, `POST /api/chats/update\|delete`, `POST /api/chat` |
| Debug issues | `POST /api/debug/issues`, `POST /api/debug/issues/update` |
| Telegram import | `POST /api/import/telegram`, `GET /api/import/telegram/session/{id}`, `POST /api/import/telegram/session/accept\|reject` |
| Translate | `GET /api/articles/translate/{path...}`, `POST /api/articles/translate` (`path` in body) |
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

- `Content-Type: application/json` for POST with body
- Every mutation body carries its own `path`/`id` (snake_case); handler extracts it, `TrimSpace`, 400 if empty
- Empty body allowed only where handler explicitly accepts it (e.g. git commit with optional message)
- Partial updates: `POST /api/nodes/update` — body `{ path, ...поля }`, only supported frontmatter
  fields (`manual_processed`, `title`, `keywords`, `labels`); decoded as `map[string]json.RawMessage`
  to distinguish absent from zero (extract & delete `path` first, then require ≥1 remaining field)

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

- [ ] Mutation is `POST /api/<resource>/<action>` with `path`/`id` in the JSON body (never PUT/DELETE/PATCH)
- [ ] Read is `GET` (path params OK) — mutation is never a GET
- [ ] JSON fields snake_case
- [ ] `router_guard_test.go` still passes (no REST verb crept in)
- [ ] Domain errors mapped via `errors.Is` to correct status
- [ ] Errors logged with `clog.Errorf` before 5xx
- [ ] Test in `internal/api/*_test.go`
