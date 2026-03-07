## Context

Ingestion pipeline существует как интерфейс `Ingester` с двумя методами (`IngestText`, `IngestURL`) и заглушкой `StubIngester`, которая возвращает `ErrNotImplemented`. Telegram-бот вызывает `IngestText` при получении текста, API — `POST /api/ingest`. Оба получают 501.

Store (`internal/kb`) — read-only: `GetNode`, `ListNodes`, `ReadTree`, `Validate`. Методов записи нет.

Frontmatter содержит `keywords`, `created`, `updated`, `annotation`. Нет полей для типа контента и источника.

Зависимости собираются в `internal/bootstrap/bootstrap.go`: `StubIngester` создаётся без параметров и передаётся в `api.Handler` и `telegram.Bot`.

## Goals / Non-Goals

**Goals:**

- Реализовать полноценный ingestion pipeline: текст и URL → узел в базе
- Парсинг URL с извлечением контента для оффлайн-хранения
- LLM-генерация метаданных (keywords, annotation, theme/subtheme, type)
- Расширить frontmatter полями `type`, `source_url`, `source_date`
- Расширить Store возможностью создания узлов
- Telegram-бот: передача текста в pipeline, подтверждение пользователю
- Git commit при добавлении записи + периодический sync с remote

**Non-Goals:**

- Интерактивный цикл правок через Telegram (отдельная задача)
- Рекаталогизация и ревизия metadata (отдельная задача)
- Сохранение картинок из статей (вопрос открыт)
- Web UI для добавления контента (API endpoint достаточно)
- Векторные эмбеддинги и RAG-поиск (ортогональная задача)

## Decisions

### 1. Архитектура Ingester: LLM как оркестратор

**Решение**: Заменить `StubIngester` на `PipelineIngester`, где LLM выступает оркестратором через function calling. LLM анализирует входной текст и решает, какие инструменты вызвать.

```
PipelineIngester
├── LLMOrchestrator (function calling — решает что делать)
│   ├── tool: fetch_url_content  → ContentFetcher
│   ├── tool: fetch_url_meta     → частичное извлечение (title, description)
│   └── tool: create_node        → формирует метаданные + контент
├── ContentFetcher  (извлечение полного контента из URL)
├── kb.Store        (запись узла в ФС)
├── GitCommitter    (git add + commit + периодический fetch)
└── config          (basePath, и т.п.)
```

**Сценарии, определяемые LLM:**

1. **Текст содержит URL на статью** → LLM вызывает `fetch_url_content` → получает полный контент → формирует node с `type: article`
2. **Текст без URL** → LLM пропускает ContentFetcher → формирует node с `type: note`
3. **URL на сервис/ресурс** (не статья) → LLM вызывает `fetch_url_meta` (только title + description страницы) → формирует node с `type: link` и аннотацией на основе мета-данных
4. **Текст + URL + инструкции** → LLM интерпретирует инструкции (тема, что сохранить) и выбирает соответствующие инструменты

**Почему function calling**: входной текст из Telegram неструктурирован — может содержать URL, пояснения, инструкции в произвольном порядке. Regex-детекция URL недостаточна: нужно понять _намерение_ (сохранить статью целиком? сделать закладку? записать мысль с ссылкой для контекста?). Function calling позволяет LLM выбирать инструменты на основе семантики.

**Альтернатива**: жёсткий pipeline с regex-детекцией URL → `IngestURL`/`IngestText`. Проще, но не справляется со смешанным вводом и инструкциями пользователя.

### 2. Парсинг URL: Jina Reader API + Go-native fallback

**Решение**: `ContentFetcher` реализован как chain: сначала Jina Reader (`GET https://r.jina.ai/{url}`), при ошибке — go-readability + html-to-markdown. Возвращает структуру `FetchResult`.

```
ContentFetcher interface {
    Fetch(ctx, url) → (*FetchResult, error)
}

FetchResult {
    Title      string
    Content    string      // markdown
    SourceDate *time.Time  // дата публикации (если извлечена)
    Author     string      // автор (если извлечён)
}

JinaFetcher        → HTTP GET r.jina.ai/{url} → FetchResult
ReadabilityFetcher → HTTP GET url → go-readability → html-to-md → FetchResult
ChainFetcher       → Jina → fallback → Readability
```

LLM-оркестратор имеет два инструмента для работы с URL:
- `fetch_url_content` — полное извлечение (для статей) → вызывает `ContentFetcher.Fetch`
- `fetch_url_meta` — только title + description из HTML `<meta>` тегов (для ссылок-закладок) → легковесный HTTP GET + парсинг `<head>`

