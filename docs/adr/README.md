# Architecture Decision Records (ADR)

Этот каталог хранит архитектурные решения проекта knowledge-db в формате ADR.

## Цель

- Зафиксировать принятые архитектурные решения и их контекст.
- Дать трассируемость от текущей реализации к OpenSpec-артефактам.
- Избежать повторного обсуждения уже принятых решений без явного пересмотра.

## Как читать

1. Начинайте с таблицы индекса ниже.
2. Открывайте ADR по номеру (порядок отражает историю решений в проекте).
3. Для деталей контекста переходите в раздел `References` каждого ADR.

## Как добавлять новый ADR

1. Скопируйте [TEMPLATE.md](TEMPLATE.md).
2. Создайте новый файл `NNNN-kebab-case.md`, где `NNNN` — следующий номер.
3. Заполните разделы: `Status`, `Date`, `Context`, `Decision`, `Consequences`, `Alternatives`, `References`.
4. Добавьте запись в индексную таблицу ниже.
5. Если решение заменяет старое, обновите старый ADR:
- смените `Status` на `superseded`;
- добавьте ссылку `Superseded-By`.

## Статусы

- `accepted` — решение принято и действует.
- `superseded` — решение заменено более новым ADR.
- `deprecated` — решение устарело, но не имеет прямой замены.
- `proposed` — решение предложено, но еще не принято.

## Правило ссылок на OpenSpec

Каждый ADR обязан ссылаться на конкретные артефакты из `openspec/changes/archive/...`:
- `proposal.md`
- `design.md`
- `tasks.md`
- при необходимости `specs/.../spec.md`

Ретроспективные ADR в этом каталоге фиксируют только уже принятые решения из archived changes и не вводят новые продуктовые решения.

## Индекс ADR

| ADR | Заголовок | Status | OpenSpec changes | Supersedes | Superseded-By |
|---|---|---|---|---|---|
| [0001](0001-offline-first-git-first.md) | Offline-first + Git-first как базовый принцип | accepted | 2026-03-06-initial-project-scaffold | - | - |
| [0002](0002-knowledge-node-storage-format.md) | Формат хранения knowledge nodes (Obsidian-compatible) | accepted | 2026-03-06-obsidian-compatible-storage | - | - |
| [0003](0003-kb-server-monolith.md) | Монолит `kb-server` (API + UI + Telegram + MCP) | accepted | 2026-03-06-initial-project-scaffold | - | - |
| [0004](0004-ingestion-pipeline-llm-orchestration.md) | Ingestion pipeline с LLM orchestration и fetch-chain | accepted | 2026-03-07-implement-ingestion-pipeline, 2026-03-10-ingest-via-ui | - | - |
| [0005](0005-rag-embedding-hybrid-architecture.md) | RAG/embedding архитектура (SQLite index + hybrid retrieval) | accepted | 2026-05-02-add-rag-semantic-search, 2026-05-04-add-hybrid-search-rag-ui | - | - |
| [0006](0006-split-sqlite-index-and-chat.md) | Разделение SQLite: `index.db` и `chat.db` | accepted | 2026-05-02-add-rag-semantic-search, 2026-05-12-chat-memory-and-history | - | - |
| [0007](0007-chat-session-memory-temporary-storage.md) | Chat sessions/memory как временное SQLite-хранилище | accepted | 2026-05-12-chat-memory-and-history | - | - |
| [0008](0008-web-auth-strategy-password-and-google-oauth.md) | Auth strategy: optional session auth + Google OAuth | accepted | 2026-03-14-add-optional-web-session-auth, 2026-04-25-google-oauth-web-auth | - | - |
| [0009](0009-web-ui-build-and-embed-pipeline.md) | Web UI build/embed pipeline (Vite -> embedded static) | accepted | 2026-03-06-initial-project-scaffold, 2026-03-29-pwa-friendly | - | - |
| [0010](0010-link-article-digest-for-retrieval.md) | Эволюция link/article digest для retrieval и RAG | accepted | 2026-05-12-add-link-profile-digests | - | - |
| [0011](0011-ingestion-content-modes.md) | Content modes для ingestion (verbatim / fetch / digest / bookmark) | proposed | 2026-06-08-ingestion-content-modes | - | - |
