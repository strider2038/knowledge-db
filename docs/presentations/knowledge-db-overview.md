---
title: Knowledge DB
subtitle: Персональная база знаний с AI-агентами
author: Igor Lazarev
theme: beige
revealjs-url: https://cdn.jsdelivr.net/npm/reveal.js@5/
mermaid-theme: default
---

# Knowledge DB

## Персональная база знаний с AI-агентами

<br/>

**Принципы:** оффлайн-first · git-first · LLM-optional

<br/>

<div style="font-size: 0.7em; color: #666;">
Igor Lazarev · 2026
</div>

---

## Концепция проекта

<table>
<tr>
<td style="vertical-align:top; width:50%;">

**Запись — онлайн**

- Web UI (React SPA)
- Telegram бот
- REST API
- MCP (в разработке)

</td>
<td style="vertical-align:top; width:50%;">

**Чтение — оффлайн**

- Локальная база в git
- Markdown + YAML frontmatter
- Версионирование, diff, merge
- Доступна без интернета

</td>
</tr>
</table>

<br/>

> Ничего не потеряется — надёжная версионируемая база под вашим контролем

---

## Стек технологий

```mermaid
graph LR
    subgraph Frontend
        A[React + Vite + TypeScript]
        B[Mermaid · Syntax Highlighting]
    end
    subgraph Backend
        C[Go · stdlib net/http]
        D[SQLite — индекс + чат]
        E[Git CLI — версионирование]
    end
    subgraph AI/ML
        F[OpenAI-compatible API]
        G[Ollama · LM Studio]
        H[Jina Reader API]
    end
    A --> C
    C --> F
    C --> G
    C --> H
    C --> D
    C --> E
```

---

## Архитектура системы

```mermaid
graph TB
    subgraph Клиенты
        WEB[Web UI<br/>React SPA]
        TG[Telegram Bot<br/>Long Polling]
        API_CL[REST API<br/>Клиенты]
        MCP[MCP Server<br/>stub]
    end

    subgraph kb-server — Go
        direction TB
        API_LAYER[API Layer<br/>HTTP Handlers + Auth]
        INGEST[Ingestion Pipeline<br/>LLM Orchestrator]
        IDX[Index System<br/>Embeddings + FTS5]
        CHAT[RAG Chat<br/>Hybrid Search + LLM]
        KB_STORE[KB Store<br/>Filesystem + Git]
        GIT[Git Sync<br/>Auto Commit + Push]
    end

    subgraph Хранилище
        FS[data/<br/>Markdown + Frontmatter]
        SQLITE_IDX[index.db<br/>Embeddings + Chunks + FTS]
        SQLITE_CHAT[chat.db<br/>Sessions + Messages]
    end

    subgraph Внешние API
        LLM[OpenAI-compat API<br/>LLM + Embeddings]
        JINA[Jina Reader<br/>Content Fetcher]
    end

    WEB --> API_LAYER
    TG --> INGEST
    API_CL --> API_LAYER
    API_LAYER --> INGEST
    API_LAYER --> CHAT
    INGEST --> KB_STORE
    INGEST --> LLM
    INGEST --> JINA
    CHAT --> IDX
    CHAT --> LLM
    IDX --> SQLITE_IDX
    CHAT --> SQLITE_CHAT
    KB_STORE --> FS
    KB_STORE --> GIT
    GIT --> FS
    IDX --> LLM
```

---

## Серверная часть: bootstrap

