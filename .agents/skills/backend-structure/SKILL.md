---
name: backend-structure
description: Backend layout for knowledge-db (cmd/, internal/). Use when adding HTTP handlers, internal packages, kb store, ingestion, index, or auth.
---

# Backend structure — knowledge-db

## Repository layout

```text
/
├── cmd/
│   ├── kb-server/     # HTTP API, embedded UI, Telegram bot, MCP
│   └── kb-cli/        # validate, init, maintenance commands
├── internal/
│   ├── api/           # HTTP handlers, routing, SPA
│   ├── auth/          # Session middleware
│   ├── bootstrap/     # Config, application wiring
│   ├── chat/          # Chat sessions (sqlite store)
│   ├── cliapp/        # Cobra commands for kb-cli
│   ├── debugdata/     # Debug issue storage
│   ├── googleoauth/   # Google OAuth client
│   ├── yandexoauth/   # Yandex OAuth client
│   ├── oauthcommon/   # Shared OAuth helpers (state, allowlist)
│   ├── index/         # Search index, embeddings, sync worker
│   ├── ingestion/     # Ingester, pipeline, fetchers, git committer, LLM
│   ├── import/        # Telegram import sessions
│   ├── kb/            # Filesystem KB: tree, nodes, frontmatter, validation
│   ├── mcp/           # MCP HTTP handler
│   ├── pkg/           # Small shared utilities (e.g. urlutil)
│   ├── telegram/      # Telegram bot runnable
│   └── ui/            # Embedded frontend static files
├── web/               # React sources (built into internal/ui/static)
└── .agents/skills/
```

## internal/kb

- Markdown + YAML frontmatter on disk under `KB_DATA_PATH`
- Tree of topics (2–3 directory levels) and node folders (`{dirname}/{dirname}.md`)
- Sentinels: `ErrNodeNotFound`, `ErrConflict`, `ErrInvalidPath`, etc.
- `kb.Store` accepts `afero.Fs` for tests

## internal/api

- Routing: `net/http.ServeMux` (Go 1.22+ path patterns) — see [api-conventions](../api-conventions/SKILL.md)
- `api.Handler` — nodes, search, ingest, git, import, chat, jobs, index
- `api.AuthHandler` — login, session, Google/Yandex OAuth (optional)
- Helpers: `writeJSON`, `writeError`
- MCP is mounted separately from bootstrap (not in `NewMux` list above — check `bootstrap` wiring)

## internal/ingestion

- `Ingester`: `IngestText`, `IngestURL`, pipeline orchestration
- `internal/ingestion/llm` — OpenAI **Responses API** (not Chat Completions)
- `internal/ingestion/git` — commit/push after saves
- `internal/ingestion/fetcher` — URL metadata fetchers

## internal/index

- SQLite-backed search + optional embeddings
- `SyncWorker` — async reindex on node changes
- Rebuild/status HTTP endpoints on `api.Handler`

## internal/bootstrap

- Loads config from environment
- Wires handler, auth, index, MCP, runnables

## cmd/kb-server

- Reads env (`KB_DATA_PATH`, tokens, auth mode)
- Registers HTTP server, Telegram bot, background workers via `pior/runnable`

## cmd/kb-cli / internal/cliapp

- `validate`, `init`, index rebuild, migrations helpers
- `init` can copy agent skills with `{{DATA_PATH}}` substitution

## Storage model

- **No application database** for knowledge content — files are source of truth
- SQLite used for **index**, **chat**, and optional debug/issue stores
- Git operations optional (`KB_GIT_DISABLED`)

## When adding a feature

1. Domain/file rules → `internal/kb` or `internal/ingestion`
2. HTTP surface → `internal/api` + route in `router.go`
3. Long-running work → runnable or job manager pattern
4. Update OpenSpec when behavior is specified there
