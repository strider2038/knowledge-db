## Context

В проекте уже есть первая версия RAG: SQLite-индекс хранит embeddings и chunks, SyncWorker синхронизирует индекс с markdown-базой, `POST /api/chat` строит контекст из `VectorSearch` и `ChunkSearch`, а Web UI содержит минимальную страницу чата. Обычный поиск в обзоре остаётся filesystem-based и ищет подстроку только в `title`, `annotation`, `keywords`; `/api/search` пока заглушка.

Проблема: векторная близость хорошо ловит смысл, но может пропускать точные термины, названия библиотек, аббревиатуры, пути, aliases и ключевики. Чат полезен для ответа на вопрос, но плох как единственный интерфейс исследования базы: пользователь часто хочет увидеть несколько материалов, сравнить карточки и сам открыть источник.

Ограничения остаются прежними: markdown/git — source of truth; индекс опционален и перестраиваем; проект должен работать offline-first, а LLM/embeddings включаются отдельно.

## Goals / Non-Goals

**Goals:**

- Ввести общий гибридный retrieval pipeline для поиска и RAG-чата.
- Объединять keyword/FTS совпадения и vector results в единый ранжированный список.
- Сохранить быстрый и понятный search UI без обязательной LLM-генерации.
- Улучшить чат так, чтобы он отвечал только при достаточном релевантном контексте и явно показывал источники.
- Добавить отдельную страницу “Поиск” с карточками, фильтрами, snippets и переходом в чат по найденному контексту.
- Сохранить текущую страницу “Обзор” как browsing/table UI для дерева, фильтров и ручной навигации.

**Non-Goals:**

- Полная замена `kb.Store` на SQLite для всех read operations.
- Подключение внешней векторной БД или обязательного sqlite-vec.
- Обязательный LLM rerank для каждого поиска.
- Полноценная многоходовая память чата.
- Изменение формата markdown-ноды как источника правды.
- MCP-интеграция нового retrieval pipeline.

## Decisions

### D1: Общий RetrievalService для поиска и чата

**Решение:** создать backend-слой, который принимает query + фильтры + настройки topK и возвращает unified ranked results. `POST /api/search` и `POST /api/chat` используют этот слой.

```
query
  │
  ├─► keyword/FTS candidates
  ├─► vector node candidates
  └─► vector chunk candidates
          │
          ▼
   normalize + fusion + cutoff
          │
          ▼
   []HybridSearchResult
```

**Почему:** иначе поиск и чат быстро разойдутся: UI будет показывать одни материалы, а чат отвечать по другим. Один retrieval pipeline проще тестировать и настраивать.

**Альтернативы:** оставить чат на `VectorSearch + ChunkSearch`, а поиск сделать отдельно. Это быстрее, но создаёт два разных “понятия релевантности”.

### D2: Два UI-режима, один backend retrieval

**Решение:** добавить отдельную вкладку “Поиск” и сохранить отдельную вкладку “Чат”.

- “Поиск” отвечает на вопрос “что есть в базе?”.
- “Чат” отвечает на вопрос “какой ответ следует из базы?”.

**Почему:** это разные пользовательские задачи. В поиске важны карточки, фильтры и ручное сравнение; в чате важны синтез, цитирование и отказ при недостатке контекста.

**Альтернативы:** объединить всё в один чат с карточками результатов. Это проще в навигации, но скрывает поисковую механику и делает исследование базы менее управляемым.

### D3: SQLite FTS5 или fallback keyword scan

**Решение:** расширить индекс searchable text для нод и чанков. Предпочтительный путь — FTS5 virtual tables, если драйвер/сборка поддерживают FTS5. Если FTS5 недоступен, использовать SQLite/Go fallback scan по нормализованным строкам.

Индексируемые поля:

- node: `path`, `title`, `aliases`, `annotation`, `keywords`, `type`, `source_url`.
- note body: включается в node searchable text.
- article chunks: `heading`, `content`, `node_path`.

**Почему:** exact/keyword поиск должен работать даже когда embeddings отключены или нерелевантны. Fallback сохраняет portability pure-Go SQLite.

**Альтернативы:** сразу использовать только Go filesystem scan. Это проще, но не создаёт единый retrieval слой и плохо сочетается со snippets/chunks.

### D4: Fusion через Reciprocal Rank Fusion

