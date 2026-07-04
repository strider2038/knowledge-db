# Agents Guide — knowledge-db

This guide is for AI agents working on **knowledge-db**, a local-first personal knowledge base manager.

## Project Overview

`knowledge-db` is an offline-first, git-first system for managing a personal knowledge base. The knowledge base itself is a directory of markdown/frontmatter files under the user's control, usually in a separate git repository and referenced through `KB_DATA_PATH`.

The application is intended to run locally. Remote access, hosted storage, and network-dependent features should stay optional unless a change explicitly requires them.

## Product Concept

`knowledge-db` optimizes for two complementary workflows:

- **Writing and capture are online-friendly**: users should be able to add notes, links, forwarded Telegram posts, imported Telegram archives, API payloads, and MCP-created material from wherever they are.
- **Reading and ownership are offline-first and git-first**: the knowledge base must remain local, versioned, mergeable, and available without internet access. Nothing important should depend on a hosted service being reachable.

The system is not just a bookmark archive. Ingested material should become useful knowledge-base nodes: annotated, keyworded, placed in the topic tree, indexed for search, and available as context for RAG/chat workflows.

Reliability matters. Articles disappear, websites change structure, services become unavailable, and networks fail. Local markdown files plus transparent git history are core product properties, not implementation details.

The project supports two main operating modes:

- **Local/offline mode** for reading, validation, search, and optional local LLM usage through tools such as Ollama or LM Studio.
- **Self-hosted mode** on a private VPS for convenient capture from mobile, Telegram bot usage, sync, and broader AI workflows.

## Architecture

```text
knowledge-db/
├── cmd/kb/          # unified CLI (serve, validate, init, maintenance)
├── internal/
│   ├── bootstrap/   # application wiring and config
│   ├── api/         # HTTP handlers and routing
│   ├── kb/          # filesystem store, frontmatter, validation, tree
│   ├── ingestion/   # ingestion pipeline, fetchers, LLM orchestration
│   ├── index/       # local search index, retrieval, embeddings
│   ├── chat/        # RAG chat sessions (SQLite)
│   ├── telegram/    # Telegram bot
│   ├── mcp/         # MCP endpoint
│   └── ui/          # embedded frontend static files
├── web/             # React + Vite frontend
├── .agents/skills/  # repository-specific agent skills
├── .cursor/         # Cursor rules and compatibility assets
├── docs/            # ADRs and project notes
└── openspec/        # OpenSpec specs and changes
```

## Core Principles

1. **Knowledge base as first-class data**: user content lives as markdown/frontmatter files, not in an application database by default.
2. **Capture anywhere, own locally**: web UI, Telegram, API, MCP, and imports may make writing convenient, but long-term ownership stays in local git-backed files.
3. **Offline-first**: validation, browsing, keyword search, and core knowledge-base operations should work without internet access. Embeddings and remote LLM calls are optional enhancements.
4. **Git as source of truth**: prefer mergeable text formats and preserve meaningful diffs.
5. **Process information into knowledge**: ingestion should preserve sources while adding structure, annotations, keywords, placement, and searchability.
6. **Local by default**: `kb serve`, web UI, Telegram bot, and MCP are designed around localhost/private or self-hosted use.
7. **Small, explicit changes**: keep implementation scoped to the relevant package and update specs/docs when behavior or contracts change.

## Language Conventions

- Code identifiers and Go/TypeScript comments should follow the local style already used in the package.
- **Agent skills** (`.agents/skills/**/SKILL.md`) and agent-facing repo docs (`AGENTS.md`) are written in **English**.
- OpenSpec artifacts (`proposal.md`, `design.md`, `tasks.md`, delta specs) are written in Russian.
- User-facing knowledge-base content is usually Russian unless the source or existing entry style says otherwise.

## Agent Skills

Skills live in **`.agents/skills/<name>/SKILL.md`** (project-local; no git subtree). Read the relevant skill before implementing in that area.

**Priority:** project skills → user rules → `.cursor/rules/*.mdc` (rules are short guardrails; skills hold detail).

