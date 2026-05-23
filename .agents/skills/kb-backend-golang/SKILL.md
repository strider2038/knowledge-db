---
name: kb-backend-golang
description: Go backend style and practices for knowledge-db (cmd/kb-server, cmd/kb-cli, internal/). Use when editing handlers, kb store, ingestion, index, or bootstrap code.
---

# Go backend — knowledge-db

Applies to `cmd/kb-server`, `cmd/kb-cli`, and `internal/`.

Goals: idiomatic Go, clear package boundaries, testable code via interfaces, offline-first behavior.

## Layers (this project)

| Package | Role |
|---------|------|
| `internal/api` | HTTP handlers, routing, JSON, status mapping |
| `internal/kb` | Filesystem knowledge base, frontmatter, tree, validation |
| `internal/ingestion` | Capture pipeline, fetchers, LLM orchestration |
| `internal/index` | Search index, embeddings, sync worker |
| `internal/bootstrap` | Config, wiring |
| `internal/mcp` | MCP endpoint |
| `cmd/kb-cli` | validate, init |

There is **no PostgreSQL** and no entity/repository layout — persistence is markdown on disk (`KB_DATA_PATH`).

Prefer **interfaces** at package boundaries (`Ingester`, `index.Store`, `igit.GitCommitter`) for tests.

## Formatting and imports

- Run `gofmt` / `goimports` on changed files.
- Import groups: stdlib → external → `github.com/strider2038/knowledge-db/...` (alphabetical within each).

## Linting

- `golangci-lint` with root `.golangci.yml`.
- Prefer inline `// #nosec` with a reason over broad config excludes.

## Code style

- Functions: verbs (`GetNode`, `Ingest`, `CommitAll`).
- Early returns; nesting depth ≤ 3.
- I/O functions: `context.Context` first; do not store context in struct fields.
- Avoid more than two bare return values — use a small struct when needed.

## Errors and logging

- Errors: `github.com/muonsoft/errors` — see [golang-errors](../golang-errors/SKILL.md).
- Logging: `github.com/muonsoft/clog` — see [golang-logging](../golang-logging/SKILL.md).
- **No panic** in production paths; return `error`.
- Do not ignore errors (`_ = err` forbidden).

## HTTP handlers

- Map `kb.ErrNodeNotFound` → 404, `kb.ErrConflict` → 409, validation/bad input → 400.
- Use shared helpers `writeJSON`, `writeError` in `internal/api`.
- Wrap unexpected failures with `errors.Errorf` before logging and 500 responses.

## Storage and testing

- Production store: `kb.NewStore(afero.NewOsFs())`.
- Tests: `afero.NewMemMapFs()` — see [golang-tests](../golang-tests/SKILL.md).
- Path to KB: `KB_DATA_PATH` (never assume `./data` in code comments for agents).

## Concurrency

- Background work: `pior/runnable` — see [runnable-background-processes](../runnable-background-processes/SKILL.md).
- Goroutines respect context cancellation; protect shared state with mutex or channels.
- Index sync worker and translation queue: clear lifecycle, log failures with `clog.Errorf`.

## API JSON

- Request/response fields use **snake_case** JSON tags (`target_path`, `source_url`) — see [api-conventions](../api-conventions/SKILL.md).
- REST-style routes with path parameters (`/api/nodes/{path...}`), not miniapp `POST .../find`.

## Related skills

- Structure: [backend-structure](../backend-structure/SKILL.md)
- Tests: [golang-tests](../golang-tests/SKILL.md)
- Validation (when adopted): [golang-validation](../golang-validation/SKILL.md)

## Pre-commit checklist

- [ ] `gofmt` / `goimports` on touched Go files
- [ ] `golangci-lint` passes
- [ ] Tests for changed behavior (`go test ./...` or targeted package)
- [ ] No debug/temporary logging left behind
