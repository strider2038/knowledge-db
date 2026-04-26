## 1. Инфраструктура и конфигурация

- [x] 1.1 Добавить зависимость `modernc.org/sqlite` (pure-Go SQLite драйвер) в go.mod
- [x] 1.2 Добавить секцию `Embedding` в `internal/bootstrap/config/config.go`: поля `Enabled`, `APIKey`, `APIURL`, `Model`, `ChatModel` (env vars: `KB_EMBEDDING_ENABLED`, `KB_EMBEDDING_API_KEY`, `KB_EMBEDDING_API_URL`, `KB_EMBEDDING_MODEL`, `KB_CHAT_MODEL`)
- [x] 1.3 Добавить валидацию: при `KB_EMBEDDING_ENABLED=true` обязательны `KB_EMBEDDING_API_KEY` и `KB_EMBEDDING_API_URL`

## 2. EmbeddingProvider — интерфейс и API-реализация

- [x] 2.1 Создать пакет `internal/index/`, определить интерфейс `EmbeddingProvider` с методом `Embed(ctx context.Context, texts []string) ([][]float32, error)`
- [x] 2.2 Реализовать `APIProvider` — OpenAI-совместимый клиент для генерации эмбеддингов (HTTP POST к `/v1/embeddings`, batch-запросы)
- [x] 2.3 Написать unit-тесты для `APIProvider` (mock HTTP server)

## 3. SQLite IndexStore — схема и базовые операции

- [x] 3.1 Создать `internal/index/store.go` — `IndexStore` с подключением к SQLite (WAL mode), миграцией схемы (indexed_nodes, chunks, embeddings)
- [x] 3.2 Реализовать CRUD операции: `UpsertNode`, `DeleteNode`, `GetNodeByPath`, `ListAllIndexed`
- [x] 3.3 Реализовать операции с chunks: `UpsertChunks`, `DeleteChunks`, `ListChunksByNode`
- [x] 3.4 Реализовать операции с embeddings: `InsertEmbedding`, `GetAllEmbeddings`, `DeleteEmbedding`
- [x] 3.5 Написать unit-тесты для IndexStore (in-memory SQLite)

## 4. Markdown-aware Chunker

- [x] 4.1 Создать `internal/index/chunker.go` — chunker для разбиения body статей по заголовкам `##`
- [x] 4.2 Реализовать логику: секции > 500 токенов → резать по параграфам; секции < 100 токенов → мержить со следующей
- [x] 4.3 Написать unit-тесты для chunker (покрыть: статью с заголовками, без заголовков, пустой body, очень длинную секцию)

## 5. Векторный поиск

- [x] 5.1 Создать `internal/index/search.go` — функции `VectorSearch` и `ChunkSearch`
- [x] 5.2 Реализовать cosine similarity (linear scan по BLOB-векторам из SQLite)
- [x] 5.3 Написать unit-тесты для векторного поиска (фикстуры с известными векторами)

## 6. SyncWorker — синхронизация индекса

- [x] 6.1 Определить типы событий: `SingleNode(path)`, `GitSyncDiff`, `FullReconcile`, `ManualRebuild`
- [x] 6.2 Создать `internal/index/sync.go` — SyncWorker (runnable) с channel-очередью и rate limiter (1 batch/сек)
- [x] 6.3 Реализовать FullReconcile: walk FS через kb.Store, сверка с indexed_nodes, добавление/обновление/удаление
- [x] 6.4 Реализовать content_hash и body_hash для определения изменений (hash от title+annotation+keywords+type и hash от body)
- [x] 6.5 Реализовать SingleNode: чтение ноды, проверка hash, генерация embedding + chunks при изменении
- [x] 6.6 Реализовать ManualRebuild: очистка индекса + FullReconcile
- [x] 6.7 Написать unit-тесты для SyncWorker

## 7. Интеграция в bootstrap

- [x] 7.1 Обновить `internal/bootstrap/bootstrap.go`: условное создание IndexStore, EmbeddingProvider, SyncWorker при `KB_EMBEDDING_ENABLED=true`
- [x] 7.2 Передать SyncWorker события из API handlers (post-write) и GitSyncRunner (post-sync)
- [x] 7.3 Зарегистрировать SyncWorker в `runnable.Manager`

## 8. API endpoints

- [x] 8.1 Создать `internal/api/chat.go` — ChatHandler: `POST /api/chat` (SSE streaming, контекстная сборка, LLM вызов)
- [x] 8.2 Реализовать контекстную сборку: VectorSearch + ChunkSearch → top-K → форматирование контекста (не более 4000 токенов)
- [x] 8.3 Реализовать SSE streaming ответ: sources event → token events → done event
- [x] 8.4 Добавить `POST /api/index/rebuild` — trigger ManualRebuild, возвращать 202 Accepted
- [x] 8.5 Добавить `GET /api/index/status` — метрики индекса (total_nodes, total_chunks, embedding_model, last_indexed_at, status)
- [x] 8.6 Все новые endpoints возвращают 503 при `KB_EMBEDDING_ENABLED=false`
- [x] 8.7 Написать API-тесты для всех новых endpoints (используя muonsoft/api-testing)

## 9. Frontend — вкладка чатбота

- [x] 9.1 Добавить вкладку «Чат» в Navbar (скрыта по умолчанию, показывается при доступности `GET /api/index/status`)
- [x] 9.2 Создать компонент ChatPage: поле ввода, область ответа, список источников
- [x] 9.3 Реализовать SSE-клиент для streaming-ответа от `POST /api/chat`
- [x] 9.4 Реализовать отображение источников (кликабельные ссылки на ноды)