### Domain and API

| Skill | Use when |
|-------|----------|
| `knowledge-db` | Creating or editing KB markdown nodes under `KB_DATA_PATH` |
| `backend-structure` | Navigating or adding packages under `internal/`, `cmd/` |
| `api-conventions` | HTTP routes, JSON shape, status codes for `internal/api` |

> **API — гибрид (POST-action мутации + REST-чтения):** мутации — `POST /api/<resource>/<action>`
> с `path`/`id` в JSON-теле, **без** `PUT`/`DELETE`/`PATCH`; `GET`-чтения остаются REST по пути
> (shareable deep-links). Enforced by `internal/api/router_guard_test.go`. См. `api-conventions`.

### Go backend

| Skill | Use when |
|-------|----------|
| `kb-backend-golang` | General Go style and layer boundaries in this repo |
| `golang-errors` | `github.com/muonsoft/errors`, wrapping, sentinels, `errors.Is` / `As` |
| `golang-logging` | `github.com/muonsoft/clog`, contextual logging |
| `golang-validation` | `github.com/muonsoft/validation` (when adding validated commands/DTOs) |
| `golang-tests` | API tests (`api-testing`), testify, afero |
| `runnable-background-processes` | Telegram bot, workers via `pior/runnable` |

### Frontend

| Skill | Use when |
|-------|----------|
| `web-frontend` | React/TypeScript UI in `web/` |
| `web-frontend-tests` | Vitest + Testing Library in `web/` |
| `ux-form-practices` | Forms, validation UX, accessibility in `web/` |

### OpenSpec workflow

| Skill | Use when |
|-------|----------|
| `openspec-apply-change` | Implementing tasks from a change |
| `openspec-verify-change` | Verifying implementation vs specs |
| `openspec-archive-change` | Archiving a completed change |
| `openspec-new-change`, `openspec-continue-change`, `openspec-explore`, `openspec-ff-change`, `openspec-onboard`, `openspec-sync-specs`, `openspec-bulk-archive-change` | OpenSpec artifact workflow |

Start with **`openspec-apply-change`** for implementation; use **`openspec-explore`** for design-only discussions.

## Working With The Knowledge Base

- Do not assume the knowledge base is in `./data`. Use `KB_DATA_PATH` or the path provided by the user.
- The knowledge base is usually a **separate git repository**; `kb init --path <kb>` creates `.gitignore` and installs `.agents/skills/knowledge-db/SKILL.md` inside that repo.
- Reading and writing knowledge-base files directly is allowed only when it is part of the requested workflow.
- Keep markdown/frontmatter compatible with the current specs and validator.
- Apply markdown normalization rules from `.cursor/rules/markdown-normalization.mdc` when normalizing entries, OpenSpec documents, or notes in an opened knowledge-base repository.

### Frontmatter contract (keep in sync)

Any change to the **node frontmatter contract** (required/optional YAML fields, validation rules, translation file rules, ingestion persistence, or web/API edit surfaces) MUST update these artifacts in the **same change**:

| Artifact | Why |
|----------|-----|
| `openspec/specs/knowledge-storage/spec.md` (+ `node-identity`, `article-translation`, or deltas in the active change) | Spec source of truth |
| `internal/kb/` (validate, create/update, frontmatter helpers) | Runtime validation |
| `internal/ingestion/`, `internal/api/`, `web/` (if the field is written or edited in UI/API) | Write and display paths |
| `.agents/skills/knowledge-db/SKILL.md` | Canonical agent instructions (English) |
| `internal/cliapp/embedskill/SKILL.md` | Must match the canonical skill byte-for-byte (`TestEmbeddedSkillMatchesCanonicalAgentsSkill`) |
| `internal/cliapp/init.go` (`--example` node) | Example must stay valid |
| `README.md` | Brief user-facing note only when the public contract changes |

After changing the skill template, users with existing KB repos should re-run `kb init --path <KB>` (overwrites `{KB}/.agents/skills/knowledge-db/SKILL.md`) or copy the skill manually.