**Почему Jina primary**: лучшее качество извлечения, поддержка JS-rendered, возвращает готовый markdown с title/date. Оффлайн-first относится к чтению базы, не к добавлению (URL = интернет по определению).

**Почему go-native fallback**: graceful degradation при недоступности Jina (rate limit, таймаут). Для Habr и типичных блогов качество достаточное.

**Отложенные альтернативы**:
- **Trafilatura (Python)** — топ по бенчмаркам, но требует Python-рантайм и subprocess. Рассмотреть, если go-readability даёт неприемлемое качество.
- **Headless browser** — overkill для текущих источников (серверный рендеринг).
- **LLM-extraction** — дорого (HTML = десятки тысяч токенов), неидемпотентно.

### 3. LLM-оркестратор с function calling

**Решение**: единый `LLMOrchestrator` — интерфейс, реализация через OpenAI-совместимый API с function calling. LLM получает контекст базы знаний и решает, какие инструменты вызвать и какие метаданные сформировать.

```
LLMOrchestrator interface {
    Process(ctx, input ProcessInput) → (*ProcessResult, error)
}

ProcessInput {
    Text           string       // входной текст от пользователя
    ExistingThemes []string     // список существующих тем (e.g. ["go/concurrency", "devops/docker"])
    ExistingKeywords []string   // список keywords из всех записей (для консистентности)
}

ProcessResult {
    Keywords    []string
    Annotation  string
    ThemePath   string       // e.g. "go/concurrency"
    Slug        string       // kebab-case имя узла
    Type        string       // "article" | "link" | "note"
    SourceURL   string       // URL источника (если есть)
    SourceDate  *time.Time
    Content     string       // markdown контент (из fetcher или из текста)
    Title       string
}
```

**Контекст для LLM**: в промпт передаются:
- **Существующие темы** — чтобы LLM размещал в существующую иерархию или осознанно предлагал новую тему
- **Существующие keywords** — чтобы LLM переиспользовал уже введённые ключевые слова вместо создания синонимов (например, использовал `goroutines`, а не `goroutine` или `go-routines`, если `goroutines` уже есть в базе)

**Function calling tools**, доступные LLM:
- `fetch_url_content(url)` → вызывает ContentFetcher, возвращает FetchResult
- `fetch_url_meta(url)` → извлекает title + description из `<meta>` тегов
- `create_node(keywords, annotation, theme, slug, type, ...)` → финальный вызов для создания узла

**Библиотека**: `github.com/openai/openai-go/v3`. Работа через Responses API (`client.Responses.New(ctx, params)`), function calling через tool definitions в параметрах запроса.

**Конфигурация**: `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL` через env. По умолчанию — OpenAI API.

**Альтернатива**: без LLM — ручное указание всех метаданных. Противоречит принципу «максимум автоматизации».

### 4. Расширение frontmatter

**Решение**: добавить опциональные поля `type`, `source_url`, `source_date`. Обязательные поля (`keywords`, `created`, `updated`) — без изменений. Валидация не ломается.

```yaml
---
keywords: [goroutines, memory-leak]
created: "2026-03-06T12:00:00Z"
updated: "2026-03-06T12:00:00Z"
annotation: "Статья о типичных утечках горутин"
type: article
source_url: "https://habr.com/..."
source_date: "2026-02-20"
---
```

- `created` — дата добавления в KB
- `source_date` — дата оригинала (если известна)
- `type` — `article` | `link` | `note`

**Почему опциональные**: обратная совместимость с существующими узлами. Старые записи без `type` и `source_url` остаются валидными.

### 5. Store: добавление метода записи

**Решение**: добавить метод `CreateNode` в `Store`.

```
CreateNode(ctx, basePath, CreateNodeParams) → (*Node, error)

CreateNodeParams {
    ThemePath   string       // e.g. "go/concurrency"
    Slug        string       // имя директории узла (kebab-case)
    Frontmatter map[string]any
    Content     string       // markdown тело
}
```

Метод создаёт: директорию `{basePath}/{themePath}/{slug}/`, файл `{slug}.md` с frontmatter + content. Если theme-путь не существует — создаёт промежуточные директории.

**Slug**: генерируется из title или первых слов контента (transliteration + kebab-case). Если slug уже существует — добавить суффикс.

### 6. Git: commit, sync и конфликты

**Решение**: отдельный интерфейс `GitCommitter` с методами для commit и синхронизации. Реализация — `exec.Command("git", ...)` в рабочей директории `basePath`.

