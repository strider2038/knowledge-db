# Исследование: Семантический поиск и RAG для knowledge-db

Дата: 2026-04-26

## Контекст

knowledge-db — система управления персональной базой знаний с offline-first и git-first принципами. База хранится локально в markdown-файлах с YAML frontmatter.

Текущее состояние:

- 86 нод в базе, рост до 10K не ожидается в ближайшие годы
- Каждый API-запрос выполняет filesystem walk (kb.Store через afero)
- Поиск — `strings.Contains` по title + annotation + keywords
- `/api/search` — заглушка
- БД, кэш, векторный индекс — отсутствуют

Цели:

1. Добавить семантический поиск по базе знаний
2. Добавить вкладку с чатботом для вопросов о контенте базы (RAG)
3. Не усложнять проект избыточными движками индексации

## Рассмотренные варианты архитектуры

### Вариант A: In-Memory индекс + отдельная векторная БД

Два движка индексации с разными моделями синхронизации.

```
RAM Index (metadata, keywords, tree)
       +
Qdrant / Chroma / Weaviate (vectors)
```

Плюсы:

- RAM-индекс быстрый для CRUD/фильтрации
- Специализированные векторные БД — зрелые решения

Минусы:

- Два pipeline синхронизации (RAM ↔ git, DB ↔ git)
- Source of truth размыт: git — данные, RAM — API, БД — поиск
- Qdrant/Chroma — внешний процесс, нарушает offline-first
- Сложность поддержки двух движков

### Вариант B: In-Memory индекс + SQLite для RAG

RAM для CRUD, SQLite только для векторного поиска.

```
RAM Index (metadata, keywords, tree)
       +
SQLite (embeddings, chunks)
```

Плюсы:

- SQLite embedded, без внешних сервисов
- RAM-индекс для быстрых reads

Минусы:

- Два движка с разными моделями синхронизации
- RAM-индекс теряется при restart (нужна реконструкция)
- Избыточно при 86 нодах

### Вариант C: SQLite как единый движок (выбранный)

Единая БД для всех задач: метаданные, полнотекстовый поиск, векторный поиск.

```
SQLite (metadata + FTS5 + vectors)
```

Плюсы:

- Один движок, одна синхронизация
- Embedded, один файл, offline-first
- FTS5 — встроенный полнотекстовый поиск
- sqlite-vec — расширение для векторного поиска
- Естественная эволюция: от RAG-индекса к полной замене FS reads

Минусы:

- Требует Care при concurrent writes (WAL mode)
- sqlite-vec требует CGO (альтернатива — cosine similarity в Go-коде)

### Сравнение


| Критерий        | RAM + Qdrant | RAM + SQLite | SQLite only |
| --------------- | ------------ | ------------ | ----------- |
| Offline-first   | нет          | да           | да          |
| Один движок     | нет (2)      | нет (2)      | да (1)      |
| Полнотекстовый  | нужен доп.   | в SQLite     | FTS5        |
| Векторный поиск | Qdrant       | sqlite-vec   | sqlite-vec  |
| Синхронизация   | 2 pipeline   | 2 pipeline   | 1 pipeline  |
| Сложность       | высокая      | средняя      | низкая      |
| Deps            | Qdrant bin   | go-sqlite3   | go-sqlite3  |


## Выбранный подход: Progressive SQLite (эволюционный)

SQLite вводится поэтапно, начиная с минимума для RAG. kb.Store остаётся без изменений для CRUD и reads.

### Фаза 1: SQLite только для RAG

- kb.Store — как сейчас (FS-based CRUD, дерево, фильтрация)
- SQLite — индекс для семантического поиска и чатбота
- Не затрагивает существующий код

```
git/filesystem  ◄── source of truth
      │
      │
   ┌──┴───────────────┐
   │                  │
   ▼                  ▼
kb.Store         SQLite (RAG only)
(как сейчас)     embeddings + chunks
   │              │
   ▼              ▼
CRUD, дерево    Векторный поиск,
фильтрация      чатбот, QA
```

### Фаза 2: SQLite расширяется (когда понадобится)

- Добавляется FTS5 для полнотекстового поиска
- `/api/search` начинает работать через SQLite
- kb.Store делегирует поиск в SQLite

### Фаза 3: SQLite заменяет kb.Store для reads (если база вырастет)

- Все reads идут через SQLite
- Write path: kb.Store → FS → sync → SQLite

## Анализ реальной базы знаний

### Статистика (~/projects/my-knowledge-base)

86 нод, 3 типа:


| Тип       | Кол-во | Доля | Body (слов) | Описание                   |
| --------- | ------ | ---- | ----------- | -------------------------- |
| `link`    | 37     | 43%  | 0           | Закладка, annotation + URL |
| `note`    | 25     | 29%  | ~142 (med)  | Рецепт, заметка, код       |
| `article` | 24     | 28%  | ~2493 (med) | Полная статья              |


Распределение размеров:

```
  <1KB:  23 шт (27%)  — link-ноды, только frontmatter
  1-5KB: 17 шт (20%)  — короткие notes
  >20KB: 19 шт (22%)  — articles
```