```mermaid
flowchart TD
    MAIN[main.go] --> BOOT[bootstrap.Run]
    BOOT --> CFG[Config.Load<br/>переменные окружения]
    CFG --> ING[buildIngester<br/>Pipeline или Stub]
    CFG --> IDX[buildIndexComponents<br/>если KB_EMBEDDING_ENABLED]
    CFG --> CHAT_STORE[chat.NewStore<br/>SQLite chat.db]
    ING --> HANDLER[api.NewHandler]
    IDX --> HANDLER
    CHAT_STORE --> HANDLER
    HANDLER --> ROUTER[api.NewMux<br/>все маршруты]
    ROUTER --> MGR[runnable.Manager]
    MGR --> HTTP_S[HTTP Server :8080]
    MGR --> TG_BOT[Telegram Bot<br/>если TELEGRAM_TOKEN]
    MGR --> GIT_SYNC[Git Sync Runner<br/>периодический fetch+merge]
    MGR --> TRANS_W[Translation Worker<br/>фоновый перевод статей]
    MGR --> IDX_SYNC[Index Sync Worker<br/>синхронизация эмбеддингов]
```

---

## Где используется LLM

<br/>

<table>
<tr>
<th style="text-align:left;">Компонент</th>
<th style="text-align:left;">Роль LLM</th>
<th style="text-align:left;">API</th>
</tr>
<tr>
<td><b>Ingestion Pipeline</b></td>
<td>Классификация, аннотирование, ключевые слова, выбор темы</td>
<td>OpenAI Responses API + Function Calling</td>
</tr>
<tr>
<td><b>Перевод статей</b></td>
<td>Чанкованный перевод на русский</td>
<td>Chat Completions</td>
</tr>
<tr>
<td><b>RAG Chat</b></td>
<td>Ответы на основе контента базы знаний</td>
<td>Chat Completions SSE</td>
</tr>
<tr>
<td><b>Query Rewrite</b></td>
<td>Оптимизация поискового запроса для RAG</td>
<td>Responses API</td>
</tr>
<tr>
<td><b>Git Commit Messages</b></td>
<td>Генерация conventional commit</td>
<td>Chat Completions</td>
</tr>
</table>

---

## Ingestion Pipeline — обзор

```mermaid
flowchart LR
    INPUT[Ввод:<br/>текст / URL] --> PREP[Подготовка контекста<br/>существующие темы + ключевые слова]
    PREP --> ORCH[LLM Orchestrator<br/>OpenAI Responses API]
    ORCH --> |function calling| FETCH[fetch_url_content<br/>Jina → Readability]
    ORCH --> |function calling| META[fetch_url_meta<br/>HTML meta / GitHub API]
    ORCH --> |tool call| CREATE[create_node<br/>structured result]
    CREATE --> SAVE[Сохранение<br/>Markdown + Frontmatter]
    SAVE --> GIT_C[Git commit + push]
    SAVE --> |опционально| TRANSLATE[LLM Translator<br/>перевод статьи]
```

---

## Ingestion — LLM Orchestrator

```mermaid
sequenceDiagram
    participant P as Pipeline
    participant O as LLM Orchestrator
    participant LLM as OpenAI API
    participant F as Content Fetcher

    P->>O: Process(input, themes, keywords)
    O->>LLM: Responses API + system prompt + tools
    LLM-->>O: tool_call: fetch_url_content(url)
    O->>F: Fetch(url)
    F-->>O: content (кэшируется)
    Note over O,LLM: Контент инжектится напрямую,<br/>LLM получает первые 2000 символов
    O->>LLM: Продолжение диалога с контентом
    LLM-->>O: tool_call: create_node(result)
    O-->>P: ProcessResult{keywords, annotation, theme, content...}

    Note over O,LLM: До 10 итераций<br/>Stateless — полная история в каждом запросе
```

---

## Ingestion — инструменты LLM

<table>
<tr>
<th style="text-align:left;">Tool</th>
<th style="text-align:left;">Назначение</th>
<th style="text-align:left;">Данные</th>
</tr>
<tr>
<td><code>fetch_url_content</code></td>
<td>Получить полный контент по URL</td>
<td>Jina Reader API → fallback Readability</td>
</tr>
<tr>
<td><code>fetch_url_meta</code></td>
<td>Метаданные URL (title, description)</td>
<td>GitHub API → HTML &lt;meta&gt; парсинг</td>
</tr>
<tr>
<td><code>create_node</code></td>
<td>Создать узел БЗ (финальное действие)</td>
<td>keywords, annotation, theme_path, slug, type, content, source_url, title</td>
</tr>
</table>

