## Why

В базе знаний ~100 нод, и единственный способ поиска — линейный `strings.Contains` по title/annotation/keywords с полным walk файловой системы на каждый запрос. Нет возможности задать вопрос о контенте базы на естественном языке. Для VPS-сценария (kb-server + веб) необходимы семантический поиск и чатбот, способный отвечать на вопросы по содержимому базы знаний (RAG).

## What Changes

- Добавляется SQLite-индекс (embeddings + chunks) для семантического поиска по содержимому базы знаний
- Добавляется интерфейс `EmbeddingProvider` с реализацией через OpenAI-совместимое API (OpenRouter)
- Добавляется markdown-aware chunker для разбиения статей на фрагменты по заголовкам `##`
- Добавляется SyncWorker (runnable) для синхронизации индекса с git-репозиторием (по событиям, при git sync, periodic reconciliation)
- Добавляется endpoint `POST /api/chat` для RAG-чатбота
- Добавляется endpoint `POST /api/index/rebuild` для ручного запуска полной перестройки индекса
- Добавляется endpoint `GET /api/index/status` для проверки состояния индекса
- Функциональность полностью опциональна и отключаема через конфиг (`KB_EMBEDDING_ENABLED`)
- kb.Store остаётся без изменений — SQLite используется только для RAG (Фаза 1 progressive-подхода)

## Capabilities

### New Capabilities

- `embedding-index`: SQLite-индекс с embeddings и chunks, синхронизация с git-репозиторием, markdown-aware chunking, интерфейс EmbeddingProvider
- `rag-chat`: RAG-чатбот — endpoint `/api/chat`, двухуровневый векторный поиск, контекстная сборка для LLM, streaming-ответ

### Modified Capabilities

- `rest-api`: добавляются новые endpoints (`/api/chat`, `/api/index/rebuild`, `/api/index/status`)
- `webapp`: добавляется вкладка чатбота в UI (скрыта при `KB_EMBEDDING_ENABLED=false`)

## Impact

- **Новое**: пакет `internal/index/` (IndexStore, EmbeddingProvider, SyncWorker, Chunker)
- **Новое**: `internal/api/chat.go` (ChatHandler)
- **Изменение**: `internal/bootstrap/config/` — новые env vars (`KB_EMBEDDING_`*)
- **Изменение**: `internal/bootstrap/bootstrap.go` — условное создание компонентов
- **Изменение**: `internal/api/` — новые routes
- **Изменение**: `web/` — вкладка чатбота
- **Зависимость**: `github.com/mattn/go-sqlite3` (CGO) или pure-Go SQLite драйвер
- **Зависимость**: OpenAI-совместимое API для embeddings (при включённой функциональности)