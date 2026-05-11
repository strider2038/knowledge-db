## MODIFIED Requirements

### Requirement: SQLite-схема индекса

Система ДОЛЖНА (SHALL) создавать SQLite-базу данных по пути `{data_path}/.kb/index.db` при `KB_EMBEDDING_ENABLED=true`. Схема SHALL содержать таблицы: `indexed_nodes` (path PK, content_hash, body_hash, indexed_at, node_embedding_id), `chunks` (id PK, node_path, chunk_index, heading, content, embedding_id; UNIQUE node_path+chunk_index), `embeddings` (id PK, vector BLOB, model, dimensions). Схема MUST также хранить searchable text для нод и чанков, достаточный для keyword/FTS поиска по `path`, `title`, `aliases`, `annotation`, `keywords`, `type`, `source_url`, `heading` и `content`. Система MUST использовать WAL mode для конкурентного доступа.

#### Scenario: Создание базы при первом запуске

- **WHEN** kb-server запускается с `KB_EMBEDDING_ENABLED=true` и файл index.db не существует
- **THEN** создаётся директория `.kb/` и файл `index.db` с полной схемой для embeddings, chunks и searchable text

#### Scenario: Директория .kb в gitignore

- **WHEN** создаётся директория `{data_path}/.kb/`
- **THEN** она добавляется в `.gitignore` репозитория данных (если управляется git)

#### Scenario: Миграция существующего индекса

- **WHEN** kb-server запускается с существующим index.db без searchable text таблиц
- **THEN** миграция добавляет необходимые таблицы/индексы без изменения markdown-файлов базы

### Requirement: Embedding-индекс для нод

Система ДОЛЖНА (SHALL) генерировать и хранить embedding для каждой ноды. Текст для embedding SHALL формироваться: `title + " " + annotation + " " + keywords` — для всех типов; для `note` — дополнительно `+ " " + body`. Embedding MUST храниться как BLOB (little-endian float32 array) в таблице embeddings. Система MUST хранить content_hash (hash от title+annotation+keywords+type) и body_hash (hash от body) для определения изменений. При индексации ноды система MUST обновлять searchable text записи для keyword/FTS поиска.

#### Scenario: Индексация новой ноды

- **WHEN** SyncWorker обрабатывает ноду, отсутствующую в indexed_nodes
- **THEN** генерируется node embedding, создаётся запись в indexed_nodes и embeddings, а searchable text ноды доступен для keyword/FTS поиска

#### Scenario: Обновление при изменении метаданных

- **WHEN** нода существует в indexed_nodes, но content_hash отличается
- **THEN** node embedding и searchable text ноды пересчитываются, indexed_nodes обновляется

#### Scenario: Обновление при изменении body статьи

- **WHEN** нода типа article существует в indexed_nodes, но body_hash отличается
- **THEN** старые чанки и их searchable text удаляются, генерируются новые чанки и их embeddings

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

### Requirement: Состояние индекса

Система ДОЛЖНА (SHALL) предоставлять информацию о состоянии индекса через `GET /api/index/status`. Ответ MUST содержать: total_nodes — количество проиндексированных нод, total_chunks — количество чанков, embedding_model — имя модели, last_indexed_at — время последней индексации, status — `ready` или `indexing`. Ответ MUST также сообщать режим keyword index (`fts5`, `scan` или `disabled`), чтобы UI мог понимать доступность гибридного поиска.

#### Scenario: Проверка статуса индекса

- **WHEN** `GET /api/index/status` при проиндексированной базе
- **THEN** возвращается JSON с total_nodes, total_chunks, embedding_model, keyword_index, last_indexed_at, status: "ready"

#### Scenario: Статус при отключённых embeddings

- **WHEN** `GET /api/index/status` при `KB_EMBEDDING_ENABLED=false`
- **THEN** возвращается 503 с сообщением о недоступности

## ADDED Requirements

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