<br/>

**Системный промпт** задаёт правила:

- Типы контента: `article`, `link`, `note`
- Аннотации и ключевые слова — на русском
- Выбор существующей темы или создание новой
- Для ссылок — конкретика без маркетинговых штампов

---

## RAG Chat — общая схема

```mermaid
flowchart TD
    USER[Пользователь<br/>отправляет сообщение] --> DETECT[detectChatMode<br/>memory / rag / hybrid]
    
    DETECT --> |memory| PURE[Чистый LLM<br/>без контекста БЗ]
    DETECT --> |rag / hybrid| REWRITE[Query Rewrite<br/>LLM оптимизирует запрос]
    
    REWRITE --> RETRIEVE[RetrievalService<br/>Hybrid Search]
    RETRIEVE --> KW[Keyword Search<br/>FTS5 + BM25]
    RETRIEVE --> VEC[Vector Search<br/>Cosine Similarity]
    KW --> RRF[RRF Fusion<br/>Reciprocal Rank Fusion]
    VEC --> RRF
    RRF --> FILTER[Фильтрация<br/>score gap + пороги]
    FILTER --> CTX[Контекст<br/>до 4000 токен]
    
    CTX --> STREAM[LLM Chat Streaming<br/>SSE → фронтенд]
    PURE --> STREAM
    STREAM --> SAVE[Сохранение ответа<br/>в Chat Store]
```

---

## RAG Chat — режимы работы

<br/>

<table>
<tr>
<th style="text-align:left;">Режим</th>
<th style="text-align:left;">Когда</th>
<th style="text-align:left;">Поведение</th>
</tr>
<tr>
<td><code>memory</code></td>
<td>«суммируй чат»</td>
<td>Только LLM, без поиска по БЗ</td>
</tr>
<tr>
<td><code>rag</code></td>
<td>«найди в базе...»</td>
<td>Строгий поиск — только контекст из БЗ</td>
</tr>
<tr>
<td><code>hybrid</code></td>
<td>по умолчанию</td>
<td>Сначала БЗ, затем fallback на знания LLM</td>
</tr>
</table>

<br/>

**Query Rewrite** — LLM переписывает запрос пользователя в компактные поисковые термины с учётом словаря базы знаний (aliases, keywords, titles).

---

## Embedding & Index System

```mermaid
flowchart TD
    subgraph SyncWorker
        FS_SCAN[Сканирование FS<br/>data/*.md файлы] --> HASH[content_hash + body_hash]
        HASH --> |изменился| EMBED_TXT[Текст для эмбеддинга<br/>title + annotation + keywords + content]
        HASH --> |не изменился| SKIP[Пропуск]
        EMBED_TXT --> API_CALL[Embedding API<br/>/v1/embeddings]
        API_CALL --> STORE_EMB[Сохранение в SQLite<br/>vector BLOB]
        STORE_EMB --> NODE_SEARCH[node_search + FTS5]
        STORE_EMB --> |article| CHUNK[Чанкинг по заголовкам ##]
        CHUNK --> CHUNK_EMB[Embedding каждого чанка]
        CHUNK_EMB --> CHUNK_STORE[chunks + chunk_search + FTS5]
    end

    subgraph Triggers
        START[Старт сервера] --> FULL[FullReconcile]
        GIT_PULL[Git Pull] --> DIFF[GitSyncDiff]
        API_CALL_2[POST /api/index/rebuild] --> MANUAL[ManualRebuild]
    end

    FULL --> FS_SCAN
    DIFF --> FS_SCAN
    MANUAL --> FS_SCAN
```

---

## Hybrid Search — RRF Fusion

