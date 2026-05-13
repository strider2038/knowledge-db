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
├── cmd/
│   ├── kb-server/   # HTTP API, embedded UI, Telegram bot, MCP endpoint
│   └── kb-cli/      # validate, init and maintenance commands
├── internal/
│   ├── api/         # HTTP handlers and routing
│   ├── bootstrap/   # application wiring and config
│   ├── index/       # local search index, retrieval, embeddings
│   ├── ingestion/   # ingestion pipeline, fetchers, LLM orchestration
│   ├── kb/          # filesystem store, frontmatter, validation, tree
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
6. **Local by default**: `kb-server`, web UI, Telegram bot, and MCP are designed around localhost/private or self-hosted use.
7. **Small, explicit changes**: keep implementation scoped to the relevant package and update specs/docs when behavior or contracts change.

## Language Conventions

- Code identifiers and Go/TypeScript comments should follow the local style already used in the package.
- OpenSpec artifacts (`proposal.md`, `design.md`, `tasks.md`, delta specs) are written in Russian.
- User-facing knowledge-base content is usually Russian unless the source or existing entry style says otherwise.
- General repository documentation may be English when it is meant for agents or tooling.

## Working With The Knowledge Base

- Do not assume the knowledge base is in `./data`. Use `KB_DATA_PATH` or the path provided by the user.
- Reading and writing knowledge-base files directly is allowed only when it is part of the requested workflow.
- Keep markdown/frontmatter compatible with the current specs and validator.
- When introducing new frontmatter fields, entry formats, or persistence behavior, update OpenSpec and relevant documentation.
- Apply markdown normalization rules from `.cursor/rules/markdown-normalization.mdc` when normalizing entries, OpenSpec documents, or notes in an opened knowledge-base repository.

## Backend Guidelines

- Backend code is Go under `cmd/` and `internal/`.
- Use package-local interfaces where they reduce coupling and make tests easier.
- Use `github.com/muonsoft/errors` for wrapping errors.
- Use contextual logging through `github.com/muonsoft/clog`; do not add direct `slog` calls in business logic unless the surrounding package already does so for bootstrap/setup.
- Preserve offline fallback paths, especially in ingestion, search, and index-related code.
- Prefer focused tests for behavior changes. Use `testify` assertions, and prefer in-memory `afero` filesystems for `kb.Store` tests.

## Frontend Guidelines

- Frontend code lives in `web/` and uses React, TypeScript, Vite, and the existing component/style conventions.
- Keep operational UI quiet, dense, and practical. This is a knowledge-management tool, not a marketing site.
- Use existing routes, API clients, components, and UI patterns before adding new abstractions.
- Run frontend checks when touching `web/`.

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
go run ./cmd/kb-server
KB_DATA_PATH=/path/to/kb go run ./cmd/kb-server
KB_GIT_DISABLED=true go run ./cmd/kb-server

# CLI
go run ./cmd/kb-cli validate --path /path/to/kb
go run ./cmd/kb-cli init --path /path/to/kb

# Build
task build
task build-server
task build-cli

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
