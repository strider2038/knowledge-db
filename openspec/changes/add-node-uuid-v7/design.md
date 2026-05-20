## Context

Узлы базы знаний идентифицируются path (`theme/slug`), производным от расположения `{slug}.md`. Локальный embedding-index (SQLite в `.kb/index.db`) использует `path` как PRIMARY KEY во всех таблицах. Ingestion всегда вызывает `CreateNode` без проверки дубликатов. Перемещение узла меняет path и триггерит удаление старой записи индекса + полную переиндексацию по новому path.

В проекте уже используется UUID v7 для request tracing (`github.com/gofrs/uuid/v5`). Jobs и chat sessions используют UUID v4/v7 отдельно.

Ограничения: git-first, Obsidian-compatible markdown, path остаётся человекочитаемым идентификатором в файловой системе и wikilinks.

## Goals / Non-Goals

**Goals:**

- Стабильный `id` (UUID v7) в frontmatter каждого узла и во всех подсистемах.
- Индекс на `node_id` PK, `path` UNIQUE; move обновляет path без потери embeddings.
- Дедуп ingestion: update по нормализованному `source_url` (если есть), иначе create; update по `id` при явной передаче.
- Один `id` на каждый `.md` (оригинал и перевод — разные id); `translation_of_id` на файлах перевода.
- Одноразовая миграция FS (`kb-cli`) + миграция/пересборка index.db.
- API: поле `id` в ответах, lookup по id.

**Non-Goals:**

- Замена wikilinks `[[slug]]` на uuid в markdown-теле.
- Общий id на «семейство» переводов (document group id) — только per-file id + `translation_of_id`.
- Дедуп note без URL по эвристикам (title similarity) — вне scope.
- `source_ids` в chat history — отдельный follow-up.
- Автообновление wikilinks в соседних файлах при move.

## Decisions

### 1. UUID v7 в frontmatter, поле `id`

**Выбор:** обязательное поле `id` (строка UUID v7, lowercase) в frontmatter главного файла узла и файла перевода.

**Почему v7:** time-order для сортировки и locality в B-tree; единый стиль с tracing.

**Библиотека:** `github.com/gofrs/uuid/v5` (`uuid.NewV7()`) — уже в go.mod для tracing.

**Альтернатива:** v4 — хуже для ordered scans; ULID — лишний новый формат.

### 2. Один id на файл, не на семейство переводов

**Выбор:** каждый `.md` — свой `id`. Перевод MAY содержать `translation_of_id` (uuid оригинала) рядом с `translation_of` (slug).

**Альтернатива:** общий document_id — ломает уникальность в индексе и API.

### 3. Дедуп ingestion: source_url на create, id на update

**Порядок при save:**

1. Если запрос содержит явный `node_id` / режим update → `GetByID` → update, id не меняется.
2. Иначе если нормализованный `source_url` непустой → lookup в `node_source_urls` → если найден, update существующего узла.
3. Иначе → create с новым `uuid.NewV7()`.

**Почему:** на первом create id ещё нет; `source_url` — естественный ключ для link/article. После create id стабилен даже при смене path.

**Конфликт двух узлов с одним url:** политика «first wins» при lookup; вторая попытка create с тем же url → update первого (документировать в spec). Validator при миграции проверяет дубликаты url.

### 4. SQLite: node_id PK, не lookup-only

**Выбор:** миграция схемы index.db:

- `indexed_nodes(node_id PK, path UNIQUE NOT NULL, …)`
- `chunks(node_id, chunk_index, …)` FK на `node_id`
- `node_search(node_id PK, path, …)`
- `node_source_urls(node_id, source_url UNIQUE)` — нормализованный url

При move: `UPDATE indexed_nodes SET path = ? WHERE node_id = ?` — embeddings сохраняются.

**Альтернатива:** отдельная lookup-таблица при path PK — два источника правды, двойная работа при move.

**Миграция старого index.db:** версия схемы; если несовместимо — `ManualRebuild` после FS migration.

### 5. Обязательность и миграция данных

**Выбор:** `CreateNode` всегда генерирует `id` если не передан. `kb-cli migrate-node-ids` — одноразовый обход всех узлов, присвоение id, отчёт о дубликатах id/url. Без lazy-on-read.

**Validator:** после миграции — отсутствие `id` = ошибка валидации.

### 6. API

- Все ответы с узлом включают `id`.
- `GET /api/nodes/by-id/{id}` — единственный способ чтения узла по id (404 если не найден).
- Существующие path-based эндпоинты сохраняются.
- Move: `id` в ответе тот же, `path` новый.

**Решение:** фильтр `GET /api/nodes?id=` **не** добавляем. Достаточно отдельного path `by-id/{id}`; список узлов остаётся с фильтрами по path/theme/type, не по id.

**Альтернатива:** query-параметр `id` на list — дублирует by-id, усложняет контракт list API.

### 7. Ingest update при дедупе

**Решение:** при update существующего узла (по `source_url` или `node_id`) система **сохраняет** текущие `theme` и `slug` (path файла). Обновляются frontmatter (кроме `id`, `created`) и markdown-body по результату LLM. LLM **не** может переместить узел в другую тему или переименовать slug через повторный ingest.

**Почему:** предсказуемость для пользователя и Obsidian; ссылки и закладки по path не ломаются неожиданно.

**Альтернатива:** применять theme/slug от LLM при update — отклонено (сюрпризы, поломка wikilinks).

### 8. SyncWorker при move

**Выбор:** вместо delete old path + index new path — событие `NodeMoved{NodeID, OldPath, NewPath}` или upsert по id с обновлением path. Старый path удаляется из индекса только если узел удалён.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Дубликаты `id` при ручном копипасте файла | Validator + migrate script проверяет глобальную уникальность |
| Два узла, один `source_url` | UNIQUE в index; ingest update первого; migrate report |
| Старый index.db без node_id | Schema migration или rebuild; документировать в README |
| Расхождение index vs FS после migrate | Запуск reindex после `migrate-node-ids` |
| Obsidian не знает про `id` | Поле опционально для Obsidian UI; не ломает совместимость |

## Migration Plan

1. Реализовать генерацию/чтение `id` в kb.Store.
2. Добавить `kb-cli migrate-node-ids` (dry-run + apply).
3. Пользователь: backup KB → migrate-node-ids → deploy server → `POST /api/index/rebuild` (или auto on first start).
4. Миграция SQLite schema при старте index (или rebuild).
5. Включить dedup в ingestion.

**Rollback:** откат кода; frontmatter `id` остаётся harmless; старый index можно восстановить из backup `.kb/index.db`.