```mermaid
flowchart LR
    QUERY[Поисковый запрос] --> KW[keyword search<br/>FTS5 BM25]
    QUERY --> VEC_NODE[vector search nodes<br/>cosine similarity]
    QUERY --> VEC_CHUNK[vector search chunks<br/>cosine similarity]
    
    KW --> RRF[Reciprocal Rank Fusion]
    VEC_NODE --> RRF
    VEC_CHUNK --> RRF
    
    RRF --> RESULT[Ранжированные результаты<br/>score = weight/k+rank + rawScore/100]
```

<br/>

**Веса RRF:**

| Источник | Вес | Примечание |
|----------|-----|------------|
| keyword node | 1.8 | BM25 + exact boost |
| keyword chunk | 1.6 | BM25 + FTS5 |
| vector node | 1.0 | Cosine similarity |
| vector chunk | 1.0→0.25 | Убывающий по кол-ву чанков ноды |

---

## Chat Session Memory

```mermaid
flowchart TD
    MSG[Новое сообщение] --> TRIM[SummarizeAndTrim]
    TRIM --> |старые сообщения| SUMMARY[LLM summary<br/>сжатие контекста]
    TRIM --> |последние N| KEEP[Сохраняются]
    SUMMARY --> BUILD[BuildPromptMessages]
    KEEP --> BUILD
    BUILD --> LLM_CALL[LLM Chat Completions<br/>SSE streaming]
    
    LLM_CALL --> SAVE_MSG[Сохранение ответа<br/>в chat_messages]
    
    CLEANUP[CleanupExpired<br/>TTL = 7 дней] --> DEL[Удаление<br/>устаревших сессий]
```

<br/>

**Параметры:** maxMessages = 40, maxContextRunes = 24000, sessionTTL = 7d

---

## Telegram Bot

```mermaid
flowchart TD
    POLL[Long Polling<br/>getUpdates] --> HANDLE[handleUpdate]
    HANDLE --> |обычный текст| COMMENT[Буфер: комментарий<br/>TTL 3 сек]
    HANDLE --> |пересланное сообщение| FORWARD[Буфер: forward<br/>TTL 3 сек]
    HANDLE --> |reply на пересланное| COMBINED[Комбинация<br/>комментарий + forward]
    
    COMMENT --> |таймаут или второе сообщение| INGEST[ingester.IngestText]
    FORWARD --> INGEST
    COMBINED --> INGEST
    INGEST --> LLM_PIPE[LLM Pipeline<br/>классификация + сохранение]
    LLM_PIPE --> REPLY[Ответ в Telegram<br/>ссылка на Web UI]
    
    FORWARD --> |parse forward_origin| ORIGIN[source_url + author<br/>channel / user / hidden]
```

---

## Telegram Bot — Entity Processing

<br/>

Telegram → Markdown конвертация:

- **Bold** → `**bold**`
- *Italic* → `*italic*`
- `Code` → `` `code` ``
- Links → `[text](url)`
- Pre blocks → ` ```code``` `

<br/>

**Forward Origin** — определение источника:

| Тип | Данные |
|-----|--------|
| Channel | source_url из channel, author из подписи |
| User | author из имени пользователя |
| Hidden user | автор не определён |

---

## Git Integration с LLM

```mermaid
flowchart LR
    subgraph Commit
        NODE[Создание/изменение<br/>узла БЗ] --> DIFF[git diff]
        DIFF --> MSG_GEN[CommitMessageGenerator]
        MSG_GEN --> LLM[LLM генерирует<br/>conventional commit]
        LLM --> COMMIT[git commit -m "msg"]
        COMMIT --> PUSH[git push]
    end
    
    subgraph Sync
        INTERVAL[Каждые 5 мин] --> FETCH[git fetch]
        FETCH --> MERGE[git merge]
        MERGE --> RECONCILE[Reconcile индекса<br/>FS vs indexed]
    end
```

