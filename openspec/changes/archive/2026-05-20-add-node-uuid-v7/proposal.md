## Why

Сейчас единственный идентификатор узла базы знаний — filesystem path (`theme/slug`). При перемещении, переименовании и повторном импорте ломаются внешние ссылки (чат, MCP, API), индекс пересоздаёт записи по path, а дедупликация при ingestion отсутствует. Нужен стабильный машинный идентификатор (UUID v7) в frontmatter и во всех подсистемах, сохраняя path как человекочитаемый ключ для Obsidian и git.

## What Changes

- Поле `id` (UUID v7) в frontmatter каждого узла `.md`; обязательно для новых узлов.
- Одноразовая миграция существующих узлов (`kb-cli`) — присвоение `id` всем файлам без lazy-on-read.
- Связь переводов: отдельный `id` на каждый файл перевода; опционально `translation_of_id` на файле перевода (slug `translation_of` сохраняется для Obsidian).
- SQLite-индекс: первичный ключ `node_id`, `path` UNIQUE; chunks и search привязаны к `node_id`; таблица/индекс для lookup по нормализованному `source_url` (дедуп ingestion).
- Ingestion: при наличии `source_url` — update существующего узла вместо create; update по `id` при явном указании; иначе create с новым id.
- REST API и типы ответов: поле `id`; lookup узла по id; path остаётся в URL и ответах.
- Move/delete: `id` не меняется, обновляется только `path` в индексе.
- **BREAKING**: схема локального embedding-index (пересборка/миграция `.kb/index.db`); контракт API (новое поле, новые эндпоинты lookup).

## Capabilities

### New Capabilities

- `node-identity`: правила UUID v7, уникальность, миграция данных, дедупликация create/update, связь оригинал↔перевод через `translation_of_id`.

### Modified Capabilities

- `knowledge-storage`: поле `id` в frontmatter, валидация, CreateNode/чтение, `translation_of_id`.
- `embedding-index`: схема БД на `node_id`, синхронизация и move без потери embeddings по смене path.
- `rest-api`: `id` в моделях узла, GET по id, поведение move с стабильным id.
- `ingestion-pipeline`: дедуп по `source_url` (create/update), сохранение id при update.
- `kb-cli`: команда миграции присвоения id существующим узлам.
- `node-move`: стабильность `id` при перемещении.
- `mcp-server`: опциональный параметр/id в ответах инструментов чтения узла.

## Impact

- `internal/kb/` — store, validator, CreateNode, translations, MoveNode
- `internal/index/` — sqlite schema, sync worker, retrieval, search
- `internal/ingestion/` — saveNode, dedup lookup
- `internal/api/`, `internal/mcp/`
- `cmd/kb-cli/` — migrate command
- `web/` — типы Node, при необходимости API client
- Пользовательские KB: одноразовый запуск миграции + reindex после деплоя
- Зависимость: `github.com/gofrs/uuid/v5` (v7 уже используется в tracing) или `google/uuid` — унификация в design
