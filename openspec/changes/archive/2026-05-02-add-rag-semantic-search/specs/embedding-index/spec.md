## ADDED Requirements

SQLite-индекс для семантического поиска: хранение embeddings и chunks, синхронизация с git-репозиторием, markdown-aware chunking, интерфейс EmbeddingProvider.

### Requirement: EmbeddingProvider — интерфейс генерации эмбеддингов

Система ДОЛЖНА (SHALL) предоставлять интерфейс `EmbeddingProvider` для генерации векторных представлений текста. Интерфейс SHALL содержать метод `Embed(ctx, texts []string) ([][]float32, error)`. Система MUST предоставлять реализацию `APIProvider`, использующую OpenAI-совместимое API (OpenRouter) через конфигурацию: `KB_EMBEDDING_API_KEY`, `KB_EMBEDDING_API_URL`, `KB_EMBEDDING_MODEL`.

#### Scenario: Генерация эмбеддингов через API

- **WHEN** вызывается `EmbedProvider.Embed()` с массивом текстов и настроенным APIProvider
- **THEN** возвращается массив векторных представлений (по одному на каждый текст) с размерностью, соответствующей модели

#### Scenario: API недоступен

- **WHEN** вызывается `EmbedProvider.Embed()` и API-запрос завершается ошибкой
- **THEN** возвращается ошибка без записи в индекс

### Requirement: SQLite-схема индекса

Система ДОЛЖНА (SHALL) создавать SQLite-базу данных по пути `{data_path}/.kb/index.db` при `KB_EMBEDDING_ENABLED=true`. Схема SHALL содержать таблицы: `indexed_nodes` (path PK, content_hash, body_hash, indexed_at, node_embedding_id), `chunks` (id PK, node_path, chunk_index, heading, content, embedding_id; UNIQUE node_path+chunk_index), `embeddings` (id PK, vector BLOB, model, dimensions). Система MUST использовать WAL mode для конкурентного доступа.

#### Scenario: Создание базы при первом запуске

- **WHEN** kb-server запускается с `KB_EMBEDDING_ENABLED=true` и файл index.db не существует
- **THEN** создаётся директория `.kb/` и файл `index.db` с полной схемой

#### Scenario: Директория .kb в gitignore

- **WHEN** создаётся директория `{data_path}/.kb/`
- **THEN** она добавляется в `.gitignore` репозитория данных (если управляется git)

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

Система ДОЛЖНА (SHALL) генерировать и хранить embedding для каждой ноды. Текст для embedding SHALL формироваться: `title + " " + annotation + " " + keywords` — для всех типов; для `note` — дополнительно `+ " " + body`. Embedding MUST храниться как BLOB (little-endian float32 array) в таблице embeddings. Система MUST хранить content_hash (hash от title+annotation+keywords+type) и body_hash (hash от body) для определения изменений.

#### Scenario: Индексация новой ноды

- **WHEN** SyncWorker обрабатывает ноду, отсутствующую в indexed_nodes
- **THEN** генерируется node embedding, создаётся запись в indexed_nodes и embeddings

#### Scenario: Обновление при изменении метаданных

- **WHEN** нода существует в indexed_nodes, но content_hash отличается
- **THEN** node embedding пересчитывается, indexed_nodes обновляется

#### Scenario: Обновление при изменении body статьи

- **WHEN** нода типа article существует в indexed_nodes, но body_hash отличается
- **THEN** старые чанки удаляются, генерируются новые чанки и их embeddings

### Requirement: SyncWorker — синхронизация индекса

Система ДОЛЖНА (SHALL) предоставлять SyncWorker (реализующий `runnable.Runnable`) для синхронизации индекса с git-репозиторием. SyncWorker MUST обрабатывать события: SingleNode(path) — индексация одной ноды; GitSyncDiff — diff после git pull; FullReconcile — полная сверка; ManualRebuild — полная перестройка. SyncWorker MUST ограничивать частоту запросов к embedding API (rate limit: не более 1 batch/сек). SyncWorker MUST логировать warn при ошибках синхронизации.