<br/>

**Сериализация** — `SerializedGitCommitter` обёрнут в mutex для предотвращения конкурентных git операций.

---

## Перевод статей

```mermaid
flowchart TD
    ARTICLE[Новая статья<br/>type=article] --> CHECK{Нужен<br/>перевод?}
    CHECK --> |да| QUEUE[Translation Queue<br/>in-memory]
    CHECK --> |нет| SKIP_2[Пропуск]
    
    QUEUE --> WORKER[Translation Worker<br/>фоновая горутина]
    WORKER --> CHUNKS[Разбиение на чанки<br/>если > 6000 символов]
    CHUNKS --> LLM_T[LLM Translation<br/>каждый чанк отдельно]
    LLM_T --> MERGE_2[Слияние чанков]
    MERGE_2 --> SAVE_T[Сохранение<br/>slug.ru.md]
    SAVE_T --> META_2[Обновление frontmatter<br/>translations: поле]
```

---

## Auth & Security

```mermaid
flowchart TD
    REQ[HTTP Request] --> MW[Auth Middleware]
    MW --> |/healthz, /api/auth/*| ALLOW[Allowlist — пропускается]
    MW --> |остальные| COOKIE[Проверка kb_session cookie]
    COOKIE --> |валидная| PASS[Доступ разрешён]
    COOKIE --> |отсутствует/невалидная| REDIR[Redirect /login]
    
    subgraph Режимы аутентификации
        MODE_OFF[Open<br/>без авторизации]
        MODE_PWD[Password<br/>KB_LOGIN + KB_PASSWORD]
        MODE_OAUTH[Google OAuth 2.0<br/>+ email allowlist]
    end
```

<br/>

Сессии — in-memory store с TTL (по умолчанию 8h), cookie `Secure` требует HTTPS.

---

## Web UI

```mermaid
flowchart TD
    subgraph React SPA — Vite
        NAV[Navbar<br/>Topic Tree] --> OVERVIEW[OverviewPage<br/>Browse + Filter]
        NAV --> NODE_P[NodePage<br/>View + Edit]
        NAV --> ADD[AddPage<br/>Text / URL Ingest]
        NAV --> SEARCH[SearchPage<br/>Hybrid Search]
        NAV --> CHAT_P[ChatPage<br/>RAG Chat SSE]
        NAV --> LOGIN_P[LoginPage<br/>Password / OAuth]
    end

    OVERVIEW --> API_TS[api.ts<br/>TypeScript клиент]
    NODE_P --> API_TS
    ADD --> API_TS
    SEARCH --> API_TS
    CHAT_P --> API_TS
    LOGIN_P --> API_TS
    API_TS --> REST[REST API :8080]
```

<br/>

**Особенности UI:** Mermaid-диаграммы, подсветка синтаксиса, PWA, мобильная адаптация, SSE-streaming чат.

---

## Model Context Protocol

<br/>

**Текущий статус:** stub — `GET/POST /api/mcp` → 501

<br/>

**Планируемая функциональность:**

- Подключение внешних AI-агентов (Claude, GPT и др.)
- Инструменты для работы с базой знаний
- Стандартизированный протокол взаимодействия

<br/>

Спецификация: `openspec/specs/mcp-server/`

---

## Data Model

```mermaid
erDiagram
    NODE ||--o{ CHUNK : "article has chunks"
    NODE {
        string path PK
        string title
        string type "article|link|note|auto"
        string annotation
        string[] keywords
        string source_url
        string source_author
        string source_date
        string content "Markdown body"
        string[] aliases
        boolean manual_processed
    }
    CHUNK {
        int id PK
        string node_path FK
        int chunk_index
        string heading
        string content
        float[] embedding
    }
    NODE ||--|| NODE_EMBEDDING : "has embedding"
    NODE_EMBEDDING {
        int id PK
        float[] vector "BLOB float32"
        string model
        int dimensions
    }
    NODE_SEARCH {
        string path PK
        string title
        string type
        string annotation
        string keywords
        string body
        string searchable_text "FTS5"
    }
```

