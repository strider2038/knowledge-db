## Purpose

Дельта: первичный ключ индекса `node_id`, стабильность embeddings при move, lookup по source_url.

## ADDED Requirements

### Requirement: Таблица node_source_urls для дедупа

Схема индекса MUST содержать таблицу `node_source_urls` с полями `node_id` (FK на indexed_nodes) и `source_url` (нормализованный URL, UNIQUE). При индексации узла с непустым `source_url` система MUST upsert запись в этой таблице. При удалении узла запись MUST удаляться.

#### Scenario: Индексация узла с source_url

- **WHEN** SyncWorker индексирует узел с `source_url` в frontmatter
- **THEN** в `node_source_urls` существует связка node_id + нормализованный source_url

#### Scenario: Lookup для ingestion

- **WHEN** ingestion ищет узел по source_url
- **THEN** Store или IndexStore возвращает node_id и path существующего узла

### Requirement: Резолв узла по node_id

Index Store MUST предоставлять операции GetNodeByID, UpdateNodePath(node_id, new_path) для обновления path без удаления embeddings.

#### Scenario: Обновление path при move

- **WHEN** узел с node_id перемещён на новый path
- **THEN** indexed_nodes.path обновлён, node_id и node_embedding_id не изменились

## MODIFIED Requirements

### Requirement: SQLite-схема индекса

Система ДОЛЖНА (SHALL) создавать SQLite-базу данных по пути `{data_path}/.kb/index.db` при `KB_EMBEDDING_ENABLED=true`. Схема SHALL содержать таблицы: `indexed_nodes` (**node_id** TEXT PRIMARY KEY, **path** TEXT NOT NULL UNIQUE, content_hash, body_hash, indexed_at, node_embedding_id), `chunks` (id PK, **node_id**, chunk_index, heading, content, embedding_id; UNIQUE **node_id**+chunk_index), `embeddings` (id PK, vector BLOB, model, dimensions), **`node_source_urls`** (node_id, source_url UNIQUE). Схема MUST также хранить searchable text для нод и чанков в таблицах с привязкой к **node_id** (и денормализованным path для отображения), достаточный для keyword/FTS поиска по `path`, `title`, `aliases`, `annotation`, `keywords`, `type`, `source_url`, `heading` и `content`. Система MUST использовать WAL mode для конкурентного доступа.

#### Scenario: Создание базы при первом запуске

- **WHEN** `kb serve` запускается с `KB_EMBEDDING_ENABLED=true` и файл index.db не существует
- **THEN** создаётся директория `.kb/` и файл `index.db` с полной схемой включая node_id PK

#### Scenario: Директория .kb в gitignore

- **WHEN** создаётся директория `{data_path}/.kb/`
- **THEN** она добавляется в `.gitignore` репозитория данных (если управляется git)

#### Scenario: Миграция существующего индекса

- **WHEN** `kb serve` запускается с существующим index.db со схемой path PK
- **THEN** миграция переводит схему на node_id PK или документированно требует rebuild; markdown-файлы не изменяются автоматически

### Requirement: Embedding-индекс для нод

Система ДОЛЖНА (SHALL) генерировать и хранить embedding для каждой ноды, идентифицируемой **node_id** из frontmatter. Текст для embedding SHALL формироваться: `title + " " + annotation + " " + keywords` — для всех типов; для `note` — дополнительно `+ " " + body`; для `link` — дополнительно `+ " " + body`, если узел содержит `content_profile` и непустое markdown-тело. Поле `labels` MUST NOT входить в текст embedding и MUST NOT входить в searchable text для смыслового/keyword поиска по контенту. Embedding MUST храниться как BLOB (little-endian float32 array) в таблице embeddings. Система MUST хранить content_hash (hash от title+annotation+keywords+type+source_kind+content_profile; **без** labels и **без** id) и body_hash (hash от body) для определения изменений. При индексации ноды система MUST читать `id` из frontmatter и использовать его как node_id в indexed_nodes. При индексации система MUST обновлять searchable text и node_source_urls. Searchable text MUST включать `source_kind`, `content_profile` и body для `note` и профильных `link` узлов. Система MUST создавать chunks для `article`, а также для `note` и `link` узлов с digest body, если body достаточно длинное для chunking.