#### Scenario: Триггер после создания ноды

- **WHEN** API handler создаёт ноду через kb.Store
- **THEN** SyncWorker получает событие SingleNode и индексирует ноду

#### Scenario: Триггер после git sync

- **WHEN** GitSyncRunner завершает pull
- **THEN** SyncWorker получает событие GitSyncDiff и индексирует изменённые ноды

#### Scenario: Periodic reconciliation

- **WHEN** срабатывает periodic timer (раз в сутки)
- **THEN** SyncWorker выполняет FullReconcile: walk FS, сверяет с indexed_nodes, добавляет/обновляет/удаляет

#### Scenario: Manual rebuild

- **WHEN** вызывается `POST /api/index/rebuild`
- **THEN** SyncWorker очищает индекс и выполняет полную переиндексацию всех нод

#### Scenario: Rate limiting при batch

- **WHEN** FullReconcile обрабатывает 100 нод
- **THEN** embedding API вызывается с rate limit не более 1 batch/сек

### Requirement: Векторный поиск

Система ДОЛЖНА (SHALL) предоставлять двухуровневый векторный поиск. Level 1 (node-level): cosine similarity между embedding запроса и node embeddings → top-K нод. Level 2 (chunk-level): cosine similarity между embedding запроса и chunk embeddings → top-K фрагментов. Cosine similarity MUST вычисляться в Go-коде (без sqlite-vec) через linear scan по BLOB-векторам.

#### Scenario: Поиск релевантных нод

- **WHEN** вызывается VectorSearch с текстом запроса и K=5
- **THEN** возвращаются 5 нод с наибольшей cosine similarity, каждая содержит path, title, annotation, score

#### Scenario: Поиск релевантных фрагментов

- **WHEN** вызывается ChunkSearch с текстом запроса и K=5
- **THEN** возвращаются 5 чанков с наибольшей cosine similarity, каждый содержит node_path, heading, content, score

#### Scenario: Пустой индекс

- **WHEN** вызывается VectorSearch при пустом индексе
- **THEN** возвращается пустой результат без ошибки

### Requirement: Feature toggle — отключаемость embeddings

Функциональность embeddings ДОЛЖНА (SHALL) быть полностью отключаемой. При `KB_EMBEDDING_ENABLED=false` (default) система MUST NOT создавать SQLite базу, MUST NOT запускать SyncWorker, endpoints `/api/chat` и `/api/index/*` MUST возвращать 503. При `KB_EMBEDDING_ENABLED=true` система MUST требовать `KB_EMBEDDING_API_KEY` и `KB_EMBEDDING_API_URL`.

#### Scenario: Embeddings отключены

- **WHEN** kb-server запускается без `KB_EMBEDDING_ENABLED=true`
- **THEN** SQLite не создаётся, SyncWorker не запускается, `/api/chat` → 503

#### Scenario: Embeddings включены без API key

- **WHEN** kb-server запускается с `KB_EMBEDDING_ENABLED=true` но без `KB_EMBEDDING_API_KEY`
- **THEN** сервер возвращает ошибку конфигурации и не стартует

### Requirement: Состояние индекса

Система ДОЛЖНА (SHALL) предоставлять информацию о состоянии индекса через `GET /api/index/status`. Ответ MUST содержать: total_nodes — количество проиндексированных нод, total_chunks — количество чанков, embedding_model — имя модели, last_indexed_at — время последней индексации, status — `ready` или `indexing`.

#### Scenario: Проверка статуса индекса

- **WHEN** `GET /api/index/status` при проиндексированной базе
- **THEN** возвращается JSON с total_nodes, total_chunks, embedding_model, last_indexed_at, status: "ready"

#### Scenario: Статус при отключённых embeddings

- **WHEN** `GET /api/index/status` при `KB_EMBEDDING_ENABLED=false`
- **THEN** возвращается 503 с сообщением о недоступности
