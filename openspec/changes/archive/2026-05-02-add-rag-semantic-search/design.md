## Context

knowledge-db — система управления персональной базой знаний (~100 нод) с offline-first и git-first принципами. Данные хранятся в markdown-файлах с YAML frontmatter в git-репозитории. Сервер (kb-server) предоставляет REST API, Telegram bot и (в будущем) MCP endpoint.

Текущий поиск — `strings.Contains` по title/annotation/keywords с полным walk файловой системы. Векторный поиск, чатбот, семантический поиск отсутствуют. Endpoint `/api/search` — заглушка.

Подход: **Progressive SQLite** — SQLite вводится поэтапно, начиная с минимума для RAG (Фаза 1). kb.Store остаётся без изменений.

## Goals / Non-Goals

**Goals:**

- Добавить опциональный SQLite-индекс для хранения embeddings и chunks
- Реализовать двухуровневый семантический поиск (node-level + chunk-level)
- Добавить RAG-чатбот с endpoint `POST /api/chat`
- Реализовать markdown-aware chunking для длинных статей
- Обеспечить синхронизацию индекса с git-репозиторием
- Сделать всю функциональность полностью отключаемой (по умолчанию выключена)
- Подготовить абстракции для будущей замены API-эмбеддингов на локальные модели

**Non-Goals:**

- Замена kb.Store на SQLite для CRUD-операций (Фазы 2-3)
- Реализация MCP server (отдельная задача)
- Реализация FTS5 полнотекстового поиска (Фаза 2)
- Локальные модели эмбеддингов (future, но интерфейс готовится)
- Поддержка нескольких моделей эмбеддингов одновременно
- Переиндексация в реальном времени через fsnotify

## Decisions

### D1: SQLite как единый движок индексации

**Решение:** Использовать SQLite (один файл) для хранения indexed_nodes, chunks и embeddings.

**Альтернативы:**

- In-memory индекс + Qdrant/Chroma — внешний процесс, нарушает offline-first, два движка синхронизации
- In-memory индекс + SQLite — два движка, избыточно при 100 нодах
- Чистый in-memory (без БД) — потеря индекса при рестарте, нет персистентности

**Обоснование:** Один движок, одна синхронизация, embedded, offline-first. При 100 нодах overhead минимален. SQLite обеспечивает путь эволюции (FTS5, sqlite-vec).

**Расположение файла:** `{data_path}/.kb/index.db` (входит в .gitignore).

### D2: CGO-free SQLite драйвер

**Решение:** Использовать pure-Go SQLite драйвер (например, `modernc.org/sqlite`) для избежания CGO-зависимости.

**Альтернативы:**

- `github.com/mattn/go-sqlite3` — требует CGO, усложняет сборку
- `github.com/glebarez/sqlite` — wrapper над modernc

**Обоснование:** CGO-free сборка упрощает cross-compilation и деплой. При 100 нодах производительность modernc/sqlite достаточна.

### D3: Cosine similarity в Go-коде (без sqlite-vec)

**Решение:** Хранить embeddings как BLOB, вычислять cosine similarity в Go-коде. При 100 нодах linear scan по ~100 векторам — мгновенный.

**Альтернативы:**

- sqlite-vec — CGO-зависимость, избыточно при 100 нодах
- chromem-go — отдельная библиотека, не нужна при наличии SQLite

**Обоснование:** Простота. При росте до 10K+ нод — подключить sqlite-vec без изменения API.

### D4: Markdown-aware chunking по заголовкам

**Решение:** Разбивать body статей по заголовкам `##`. Секции > 500 токенов резать по параграфам, секции < 100 токенов мержить со следующей.

**Альтернативы:**

- Фиксированный размер (500 токенов) — режет посреди мысли/кода
- По параграфам — игнорирует логическую структуру

**Обоснование:** 67% статей имеют `##` заголовки — естественные границы. Для ссылок и заметок chunking не нужен.

### D5: Стратегия embeddings по типам нод

**Решение:**


| Тип       | Node embedding                       | Chunks             |
| --------- | ------------------------------------ | ------------------ |
| `link`    | title + annotation + keywords        | нет                |
| `note`    | title + annotation + keywords + body | нет                |
| `article` | title + annotation + keywords        | body по секциям ## |


**Обоснование:** 43% базы — ссылки (body пустой), 29% — короткие заметки (median 142 слова). Для 72% нод chunking не нужен.

### D6: SyncWorker как runnable

**Решение:** SyncWorker реализует интерфейс `runnable.Runnable`, запускается через `runnable.Manager`. Принимает события через channel, выполняет rate-limited embedding generation.

**Триггеры:** post-write, git sync, manual rebuild, periodic (раз в сутки).

**Обоснование:** Следует существующему паттерну проекта (GitSyncRunner, TranslationWorker).

### D7: Feature toggle через KB_EMBEDDING_ENABLED

**Решение:** По умолчанию `false`. При `false` — SQLite не создаётся, SyncWorker не запускается, `/api/chat` → 503, в UI вкладка скрыта.

**Обоснование:** Offline-first: локально embeddings не нужны. На VPS — включается через env var.

### D8: content_hash для определения изменений

**Решение:**

- `content_hash` = hash(title + annotation + keywords + type) → пересчёт node embedding
- `body_hash` = hash(body markdown) → пересчёт chunks

Поля `created`, `updated`, `source_url`, `manual_processed`, `aliases`, `translations`, `source_author`, `source_date` не включаются — они не влияют на семантику.

## Risks / Trade-offs

**[Rate limiting API] →** Embedding API (OpenRouter) имеет rate limits. SyncWorker ограничивает 1 запрос/сек. При batch-генерации (full rebuild) — добавить backoff и retry.

**[CGO-free SQLite performance] →** modernc/sqlite медленнее mattn/go-sqlite3 на writes. При 100 нодах — незаметно. Мониторить; при проблемах — миграция на mattn.

**[Stale index] →** Индекс может отставать от FS при ошибках. Periodic reconciliation + manual rebuild endpoint компенсируют. Warn-логирование при обнаружении расхождений.

**[Embedding model change] →** При смене модели нужен full rebuild (dimensions могут отличаться). В таблице embeddings хранится model name; при несовпадении с конфигом — trigger rebuild.

**[Крупные статьи] →** Статья до 97KB (6803 слова) создаёт много chunks. При batch embedding — учитывать rate limits. Chunk size ~500 токенов ограничивает количество.