## Backend Guidelines

- Backend code is Go under `cmd/` and `internal/`.
- **Muonsoft stack:** `github.com/muonsoft/errors`, `github.com/muonsoft/clog`, `github.com/muonsoft/api-testing` (tests). See skills `golang-errors`, `golang-logging`, `golang-tests`.
- Use package-local interfaces where they reduce coupling and make tests easier.
- Do not use `fmt.Errorf` for wrapped errors; do not use bare `slog` in `internal/` business code (bootstrap/middleware may attach loggers to context).
- Preserve offline fallback paths, especially in ingestion, search, and index-related code.
- Map `kb` sentinel errors to HTTP in handlers — see `api-conventions` and `golang-errors`.

## Frontend Guidelines

- Frontend code lives in `web/` (React 19, TypeScript, Vite, Tailwind). See skills `web-frontend`, `web-frontend-tests`, `ux-form-practices`.
- API JSON uses **snake_case** — align TypeScript types in `web/src/services/api.ts`.
- Keep operational UI quiet, dense, and practical. This is a knowledge-management tool, not a marketing site.
- Use existing routes, API clients, components, and UI patterns before adding new abstractions.
- Run `task web:test` and `task web:build` (or `npm test` / `npm run build` in `web/`) when touching `web/`.

## OpenSpec Workflow

- Use OpenSpec for non-trivial behavior changes.
- Changes live under `openspec/changes/<change-name>/`.
- Always read the relevant `proposal.md`, `design.md`, `tasks.md`, and delta specs before implementing an OpenSpec change.
- Mark tasks complete only after the implementation and verification for that task are actually done.
- Validate the change before considering it complete:

```bash
openspec validate <change-name>
openspec status --change <change-name>
```

## Useful Commands

```bash
# Run the server locally
KB_DATA_PATH=/path/to/kb go run ./cmd/kb serve
KB_GIT_DISABLED=true KB_DATA_PATH=/path/to/kb go run ./cmd/kb serve

# CLI
go run ./cmd/kb validate --path /path/to/kb
go run ./cmd/kb init --path /path/to/kb

# Build
task build

# Tests and linters
go test ./...
go test ./... -race
golangci-lint run ./...
task test
task lint

# Frontend
task web:dev
task web:build
task web:test
task web:lint

# OpenSpec
openspec list
openspec status --change <change-name>
openspec validate <change-name>
```

## Completion Checklist For Agents

Before reporting a task as done:

- Review `git diff` and make sure only relevant files changed.
- Run formatting for touched code (`gofmt` for Go; frontend formatter/linter if applicable).
- Run linters:
  - Go/backend: `golangci-lint run ./...` or `task lint`.
  - Frontend changes: `task web:lint`.
- Run tests appropriate to the change:
  - Go/backend: `go test ./...` or a narrower package set when the full suite is too expensive.
  - Race-sensitive backend changes: `go test ./... -race` when practical.
  - Frontend changes: `task web:test` and usually `task web:build`.
- For OpenSpec changes, run `openspec validate <change-name>` and check `openspec status --change <change-name>`.
- If a check cannot be run, state the reason clearly in the final response.
- Do not archive an OpenSpec change until tasks are complete, validation passes, and the user agrees to archive.
- When adding/changing environment variables or auth/access parameters, update both `README.md` and `.env.example` in the same change.
- If the change touches node frontmatter (fields, validation, or persistence): update OpenSpec specs, `.agents/skills/knowledge-db/SKILL.md`, `internal/cliapp/embedskill/SKILL.md`, and related code/tests; note whether users should re-run `kb init` on existing KB repos.

<!-- agentmem:closeout:start -->
This repository is registered in agentmem as `strider2038/knowledge-db`.
Run `@closeout for strider2038/knowledge-db` after non-trivial work (skill: `.agents/skills/closeout/SKILL.md`).
Consult `.agents/skills/agent-memory-usage/SKILL.md` for MCP usage.
<!-- agentmem:closeout:end -->
