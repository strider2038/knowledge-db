# Personal knowledge database

Локальная система управления персональной базой знаний: markdown под git, ingestion через LLM, поиск и RAG-чат.

## Концепция

- **Запись** — удобно онлайн: web UI, Telegram, REST API, MCP, массовый импорт из экспорта Telegram.
- **Чтение** — offline-first и git-first: база в отдельном репозитории на вашей машине или VPS, версионируется и мержится без привязки к облачному SaaS.

Материалы не только сохраняются, но и структурируются: аннотации, keywords, размещение в дереве тем, индекс для поиска, контекст для RAG. Статьи в интернете исчезают — локальные файлы и git-история остаются под вашим контролем.

**Режимы:** локальное чтение без сетевых LLM; локальный AI (Ollama / LM Studio); self-hosted VPS для записи с телефона, бота и синхронизации.

## Возможности

- Дерево тем и узлы в markdown с YAML frontmatter (UUID `id`, keywords, type, title, …)
- Ingestion по URL и тексту (LLM, автоперевод, раскрытие ссылок — опционально)
- Telegram-бот для пересланного контента
- Массовый импорт сохранённых заметок Telegram (`KB_UPLOADS_DIR`)
- Git commit/sync из UI и API
- Keyword- и семантический поиск, RAG-чат с источниками
- MCP (`/api/mcp`) для агентов
- Авторизация: без пароля, пароль, Google OAuth, Yandex OAuth (независимо комбинируются)
- CLI: validate, init, migrate-node-ids, rebuild-index, обслуживание контента

## Требования