#### Scenario: Индексация новой ноды

- **WHEN** SyncWorker обрабатывает ноду с frontmatter id, отсутствующую в indexed_nodes
- **THEN** генерируется node embedding, создаётся запись с node_id=id, path из FS, embeddings и searchable text

#### Scenario: Обновление при изменении метаданных

- **WHEN** нода существует в indexed_nodes по node_id, но content_hash отличается
- **THEN** node embedding и searchable text пересчитываются, node_id не меняется

#### Scenario: Обновление при изменении body статьи

- **WHEN** нода типа article существует в indexed_nodes, но body_hash отличается
- **THEN** старые чанки удаляются по node_id, генерируются новые чанки и embeddings

#### Scenario: Индексация repository profile body

- **WHEN** нода `type=link` содержит `content_profile=repository_profile` и markdown-тело
- **THEN** body включается в node embedding, searchable text и chunk index при достаточном размере

#### Scenario: Индексация conceptual digest note

- **WHEN** нода `type=note` содержит `content_profile=conceptual_digest` и markdown-тело
- **THEN** body включается в node embedding, searchable text и chunk index при достаточном размере

#### Scenario: Старый link без digest

- **WHEN** нода `type=link` не содержит `content_profile` и имеет пустое тело
- **THEN** индексирование продолжает использовать title, annotation и keywords без ошибки

#### Scenario: Изменение только labels

- **WHEN** у узла изменилось только поле labels во frontmatter
- **THEN** content_hash не меняется и переиндексация embedding не выполняется

#### Scenario: Узел без id в frontmatter

- **WHEN** SyncWorker обрабатывает ноду без поля id (до миграции)
- **THEN** SyncWorker MUST пропустить индексацию с предупреждением в лог или использовать одноразовую политику миграции (см. node-identity)

### Requirement: SyncWorker — синхронизация индекса

Система ДОЛЖНА (SHALL) предоставлять SyncWorker (реализующий `runnable.Runnable`) для синхронизации индекса с git-репозиторием. SyncWorker MUST обрабатывать события: SingleNode(path) — индексация одной ноды по path с резолвом node_id из frontmatter; GitSyncDiff — diff после git pull; FullReconcile — полная сверка по FS с node_id; ManualRebuild — полная перестройка. SyncWorker MUST ограничивать частоту запросов к embedding API (rate limit: не более 1 batch/сек). SyncWorker MUST логировать warn при ошибках синхронизации.

#### Scenario: Триггер после создания ноды

- **WHEN** API handler создаёт ноду через kb.Store
- **THEN** SyncWorker получает событие SingleNode и индексирует ноду по node_id

#### Scenario: Триггер после перемещения ноды

- **WHEN** API handler успешно перемещает ноду из `old/path` в `new/path` с неизменным `id`
- **THEN** SyncWorker MUST обновить path в indexed_nodes, node_search и chunk_search для того же node_id
- **AND** embeddings и chunk embeddings MUST NOT удаляться только из-за смены path

#### Scenario: Триггер после git sync

- **WHEN** GitSyncRunner завершает pull
- **THEN** SyncWorker получает событие GitSyncDiff и индексирует изменённые ноды

#### Scenario: Periodic reconciliation

- **WHEN** срабатывает periodic timer (раз в сутки)
- **THEN** SyncWorker выполняет FullReconcile: walk FS, сверяет по node_id и path, добавляет/обновляет/удаляет

#### Scenario: Manual rebuild

- **WHEN** вызывается `POST /api/index/rebuild`
- **THEN** SyncWorker очищает индекс и выполняет полную переиндексацию всех нод с валидным id

#### Scenario: Rate limiting при batch

- **WHEN** FullReconcile обрабатывает 100 нод
- **THEN** embedding API вызывается с rate limit не более 1 batch/сек

#### Scenario: Удаление узла

- **WHEN** узел удалён из FS
- **THEN** SyncWorker удаляет записи indexed_nodes, chunks, node_search и node_source_urls по node_id