Frontmatter: 7 полей в 100% файлов (title, type, aliases, annotation, keywords, created, updated). Annotation — всегда качественное резюме на русском, 30-60 слов.

### Ключевой инсайт

**72% нод (links + notes) не требуют chunking.** Достаточно одного embedding на ноду. Chunking нужен только для `type=article` (24 ноды).

## Выводы и решения

### 1. Архитектура: Progressive SQLite

SQLite — единый движок, вводится поэтапно. Фаза 1 — только для RAG, без изменений kb.Store.

Расположение файла: `{data_path}/.kb/index.db`.

### 2. Chunking: по типам нод


| Тип       | Стратегия                                                               |
| --------- | ----------------------------------------------------------------------- |
| `link`    | Один embedding: title + annotation + keywords. Chunks не нужны.         |
| `note`    | Один embedding: title + annotation + keywords + body целиком.           |
| `article` | Node embedding: title + annotation + keywords. Chunks: по секциям `##`. |


Правила chunking для articles:

- Каждый `##` заголовок — граница чанка
- Секция > ~500 токенов — резать по параграфам
- Секция < ~100 токенов — мержить со следующей

### 3. Двухуровневый поиск

Запрос пользователя → embedding запроса:

- **Level 1 (node embeddings):** какие ноды релевантны → top-K
- **Level 2 (chunk embeddings):** какие фрагменты статей релевантны → top-K

Результат: комбинация annotations (links/notes) + фрагменты (articles) → контекст для LLM.

### 4. Схема SQLite

```sql
CREATE TABLE indexed_nodes (
    path             TEXT PRIMARY KEY,
    content_hash     TEXT NOT NULL,
    body_hash        TEXT,
    indexed_at       DATETIME NOT NULL,
    node_embedding_id INTEGER REFERENCES embeddings(id)
);

CREATE TABLE chunks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    node_path    TEXT NOT NULL REFERENCES indexed_nodes(path),
    chunk_index  INTEGER NOT NULL,
    heading      TEXT,
    content      TEXT NOT NULL,
    embedding_id INTEGER REFERENCES embeddings(id),
    UNIQUE(node_path, chunk_index)
);

CREATE TABLE embeddings (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    vector     BLOB NOT NULL,
    model      TEXT NOT NULL,
    dimensions INTEGER NOT NULL DEFAULT 1536
);

CREATE INDEX idx_chunks_node ON chunks(node_path);
```

### 5. content_hash

**content_hash** = hash(title + annotation + keywords + type)

- Пересчёт node embedding при изменении

**body_hash** = hash(body markdown)

- Пересчёт chunks при изменении

Поля, НЕ включаемые в hash: created, updated, source_url, manual_processed, aliases, translations, source_author, source_date.

### 6. Триггеры обновления индекса

1. **Post-write:** после CreateNode / PatchNode — проверить hash, enqueue embedding job
2. **Git sync:** GitSyncRunner отработал → git diff → enqueue changed paths
3. **Manual:** `POST /api/index/rebuild` — полная перестройка
4. **Periodic:** reconciliation loop (редко, раз в сутки)

### 7. SyncWorker (runnable)

- Очередь задач: FullReconcile, SingleNode(path), GitSyncDiff
- Rate limiting: 1 embedding request/sec (API limits)
- FullReconcile: walk FS → check indexed_nodes → add/update/delete
- Логирование warn при fallback (SQLite недоступен)

### 8. EmbeddingProvider (интерфейс)

```go
type EmbeddingProvider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}
```

Реализации:

- `APIProvider` — OpenAI / OpenRouter (Фаза 1)
- `LocalProvider` — локальная модель (future)

Векторный поиск при ~100 нодах: cosine similarity в Go-коде, без sqlite-vec. При росте — подключить sqlite-vec.

### 9. Отключаемость

Embeddings опциональны, работают только на сервере.

```env
KB_EMBEDDING_ENABLED=false   # default, embeddings disabled
KB_EMBEDDING_ENABLED=true    # включает SQLite + SyncWorker + /api/chat
KB_EMBEDDING_API_KEY=...
KB_EMBEDDING_API_URL=https://openrouter.ai/api/v1/...
KB_EMBEDDING_MODEL=text-embedding-3-small
```

При `KB_EMBEDDING_ENABLED=false`:

- SQLite не создаётся
- SyncWorker не запускается
- `/api/chat` → 503
- В UI вкладка чата скрыта или disabled

### 10. Компоненты Фазы 1


| Компонент         | Путь                          | Описание                              |
| ----------------- | ----------------------------- | ------------------------------------- |
| IndexStore        | `internal/index/store.go`     | SQLite: schema, CRUD, vector search   |
| EmbeddingProvider | `internal/index/embedding.go` | Интерфейс + APIProvider               |
| SyncWorker        | `internal/index/sync.go`      | Runnable: events + periodic reconcile |
| Chunker           | `internal/index/chunker.go`   | Markdown-aware chunking по ##         |
| ChatHandler       | `internal/api/chat.go`        | `POST /api/chat`                      |
| Config            | `internal/bootstrap/config/`  | `KB_EMBEDDING_*` env vars             |
| Bootstrap wiring  | `internal/bootstrap/`         | Условное создание компонентов         |