- **Go** 1.25+ (сборка `kb`)
- **Node.js** 22+ (сборка `web/`, dev с HMR)
- Опционально: [Task](https://taskfile.dev), [air](https://github.com/air-verse/air) для `task server:dev`

## База знаний (отдельно от приложения)

Репозиторий **knowledge-db** — это приложение. Сами заметки живут в **другом git-репозитории**, путь задаётся `KB_DATA_PATH` (часто `my-knowledge-base`).

```bash
mkdir -p /path/to/my-knowledge-base
cd /path/to/my-knowledge-base
git init   # при необходимости

/path/to/kb init --path .
# опционально: --example — sample-node.md в example/topic/
```

`init` создаёт:

- `.gitignore` (`.local/`, `.kb/`, Obsidian, OS)
- `.agents/skills/knowledge-db/SKILL.md` — инструкции для агентов (путь к базе уже подставлен)

Индекс эмбеддингов: `{KB_DATA_PATH}/.kb/index.db` (не коммитится).

Формат узлов: [.agents/skills/knowledge-db/SKILL.md](.agents/skills/knowledge-db/SKILL.md) в репозитории приложения; в базе — копия после `init`.

## Быстрый старт

```bash
cp .env.example .env   # задайте KB_DATA_PATH и при необходимости KB_UPLOADS_DIR

task build
export KB_DATA_PATH=/path/to/my-knowledge-base
export KB_UPLOADS_DIR=/path/to/uploads   # нужен для POST /api/import/telegram
./kb serve
```

Откройте http://localhost:8080 (встроенный UI).

**Разработка UI:** в двух терминалах — `task web:dev` (http://localhost:5173) и `task server:dev` (API на :8080, нужен `task server:dev:install` для air).

**Без Task:**

```bash
cd web && npm ci && npm run build
go build -o kb ./cmd/kb
KB_DATA_PATH=/path/to/my-knowledge-base ./kb serve
```

**Без git в базе:**

```bash
KB_GIT_DISABLED=true KB_DATA_PATH=/path/to/my-knowledge-base ./kb serve
```

### Апгрейд на UUID v7 (`id` в frontmatter)

```bash
cp -a /path/to/my-knowledge-base /path/to/my-knowledge-base.bak
./kb migrate-node-ids --path /path/to/my-knowledge-base --dry-run
./kb migrate-node-ids --path /path/to/my-knowledge-base
KB_DATA_PATH=/path/to/my-knowledge-base ./kb rebuild-index
```

## Структура проекта

```
knowledge-db/
├── cmd/kb/              # CLI: serve, validate, init, …
├── internal/
│   ├── bootstrap/       # конфиг, wiring
│   ├── api/             # HTTP
│   ├── kb/              # FS, frontmatter, дерево
│   ├── ingestion/       # pipeline, LLM
│   ├── index/           # SQLite index, embeddings
│   ├── chat/            # сессии RAG-чата
│   ├── telegram/        # бот
│   ├── mcp/
│   └── ui/              # embedded SPA
├── web/                 # React (Vite)
├── .agents/skills/      # agent skills (шаблон knowledge-db)
├── docs/                # ADR, презентации
├── openspec/
├── AGENTS.md
└── README.md
```

## CLI

| Команда | Назначение |
| ------- | ---------- |
| `serve` | HTTP API, UI, фоновые workers |
| `validate --path PATH` | проверка структуры и frontmatter |
| `init --path PATH [--example]` | `.gitignore`, skill в `.agents/skills/knowledge-db/` |
| `migrate-node-ids` | присвоить `id` узлам без поля |
| `rebuild-index` | полная перестройка `.kb/index.db` |
| `dump-images` | выгрузка изображений из узлов |
| `expand-urls` | раскрытие URL в контенте |
| `reindex-links` | переиндексация link-узлов (Jina) |

```bash
./kb validate --path /path/to/my-knowledge-base
./kb init --path /path/to/my-knowledge-base --example
KB_DATA_PATH=/path/to/my-knowledge-base ./kb rebuild-index
```

## Команды Task

| Команда | Описание |
| ------- | -------- |
| `task build` | web + `kb` |
| `task build-kb` | только `kb` (web → embed) |
| `task web:dev` | Vite, прокси `/api` → :8080 |
| `task server:dev` | `kb serve` с air |
| `task dev` | подсказка по dev-окружению |
| `task test` | Go + web тесты |
| `task lint` | golangci-lint + ESLint |
| `task web:test` / `task web:lint` | только frontend |

## Конфигурация

Полный список переменных: [.env.example](.env.example).

### Core

| Переменная | Описание |
| ---------- | -------- |
| **KB_DATA_PATH** | Корень базы знаний (обязателен для `serve`) |
| **KB_UPLOADS_DIR** | Каталог импорта (сессии Telegram JSON); без него import API недоступен |
| **KB_HTTP_ADDR** | Адрес HTTP (по умолчанию `:8080`) |
| **KB_GIT_DISABLED** | Отключить git commit/sync |
| **GIT_SYNC_INTERVAL** | Интервал фонового sync (по умолчанию `5m`) |
| **LOG_LEVEL** | `debug`, `info`, `warn`, `error` |
| **ALLOWED_CORS_ORIGIN** | CORS для dev (например `http://localhost:5173`) |
| **VITE_API_URL** | URL API при сборке web (по умолчанию `http://localhost:8080`) |

### Ingestion / LLM

| Переменная | Описание |
| ---------- | -------- |
| **LLM_API_URL**, **LLM_API_KEY**, **LLM_MODEL** | OpenAI-совместимый API для ingestion |
| **JINA_API_KEY** | Jina (опционально) |
| **KB_AUTO_TRANSLATE** | Автоперевод при ingestion (по умолчанию `true`) |
| **KB_INGEST_EXPAND_URLS** | Раскрытие URL после LLM (по умолчанию `true`) |

### Telegram

| Переменная | Описание |
| ---------- | -------- |
| **TELEGRAM_TOKEN** | Токен бота (без токена бот не стартует) |
| **TELEGRAM_OWNER_ID** | User ID владельца; при `TELEGRAM_TOKEN` значение `0` — **ошибка старта** |
| **KB_PUBLIC_WEB_BASE_URL** | URL SPA без `/` (ссылки в боте, OAuth redirect) |
| **KB_TELEGRAM_RAW_LOG_ENABLED** | Сырые update в `.kb/telegram-raw/*.ndjson` |

### MCP

| Переменная | Описание |
| ---------- | -------- |
| **KB_MCP_API_KEY** | Bearer для `/api/mcp`; пусто — endpoint выключен |
| **KB_MCP_DEBUG_API_KEY** | Bearer для `/api/mcp/debug` |

### Auth (кратко)

Способы входа независимы; `GET /api/auth/session` → `auth_methods`.

| Способ | Переменные |
| ------ | ---------- |
| password | `KB_LOGIN` + `KB_PASSWORD` |
| google | `KB_GOOGLE_OAUTH_*` + общие OAuth |
| yandex | `KB_YANDEX_OAUTH_*` + общие OAuth |

Общие для OAuth: `KB_OAUTH_STATE_SECRET`, `KB_AUTH_ALLOWED_EMAILS`, `KB_PUBLIC_WEB_BASE_URL`, `KB_SESSION_TTL`.

Частичный набор внутри одного способа — сервер **не стартует**. Подробности — раздел [Веб-авторизация](#веб-авторизация) ниже.

### Embeddings / RAG

| Переменная | Описание |
| ---------- | -------- |
| **KB_EMBEDDING_ENABLED** | RAG и чат (по умолчанию `false`) |
| **KB_EMBEDDING_API_URL**, **KB_EMBEDDING_API_KEY**, **KB_EMBEDDING_MODEL** | API эмбеддингов; при `ENABLED=true` URL и **непустой** key обязательны |
| **KB_CHAT_MODEL**, **KB_CHAT_API_URL**, **KB_CHAT_API_KEY** | Чат (если URL чата задан — key обязателен) |
| **KB_SEARCH_REWRITE_ENABLED** | Переписывание запроса перед векторным поиском |
| **KB_EMBEDDING_RATE_LIMIT** | Пауза между запросами к embedding API (по умолчанию `1s`) |

## Веб-авторизация

Способы входа **независимы**. `GET /api/auth/session` возвращает `auth_methods` — массив в порядке `password`, `google`, `yandex`. Поле `auth_mode` устарело.

**Открытый доступ** (по умолчанию): ни один способ не настроен.

```bash
KB_DATA_PATH=/path/to/data ./kb serve
```

**Только пароль:**

```bash
KB_DATA_PATH=/path/to/data KB_LOGIN=admin KB_PASSWORD=secret ./kb serve
```

**Google OAuth:** [Google Cloud Console](https://console.cloud.google.com/) → Web application → redirect `KB_GOOGLE_OAUTH_REDIRECT_URL` (`…/api/auth/google/callback`). Вход при `email_verified` и email из allowlist.

**Yandex OAuth:** [oauth.yandex.com](https://oauth.yandex.com/) → redirect `…/api/auth/yandex/callback`, право email. Allowlist по `default_email`.

**Production:** HTTPS, `X-Forwarded-Proto`, `ALLOWED_CORS_ORIGIN` = origin SPA, минимальный allowlist, не оставляйте слабый пароль на публичном инстансе.

**Dev с OAuth:** `KB_PUBLIC_WEB_BASE_URL=http://localhost:5173`, callback на `:8080`.

```bash
export KB_DATA_PATH=/path/to/data
export KB_PUBLIC_WEB_BASE_URL=https://kb.example
export KB_GOOGLE_OAUTH_CLIENT_ID=....apps.googleusercontent.com
export KB_GOOGLE_OAUTH_CLIENT_SECRET=...
export KB_GOOGLE_OAUTH_REDIRECT_URL=https://api.example.com/api/auth/google/callback
export KB_OAUTH_STATE_SECRET=$(openssl rand -hex 32)
export KB_AUTH_ALLOWED_EMAILS=you@example.com
./kb serve
```

## RAG и чат-бот

Семантический поиск и чат (Ollama, LM Studio, OpenAI-compatible API).

`LOG_LEVEL=debug` — подробные логи sync индекса. `KB_EMBEDDING_RATE_LIMIT=500ms` — ускорить переиндексацию (осторожно с лимитами API).

### Быстрый старт (Ollama + LM Studio)

```bash
ollama serve && ollama pull bge-m3
# LM Studio: локальный сервер на http://localhost:1234/v1

export KB_DATA_PATH=/path/to/data
export KB_EMBEDDING_ENABLED=true
export KB_EMBEDDING_API_URL=http://localhost:11434
export KB_EMBEDDING_API_KEY="-"
export KB_EMBEDDING_MODEL=bge-m3
export KB_CHAT_MODEL=openai/gpt-oss-20b
export KB_CHAT_API_URL=http://localhost:1234/v1
export KB_CHAT_API_KEY="-"
./kb serve
```

### API (основное)

- `GET /api/search`, `POST /api/search` — поиск
- `GET /api/chats`, `POST /api/chats`, `GET/PATCH/DELETE /api/chats/{id}`
- `POST /api/chat` — сообщение (SSE), body: `{ "session_id", "message", "source_paths"? }`
- `GET /api/index/status`, `POST /api/index/rebuild`
- `GET /healthz`, `GET /readyz` — health checks

### Переиндексация

При старте — полный reconcile. Вручную:

```bash
KB_DATA_PATH=/path/to/data ./kb rebuild-index
# или: curl -X POST http://localhost:8080/api/index/rebuild
```

## Docker

Образ публикуется в GHCR при успешном CI на `main`: `ghcr.io/strider2038/knowledge-db:latest`.

```bash
docker build -t kb .
docker run -d -p 8080:8080 \
  -v /path/to/knowledge-base:/data \
  -e KB_DATA_PATH=/data \
  ghcr.io/strider2038/knowledge-db:latest
```

В образе установлен `cursor-agent` (фоновые jobs нормализации/редактирования узлов в UI).

### Git в Docker

**Deploy key (рекомендуется):**

```bash
docker run -d -p 8080:8080 \
  -v /path/to/knowledge-base:/data \
  -v /opt/kb-deploy:/opt/kb-deploy:ro \
  -e KB_DATA_PATH=/data \
  -e GIT_SSH_COMMAND="ssh -i /opt/kb-deploy/key -o StrictHostKeyChecking=accept-new" \
  ghcr.io/strider2038/knowledge-db:latest
```

**Без git:** `-e KB_GIT_DISABLED=true`

Перед первым запуском клонируйте базу на хост и задайте `git config user.name` / `user.email` в репозитории.

## Для разработчиков

- [AGENTS.md](AGENTS.md) — правила для AI-агентов, в т.ч. синхронизация frontmatter
- [docs/adr/](docs/adr/) — архитектурные решения
- [openspec/](openspec/) — спецификации и changes

```bash
task test
task lint
go test ./... -race
```

## Лицензия

MIT © 2026 Igor Lazarev