**Решение:** объединять ранги keyword и vector кандидатов через RRF:

```
score = Σ weight(source) / (k + rank(source))
```

Начальные веса:

- exact title/path/keyword/alias match: высокий boost.
- FTS/node keyword result: средний boost.
- chunk FTS result: средний boost.
- vector node/chunk result: базовый semantic boost.

**Почему:** RRF устойчив к разным шкалам score: cosine similarity и FTS rank нельзя честно складывать без калибровки. Для небольшой базы RRF достаточно прозрачен и прост.

**Альтернативы:** weighted sum normalized scores. Это гибко, но потребует калибровки и может вести себя нестабильно на маленькой базе.

### D5: Result model с причинами совпадения и фрагментами

**Решение:** единый результат поиска должен включать:

- `path`, `title`, `type`, `annotation`, `keywords`, `source_url`.
- `score`, `rank`, `match_reasons`.
- `fragments[]` для article/chunk hits: heading, snippet, score, match type.
- `source_kinds`: например `keyword`, `vector_node`, `vector_chunk`, `exact`.

**Почему:** карточки поиска и чатовые источники должны объяснять, почему результат найден. Это особенно важно для личной базы: доверие к поиску часто важнее “умной” выдачи.

### D6: Relevance cutoff для чата

**Решение:** чат использует только результаты выше минимального порога или с достаточными match reasons. Если после cutoff нет результатов, backend отправляет пустые sources и LLM получает пустой контекст либо сервер возвращает заранее заданное сообщение через stream.

**Почему:** top-K vector search почти всегда что-то вернёт, даже если вопрос не относится к базе. Для RAG это опаснее, чем пустая выдача.

**Альтернативы:** доверять LLM, чтобы она сама отказалась. Это слабее: если контекст нерелевантен, модель может всё равно синтезировать ответ.

### D7: Search API как JSON, Chat API как SSE

**Решение:** гибридный поиск реализуется обычным JSON endpoint (`POST /api/search`), чат остаётся streaming SSE (`POST /api/chat`).

**Почему:** поиск должен быть быстрым, кешируемым на уровне UI state и удобным для карточек. Чат генерирует токены и уже использует SSE.

**Альтернативы:** SSE для поиска с progressive results. Сейчас избыточно для базы ~100-10K нод.

### D8: UI-связка “искать → спросить”

**Решение:** карточки поиска могут отправить пользователя в чат с тем же query и выбранными source paths. Чат должен уметь принять initial query/source hints из route state или query params.

**Почему:** это соединяет исследовательский и ответный режимы без смешивания интерфейсов.

## Risks / Trade-offs

**[Risk] FTS5 может быть недоступен в pure-Go SQLite окружении.** → Сделать capability detection при миграции и fallback keyword scan; статус индекса может сообщать `keyword_index: "fts5"` или `"scan"`.

**[Risk] Ранжирование будет требовать настройки на реальной базе.** → Начать с RRF и match reasons, добавить тестовые фикстуры; коэффициенты держать в коде как маленький набор констант.

**[Risk] Чат может потерять полезный контекст из-за слишком строгого cutoff.** → Использовать разные thresholds для exact/keyword и vector-only результатов; показывать “найдено мало данных” вместо полного молчания.

**[Risk] Расширение индекса усложнит миграции.** → Делать additive schema migration; индекс остаётся rebuildable, rollback — удалить `.kb/index.db` и перестроить.

**[Risk] Search UI может дублировать Overview.** → Явно разделить назначения: Overview — дерево/таблица/администрирование; Search — relevance cards/snippets/semantic exploration.

## Migration Plan

1. Добавить новую схему/таблицы searchable text/FTS как additive migration.
2. Обновить SyncWorker, чтобы при индексации ноды записывать searchable text и chunk searchable text.
3. Реализовать RetrievalService и покрыть unit-тестами fusion/ranking/cutoff.
4. Перевести `POST /api/chat` на RetrievalService.
5. Реализовать `POST /api/search` и API-тесты.
6. Добавить Search UI и обновить Chat UI.
7. Проверить ручной rebuild: удаление `.kb/index.db` или `POST /api/index/rebuild` должны полностью восстановить индекс.

Rollback: отключить `KB_EMBEDDING_ENABLED` или удалить `.kb/index.db`; markdown-база не меняется.
