# embedding-index Specification

## Purpose

Embedding-индекс на SQLite: генерация векторных представлений, markdown-aware chunking, SyncWorker для синхронизации с git-репозиторием, keyword/FTS поиск, feature toggle для отключаемости.
## Requirements
### Requirement: EmbeddingProvider — интерфейс генерации эмбеддингов

Система ДОЛЖНА (SHALL) предоставлять интерфейс `EmbeddingProvider` для генерации векторных представлений текста. Интерфейс SHALL содержать метод `Embed(ctx, texts []string) ([][]float32, error)`. Система MUST предоставлять реализацию `APIProvider`, использующую OpenAI-совместимое API (OpenRouter) через конфигурацию: `KB_EMBEDDING_API_KEY`, `KB_EMBEDDING_API_URL`, `KB_EMBEDDING_MODEL`.

#### Scenario: Генерация эмбеддингов через API

- **WHEN** вызывается `EmbedProvider.Embed()` с массивом текстов и настроенным APIProvider
- **THEN** возвращается массив векторных представлений (по одному на каждый текст) с размерностью, соответствующей модели

#### Scenario: API недоступен

- **WHEN** вызывается `EmbedProvider.Embed()` и API-запрос завершается ошибкой
- **THEN** возвращается ошибка без записи в индекс

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
- **THEN** миграция переводит схему на node_id PK или документированно требует rebuild (`kb rebuild-index` или `POST /api/index/rebuild`); markdown-файлы не изменяются автоматически

### Requirement: Markdown-aware chunking

Система ДОЛЖНА (SHALL) предоставлять chunker для разбиения body нод типа `article` на фрагменты. Chunker SHALL использовать заголовки `##` как границы чанков. Секции более ~500 токенов MUST разбиваться по параграфам. Секции менее ~100 токенов MUST объединяться со следующей секцией. Для нод типа `link` (body пустой) и `note` (короткий body) chunking НЕ ДОЛЖЕН применяться — embedding генерируется для полного текста ноды.

#### Scenario: Статья с заголовками ##

- **WHEN** chunker обрабатывает статью с тремя секциями `##`
- **THEN** создаётся три чанка, каждый содержит заголовок секции и её содержимое

#### Scenario: Статья без заголовков

- **WHEN** chunker обрабатывает статью без `##` заголовков
- **THEN** создаётся один чанк, содержащий весь body

#### Scenario: Link-нода без body

- **WHEN** chunker обрабатывает ноду типа `link` с пустым body
- **THEN** чанки не создаются, генерируется только node embedding

#### Scenario: Note-нода с коротким body

- **WHEN** chunker обрабатывает ноду типа `note` с body < 500 токенов
- **THEN** чанки не создаются, body включается целиком в node embedding

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

Система ДОЛЖНА (SHALL) предоставлять SyncWorker (реализующий `runnable.Runnable`) для синхронизации индекса с git-репозиторием. SyncWorker MUST обрабатывать события: SingleNode(path) — индексация одной ноды по path с резолвом node_id из frontmatter; GitSyncDiff — diff после git pull; FullReconcile — полная сверка по FS с node_id; ManualRebuild — полная перестройка. SyncWorker MUST экспортировать синхронный метод `ManualRebuild(ctx)` (очистка индекса + FullReconcile) для CLI `kb rebuild-index`. SyncWorker MUST ограничивать частоту запросов к embedding API (rate limit: не более 1 batch/сек). SyncWorker MUST логировать warn при ошибках синхронизации.

#### Scenario: Триггер после создания ноды

- **WHEN** API handler создаёт ноду через kb.Store
- **THEN** SyncWorker получает событие SingleNode и индексирует ноду по node_id

#### Scenario: Триггер после ingest

- **WHEN** ingestion pipeline создаёт или обновляет ноду на диске (UI `/api/ingest`, Telegram, импорт Telegram)
- **THEN** SyncWorker получает событие SingleNode для path ноды

#### Scenario: Триггер после удаления ноды через API

- **WHEN** `DELETE /api/nodes/{path}` успешно удаляет markdown-файл
- **THEN** SyncWorker получает событие SingleNode для path ноды и удаляет запись из индекса (включая `node_source_urls`)

#### Scenario: Триггер после изменения контента job-операциями

- **WHEN** node normalization, agent edit или dump images успешно изменили markdown узла на диске
- **THEN** SyncWorker получает событие SingleNode для path ноды

#### Scenario: Триггер после перевода статьи

- **WHEN** translation worker создал файл перевода или обновил оригинал
- **THEN** SyncWorker получает событие SingleNode для path оригинала и path перевода (`{slug}.ru`)

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

#### Scenario: Manual rebuild через API

- **WHEN** вызывается `POST /api/index/rebuild` при работающем `kb serve`
- **THEN** SyncWorker получает событие ManualRebuild, очищает индекс и выполняет полную переиндексацию всех нод с валидным id; API возвращает 202 Accepted

#### Scenario: Manual rebuild через CLI