```
GitCommitter interface {
    CommitNode(ctx, nodePath, message) → error
    Sync(ctx) → error   // периодический fetch + merge/rebase
}
```

**Commit**: после создания узла — `git add <path> && git commit -m "<message>"`.

**Периодический sync**: сервер на VPS периодически выполняет `git fetch origin && git merge origin/main` (или `rebase`). Интервал — конфигурируемый (env `GIT_SYNC_INTERVAL`, по умолчанию 5 мин). Реализуется как отдельный `runnable` в bootstrap.

**Сценарий использования**: сервер хостится на VPS, но пользователь иногда работает с заметками локально через Cursor. Обе стороны коммитят в один remote.

**Стратегия при конфликтах**: конфликты маловероятны (разные файлы), но возможны при одновременном редактировании одного узла. Стратегия:
1. `git merge` — если merge clean, всё ок
2. Если merge conflict → логировать ошибку, оставить конфликт для ручного разрешения пользователем
3. Не пытаться автоматически резолвить — потеря данных хуже задержки

**Почему отдельный интерфейс**: позволяет не коммитить в тестах (заглушка).

**Альтернатива**: go-git библиотека. Избыточно для `add + commit + fetch + merge`, exec проще.

### 7. Telegram-бот: авторизация и единый вход через IngestText

**Решение**: расширить `handleUpdate` в `telegram.Bot`. Авторизация по Telegram user ID — сравнение `from.id` входящего сообщения с `TELEGRAM_OWNER_ID` из env.

1. Проверить `from.id == TELEGRAM_OWNER_ID` — если нет, игнорировать сообщение
2. Получить текст сообщения (включая пересылки)
3. Вызвать `IngestText` — LLM внутри pipeline определит, что делать
4. Отправить ответ пользователю с подтверждением (путь, keywords, type)

**Почему user ID**: single-user система, достаточно одного числового идентификатора. Не требует токенов, OAuth или сложной auth-логики. ID стабилен, не меняется при смене username.

Для отправки ответа — добавить метод `sendMessage` (POST `/sendMessage`).

**Пересылка**: пока не отличаем от обычного текста. Forwarded message содержит тот же `text` — обрабатывается как текст. Усложнение (разбор forward_from, caption) — в будущем.

**Разделение IngestText/IngestURL**: интерфейс `Ingester` сохраняет оба метода для API (`POST /api/ingest` может явно указать URL). Telegram всегда использует `IngestText` — LLM разберётся.

### 8. Wiring в bootstrap

**Решение**: в `bootstrap.Run()` создавать `PipelineIngester` с реальными зависимостями вместо `StubIngester`. Новые env-переменные: `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`, `JINA_API_KEY` (опционально), `GIT_SYNC_INTERVAL` (по умолчанию `5m`), `TELEGRAM_OWNER_ID` (обязателен при запуске с TELEGRAM_TOKEN).

Если LLM-конфигурация не задана — fallback на `StubIngester` (логировать warning). Это сохраняет работоспособность сервера без LLM (read-only режим).

Регистрация git sync как отдельного `runnable` с настроенным интервалом.

## Risks / Trade-offs

**[Jina Reader недоступен]** → Fallback на go-readability. Качество ниже, но pipeline не падает. Логировать warning при fallback.

**[LLM API недоступен]** → Ingestion невозможен (keywords и theme обязательны для создания узла). Возвращать ошибку, не создавать битые записи. В будущем можно добавить режим "без LLM" с ручным вводом метаданных.

**[LLM генерирует некачественные keywords/theme]** → Пользователь правит постфактум. Рекаталогизация — отдельная задача. Качество промпта критично; итерировать.

**[Дублирование контента]** → Нет проверки на дубли при добавлении. Допустимо на начальном этапе. Дедупликация — в рекаталогизации.

**[Git конфликты при sync]** → Сервер на VPS + локальная работа через Cursor = два источника коммитов. Конфликты маловероятны (обычно разные файлы), но при одновременном редактировании одного узла — merge conflict. Стратегия: логировать ошибку, оставить для ручного разрешения. Не автоматически резолвить.

**[Git sync fails]** → Периодический fetch/merge может упасть (сеть, auth). Логировать ошибку, retry на следующем интервале. Ingestion продолжает работать локально (коммиты копятся, push при восстановлении связи).

**[Slug-коллизии]** → Два узла с одинаковым названием в одной теме. Решение: суффикс `-2`, `-3` и т.д. при коллизии.
