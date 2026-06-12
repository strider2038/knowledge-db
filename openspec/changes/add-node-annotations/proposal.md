## Why

При чтении узлов в базе знаний пользователю нужно оставлять личные аннотации — общие размышления по материалу и комментарии к конкретным фрагментам текста — без изменения основного markdown-тела узла и без смешения с полем `annotation` (краткое описание источника). Сейчас такой возможности нет: страница узла read-only для контента, а существующие `labels` и `manual_processed` решают другую задачу (разметка и workflow).

## What Changes

- Добавить git-tracked sidecar `{slug}/annotations.yaml` рядом с узлом для личных аннотаций (plain text).
- Поддержать два типа аннотаций: **общие** (к узлу целиком) и **привязанные к фрагменту** (якорь `text_quote` с цитатой и контекстом).
- Одна лента аннотаций на логический узел (базовый путь без `.ru` и др. переводов); при move узла переносить sidecar вместе с вложениями.
- REST API: CRUD аннотаций (`GET/POST/PATCH/DELETE /api/nodes/{basePath}/annotations`).
- Web UI: панель «Заметки» на странице узла; выделение текста в «Содержании» → новая привязанная аннотация; jump между заметкой и цитатой; индикация устаревших якорей.
- Аннотации **не** участвуют в hybrid search, FTS, embedding и RAG.
- Формат тела заметки в MVP — plain text (без markdown).

## Capabilities

### New Capabilities

- `node-annotations`: хранение, валидация, API и UI личных аннотаций к узлам (общие и привязанные к фрагменту).

### Modified Capabilities

- `knowledge-storage`: структура узла — sidecar `annotations.yaml` в директории вложений.
- `rest-api`: эндпоинты CRUD аннотаций.
- `webapp`: панель заметок и взаимодействие с контентом на странице узла.
- `node-move`: перенос `annotations.yaml` при перемещении узла.
- `embedding-index`: явное исключение аннотаций из индексируемого контента.

## Impact

- `internal/kb` — чтение/запись `annotations.yaml`, resolve якорей, перенос при move/delete.
- `internal/api` — handlers и маршруты `/annotations`.
- `web/src` — `NodePage`, компоненты панели заметок, обёртка выделения текста в `MarkdownContent`, `api.ts`.
- OpenSpec main specs после архивации change.
- Без изменений: Telegram, MCP, поиск, чат/RAG, ingestion.
