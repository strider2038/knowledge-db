## 1. Инфраструктура и конфигурация

- [ ] 1.1 Добавить зависимость `modernc.org/sqlite` (pure-Go SQLite драйвер) в go.mod
- [ ] 1.2 Добавить секцию `Embedding` в `internal/bootstrap/config/config.go`: поля `Enabled`, `APIKey`, `APIURL`, `Model`, `ChatModel` (env vars: `KB_EMBEDDING_ENABLED`, `KB_EMBEDDING_API_KEY`, `KB_EMBEDDING_API_URL`, `KB_EMBEDDING_MODEL`, `KB_CHAT_MODEL`)
- [ ] 1.3 Добавить валидацию: при `KB_EMBEDDING_ENABLED=true` обязательны `KB_EMBEDDING_API_KEY` и `KB_EMBEDDING_API_URL`

## 2. EmbeddingProvider — интерфейс и API-реализация

- [ ] 2.1 Создать пакет `internal/index/`, определить интерфейс `EmbeddingProvider` с методом `Embed(ctx context.Context, texts []string) ([][]float32, error)`
- [ ] 2.2 Реализовать `APIProvider` — OpenAI-совместимый клиент для генерации эмбеддингов (HTTP POST к `/v1/embeddings`, batch-запросы)
- [ ] 2.3 Написать unit-тесты для `APIProvider` (mock HTTP server)

## 3. SQLite IndexStore — схема и базовые операции

- [ ] 3.1 Создать `internal/index/store.go` — `IndexStore` с подключением к SQLite (WAL mode), миграцией схемы (indexed_nodes, chunks, embeddings)
- [ ] 3.2 Реализовать CRUD операции: `UpsertNode`, `DeleteNode`, `GetNodeByPath`, `ListAllIndexed`
- [ ] 3.3 Реализовать операции с chunks: `UpsertChunks`, `DeleteChunks`, `ListChunksByNode`
- [ ] 3.4 Реализовать операции с embeddings: `InsertEmbedding`, `GetAllEmbeddings`, `DeleteEmbedding`
- [ ] 3.5 Написать unit-тесты для IndexStore (in-memory SQLite)

## 4. Markdown-aware Chunker

- [ ] 4.1 Создать `internal/index/chunker.go` — chunker для разбиения body статей по заголовкам `##`
- [ ] 4.2 Реализовать логику: секции > 500 токенов → резать по параграфам; секции < 100 токенов → мержить со следующей
- [ ] 4.3 Написать unit-тесты для chunker (покрыть: статью с заголовками, без заголовков, пустой body, очень длинную секцию)

## 5. Векторный поиск

- [ ] 5.1 Создать `internal/index/search.go` — функции `VectorSearch` и `ChunkSearch`
- [ ] 5.2 Реализовать cosine similarity (linear scan по BLOB-векторам из SQLite)
- [ ] 5.3 Написать unit-тесты для векторного поиска (фикстуры с известными векторами)

## 6. SyncWorker — синхронизация индекса

- [ ] 6.1 Определить типы событий: `SingleNode(path)`, `GitSyncDiff`, `FullReconcile`, `ManualRebuild`
- [ ] 6.2 Создать `internal/index/sync.go` — SyncWorker (runnable) с channel-очередью и rate limiter (1 batch/сек)
- [ ] 6.3 Реализовать FullReconcile: walk FS через kb.Store, сверка с indexed_nodes, добавление/обновление/удаление
- [ ] 6.4 Реализовать content_hash и body_hash для определения изменений (hash от title+annotation+keywords+type и hash от body)
- [ ] 6.5 Реализовать SingleNode: чтение ноды, проверка hash, генерация embedding + chunks при изменении
- [ ] 6.6 Реализовать ManualRebuild: очистка индекса + FullReconcile
- [ ] 6.7 Написать unit-тесты для SyncWorker

## 7. Интеграция в bootstrap

- [ ] 7.1 Обновить `internal/bootstrap/bootstrap.go`: условное создание IndexStore, EmbeddingProvider, SyncWorker при `KB_EMBEDDING_ENABLED=true`
- [ ] 7.2 Передать SyncWorker события из API handlers (post-write) и GitSyncRunner (post-sync)
- [ ] 7.3 Зарегистрировать SyncWorker в `runnable.Manager`

## 8. API endpoints

- [ ] 8.1 Создать `internal/api/chat.go` — ChatHandler: `POST /api/chat` (SSE streaming, контекстная сборка, LLM вызов)
- [ ] 8.2 Реализовать контекстную сборку: VectorSearch + ChunkSearch → top-K → форматирование контекста (не более 4000 токенов)
- [ ] 8.3 Реализовать SSE streaming ответ: sources event → token events → done event
- [ ] 8.4 Добавить `POST /api/index/rebuild` — trigger ManualRebuild, возвращать 202 Accepted
- [ ] 8.5 Добавить `GET /api/index/status` — метрики индекса (total_nodes, total_chunks, embedding_model, last_indexed_at, status)
- [ ] 8.6 Все новые endpoints возвращают 503 при `KB_EMBEDDING_ENABLED=false`
- [ ] 8.7 Написать API-тесты для всех новых endpoints (используя muonsoft/api-testing)

## 9. Frontend — вкладка чатбота

- [ ] 9.1 Добавить вкладку «Чат» в Navbar (скрыта по умолчанию, показывается при доступности `GET /api/index/status`)
- [ ] 9.2 Создать компонент ChatPage: поле ввода, область ответа, список источников
- [ ] 9.3 Реализовать SSE-клиент для streaming-ответа от `POST /api/chat`
- [ ] 9.4 Реализовать отображение источников (кликабельные ссылки на ноды)