---

## Дизайн-решения

<br/>

<table>
<tr>
<td style="vertical-align:top; width:50%;">

**Хранение**

- Markdown + YAML frontmatter
- Git — источник правды
- SQLite — только индексы
- Локальность — без облака

</td>
<td style="vertical-align:top; width:50%;">

**AI — опционально**

- LLM не настроен → `StubIngester`
- Embeddings off → FTS5 keyword search
- Graceful degradation на каждом уровне

</td>
</tr>
<tr>
<td style="vertical-align:top; width:50%;">

**Совместимость**

- OpenAI-compatible API
- Ollama, LM Studio, OpenRouter
- Любая модель эмбеддингов

</td>
<td style="vertical-align:top; width:50%;">

**Синхронизация**

- Git auto-commit с LLM-сообщениями
- Периодический fetch+merge
- FS watcher → re-index

</td>
</tr>
</table>

---

## Ключевые технологии

<br/>

<div style="font-size: 0.9em;">

| Слой | Технология | Назначение |
|------|-----------|------------|
| Backend | Go 1.25 | Сервер, API, Telegram бот |
| Frontend | React + Vite + TypeScript | SPA интерфейс |
| Хранение | Markdown + YAML frontmatter | Узлы базы знаний |
| Индексация | SQLite + FTS5 | Полнотекстовый поиск |
| Векторный поиск | SQLite + float32 BLOB | Cosine similarity |
| LLM | OpenAI Responses API | Ingestion, function calling |
| LLM | Chat Completions API | RAG чат, перевод, коммиты |
| Embeddings | /v1/embeddings | Векторные представления |
| Контент | Jina Reader + Readability | Извлечение контента URL |
| Git | CLI exec | Версионирование базы |
| Контейнеризация | Docker + GitHub Actions | Сборка и деплой |

</div>

---

## Запуск и конфигурация

<br/>

**Минимальный запуск:**

```bash
KB_DATA_PATH=/path/to/data ./kb-server
```

**С AI-функциями:**

```bash
export KB_DATA_PATH=/path/to/data
export LLM_API_URL=https://openrouter.ai/api/v1
export LLM_API_KEY=sk-...
export LLM_MODEL=gpt-4o
export KB_EMBEDDING_ENABLED=true
export KB_EMBEDDING_API_URL=http://localhost:11434
export KB_EMBEDDING_MODEL=bge-m3
export KB_CHAT_MODEL=llama3
./kb-server
```

**Docker:**

```bash
docker run -d -p 8080:8080 \
  -v /path/to/knowledge-base:/data \
  -e KB_DATA_PATH=/data \
  ghcr.io/strider2038/knowledge-db:latest
```

---

## Итоги

<br/>

**Knowledge DB** — персональная система управления знаниями, где AI-компоненты играют ключевую роль:

<br/>

1. **LLM Orchestrator** — интеллектуальная классификация и структурирование контента
2. **RAG Chat** — гибридный поиск + генерация ответов на основе базы знаний
3. **Embedding Index** — векторные представления для семантического поиска
4. **Function Calling** — многошаговые LLM-пайплайны с инструментами
5. **Query Rewrite** — оптимизация поисковых запросов с учётом словаря БЗ
6. **Auto Translation** — фоновый перевод статей через LLM
7. **Smart Commits** — осмысленные git-коммиты через LLM

<br/>

> Все AI-функции опциональны — система работает и без них

---

## Ссылки

<br/>

- **Репозиторий:** https://github.com/strider2038/knowledge-db
- **Лицензия:** MIT © 2026 Igor Lazarev
- **Спецификации:** `openspec/specs/` — 25 детальных спецификаций
- **Agent Skills:** `.cursor/skills/` — навыки для IDE