- **WHEN** выполняется `kb rebuild-index --path /data/kb` при настроенных embedding-переменных
- **THEN** вызывается `SyncWorker.ManualRebuild` синхронно: индекс очищается и все ноды с валидным `id` переиндексируются без запущенного HTTP-сервера

#### Scenario: Rate limiting при batch

- **WHEN** FullReconcile обрабатывает 100 нод
- **THEN** embedding API вызывается с rate limit не более 1 batch/сек

#### Scenario: Удаление узла

- **WHEN** узел удалён из FS
- **THEN** SyncWorker удаляет записи indexed_nodes, chunks, node_search и node_source_urls по node_id

### Requirement: Векторный поиск

Система ДОЛЖНА (SHALL) предоставлять двухуровневый векторный поиск. Level 1 (node-level): cosine similarity между embedding запроса и node embeddings → top-K нод. Level 2 (chunk-level): cosine similarity между embedding запроса и chunk embeddings → top-K фрагментов. Cosine similarity MUST вычисляться в Go-коде (без sqlite-vec) через linear scan по BLOB-векторам. Результаты vector search MUST быть пригодны для передачи в общий hybrid retrieval pipeline вместе с keyword/FTS кандидатами.

#### Scenario: Поиск релевантных нод

- **WHEN** вызывается VectorSearch с текстом запроса и K=5
- **THEN** возвращаются 5 нод с наибольшей cosine similarity, каждая содержит path, title, annotation, score

#### Scenario: Поиск релевантных фрагментов

- **WHEN** вызывается ChunkSearch с текстом запроса и K=5
- **THEN** возвращаются 5 чанков с наибольшей cosine similarity, каждый содержит node_path, heading, content, score

#### Scenario: Пустой индекс

- **WHEN** вызывается VectorSearch при пустом индексе
- **THEN** возвращается пустой результат без ошибки

#### Scenario: Передача в hybrid retrieval

- **WHEN** hybrid retrieval вызывает vector search
- **THEN** vector results содержат score и source kind, достаточные для fusion/ranking

### Requirement: Feature toggle — отключаемость embeddings

Функциональность embeddings ДОЛЖНА (SHALL) быть полностью отключаемой. При `KB_EMBEDDING_ENABLED=false` (default) система MUST NOT создавать SQLite базу, MUST NOT запускать SyncWorker, endpoints `/api/chat` и `/api/index/*` MUST возвращать 503. При `KB_EMBEDDING_ENABLED=true` система MUST требовать `KB_EMBEDDING_API_KEY` и `KB_EMBEDDING_API_URL`.

#### Scenario: Embeddings отключены

- **WHEN** `kb serve` запускается без `KB_EMBEDDING_ENABLED=true`
- **THEN** SQLite не создаётся, SyncWorker не запускается, `/api/chat` → 503

#### Scenario: Embeddings включены без API key

- **WHEN** `kb serve` запускается с `KB_EMBEDDING_ENABLED=true` но без `KB_EMBEDDING_API_KEY`
- **THEN** сервер возвращает ошибку конфигурации и не стартует

### Requirement: Состояние индекса

Система ДОЛЖНА (SHALL) предоставлять информацию о состоянии индекса через `GET /api/index/status`. Ответ MUST содержать: total_nodes — количество проиндексированных нод, total_chunks — количество чанков, embedding_model — имя модели, last_indexed_at — время последней индексации, status — `ready` или `indexing`. Ответ MUST также сообщать режим keyword index (`fts5`, `scan` или `disabled`), чтобы UI мог понимать доступность гибридного поиска.

#### Scenario: Проверка статуса индекса

- **WHEN** `GET /api/index/status` при проиндексированной базе
- **THEN** возвращается JSON с total_nodes, total_chunks, embedding_model, keyword_index, last_indexed_at, status: "ready"

#### Scenario: Статус при отключённых embeddings

- **WHEN** `GET /api/index/status` при `KB_EMBEDDING_ENABLED=false`
- **THEN** возвращается 503 с сообщением о недоступности

### Requirement: Keyword/FTS индекс

Система ДОЛЖНА (SHALL) предоставлять keyword/FTS поиск по индексированным нодам и чанкам. Индекс MUST поддерживать поиск по точным словам и фразам в `title`, `aliases`, `keywords`, `annotation`, `path`, `source_url`, `heading` и `content`. Если FTS5 недоступен, система MUST использовать fallback keyword scan по сохранённому searchable text.

#### Scenario: Поиск по keyword

- **WHEN** пользователь ищет слово, присутствующее в `keywords` ноды
- **THEN** keyword/FTS поиск возвращает эту ноду как кандидата

#### Scenario: Поиск по chunk content

- **WHEN** пользователь ищет фразу, присутствующую только в body статьи
- **THEN** keyword/FTS поиск возвращает article-ноду с фрагментом соответствующего chunk

#### Scenario: FTS fallback

- **WHEN** FTS5 недоступен в SQLite окружении
- **THEN** keyword поиск продолжает работать через fallback scan без внешних сервисов

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

