# Personal knowledge database

Система управления персональной базой знаний.

## Концепция

- **Запись** — онлайн: web UI, Telegram, API, MCP. Добавлять заметки удобно из любого места.
- **Чтение** — offline-first + git-first: база хранится локально в отдельной директории под git. Знания всегда доступны без интернета, версионируются, удобно мержатся. **Ничего не потеряется** — надёжная версионируемая база под вашим контролем.

## Мотивация

В современной IT-среде слишком большой поток информации: статьи, новости, релизы, идеи, заметки из чатов, ссылки из Telegram-каналов. Простого “сохранить в избранное” уже недостаточно: со временем становится сложно вспомнить, что именно было сохранено, найти материал по смыслу и задать вопрос по накопленным знаниям.

**knowledge-db** создаётся как персональная база знаний, где материалы не только складываются в архив, но и обрабатываются: получают аннотации, ключевые слова, структуру, индексируются для поиска и могут использоваться как контекст для RAG-чата.

Ещё одна причина — надёжность доступа. Статьи исчезают из интернета, сайты меняют структуру, сервисы бывают недоступны, сеть может подвести. Поэтому основой выбран Git и локальные Markdown-файлы: копия базы всегда доступна на рабочей машине, ноутбуке или VPS, а история изменений остаётся прозрачной и версионируемой.

Проект рассчитан на два основных режима:

- **Локальное чтение и работа без сетевых LLM** — база доступна оффлайн; при необходимости можно подключать локальные модели через Ollama или LM Studio.
- **Self-hosted режим на своей VPS** — удобен для записи, быстрого доступа со смартфона, Telegram-бота, синхронизации и более широких AI-сценариев.

Заполнение базы поддерживает разные привычные входы: веб-интерфейс, Telegram-бот для пересланного контента из каналов, API, а также массовый импорт из сохранённых заметок Telegram.

## Структура проекта

```
knowledge-db/
├── cmd/
│   └── kb/          # serve + validate/init/служебные команды
├── internal/
│   ├── kb/          # работа с data/, валидация, дерево тем
│   ├── api/         # HTTP handlers, роутинг
│   ├── ingestion/   # интерфейс Ingester, pipeline
│   ├── mcp/         # MCP endpoint /api/mcp
│   └── ui/          # embed статики (embed.go, static/)
├── web/             # React исходники (Vite)
├── .cursor/skills/  # Agent skills
├── openspec/        # Спецификации (OpenSpec workflow)
├── data/            # База знаний (git subtree/submodule, локальная)
├── AGENTS.md        # Руководство для AI-агентов
└── README.md
```

## Быстрый старт

```bash
# Сборка
task build

# Запуск сервера (KB_DATA_PATH обязателен)
KB_DATA_PATH=/path/to/data ./kb serve

# Без git (коммиты и sync отключены)
KB_DATA_PATH=/path/to/data KB_GIT_DISABLED=true ./kb serve

# CLI: валидация структуры базы
./kb validate --path /path/to/data

# CLI: инициализация новой базы
./kb init --path /path/to/data

# CLI: инициализация с примером узла (формат Obsidian)
./kb init --path /path/to/data --example

# CLI: полная перестройка embedding-index (нужны KB_EMBEDDING_*)
KB_DATA_PATH=/path/to/data ./kb rebuild-index
```

### Апгрейд на версию с UUID v7 (`id` в frontmatter)

После обновления бинарника для существующей базы:

```bash
# 1. Резервная копия базы знаний
cp -a /path/to/data /path/to/data.bak

# 2. Присвоить id всем узлам без поля id (сначала dry-run)
./kb migrate-node-ids --path /path/to/data --dry-run
./kb migrate-node-ids --path /path/to/data

# 3. Пересобрать embedding-index (схема index.db меняется)
KB_DATA_PATH=/path/to/data ./kb rebuild-index
# или: ./kb serve и POST /api/index/rebuild через UI или curl
```

Старый `index.db` (path PK) при старте мигрируется на схему `node_id` PK; при несовместимости данные индекса сбрасываются — нужен rebuild.

## Команды Taskfile


| Команда             | Описание                           |
| ------------------- | ---------------------------------- |
| `task build`        | Собрать web + kb |
| `task build-kb`     | Собрать только kb |
| `task web:dev`      | Vite dev server (HMR, прокси /api) |
| `task server:dev`   | kb с hot reload (air), без пересборки embedded UI |
| `task dev`          | Подсказка по запуску dev-окружения |
| `task test`         | Запустить тесты                    |
| `task lint`         | golangci-lint                      |
| `task lint:fix`     | golangci-lint с автоисправлением   |


## Разработка

Для разработки запустите в двух терминалах:

1. `task web:dev` — Vite dev server ([http://localhost:5173](http://localhost:5173))
2. `task server:dev` — kb с hot reload (embedded UI не пересобирается)

Если нужно обновить встроенную статику (`internal/ui/static`) для бинарника `kb`, используйте `task build-kb` или `task build`.

Для `server:dev` нужен [air](https://github.com/air-verse/air): `task server:dev:install`.

## Конфигурация


| Переменная                                                       | Описание                                                                                                                                                                                        |
| ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **KB_DATA_PATH**                                                 | Путь к корню базы знаний (обязателен для kb)                                                                                                                                             |
| **KB_HTTP_ADDR**                                                 | Адрес HTTP-сервера (по умолчанию :8080)                                                                                                                                                         |
| **KB_MCP_API_KEY**                                               | API-ключ для MCP endpoint `/api/mcp` (заголовок `Authorization: Bearer <key>`). Если пустой/не задан — MCP endpoint отключён                                                                     |
| **KB_MCP_DEBUG_API_KEY**                                         | API-ключ для debug MCP endpoint `/api/mcp/debug` (заголовок `Authorization: Bearer <key>`). Если пустой/не задан — debug MCP endpoint отключён                                                   |
| **KB_TELEGRAM_RAW_LOG_ENABLED**                                  | Включить запись сырых Telegram update payload в `.kb/telegram-raw/*.ndjson` и периодическую очистку файлов старше 14 дней                                                                       |
| **KB_GIT_DISABLED**                                              | Отключить git (коммиты и sync)                                                                                                                                                                  |
| **KB_LOGIN**, **KB_PASSWORD**                                    | Пароль: при задании **обоих** включается `password` в `auth_methods` (можно вместе с OAuth)                                                                                                    |
| **KB_GOOGLE_OAUTH_CLIENT_ID**, **KB_GOOGLE_OAUTH_CLIENT_SECRET** | Google OAuth: клиент типа **Web application** в Google Cloud Console                                                                                                                          |
| **KB_GOOGLE_OAUTH_REDIRECT_URL**                                 | Google OAuth: redirect URI — `https://<хост API>/api/auth/google/callback`                                                                                                                    |
| **KB_YANDEX_OAUTH_CLIENT_ID**, **KB_YANDEX_OAUTH_CLIENT_SECRET** | Yandex OAuth: приложение на [oauth.yandex.com](https://oauth.yandex.com/)                                                                                                                     |
| **KB_YANDEX_OAUTH_REDIRECT_URL**                                 | Yandex OAuth: redirect — `https://<хост API>/api/auth/yandex/callback`                                                                                                                        |
| **KB_OAUTH_STATE_SECRET**                                        | Любой OAuth: секрет подписи `state` (CSRF), ≥16 байт                                                                                                                                            |
| **KB_AUTH_ALLOWED_EMAILS**                                       | Любой OAuth: allowlist email через запятую (общий для Google и Yandex)                                                                                                                          |
| **KB_SESSION_TTL**                                               | TTL сессии (по умолчанию 8h)                                                                                                                                                                    |
| **TELEGRAM_TOKEN**                                               | Токен Telegram-бота (опционально)                                                                                                                                                               |
| **TELEGRAM_OWNER_ID**                                            | Telegram user ID владельца (обязателен при TELEGRAM_TOKEN)                                                                                                                                      |
| **KB_PUBLIC_WEB_BASE_URL**                                       | Публичный URL SPA без `/` (например `https://kb.example`); **обязателен при любом OAuth** (редирект после callback), в Telegram — «Открыть на сайте»                                           |
| **LLM_API_URL**, **LLM_API_KEY**, **LLM_MODEL**                  | LLM для ingestion (OpenAI-совместимый API)                                                                                                                                                      |
| **JINA_API_KEY**                                                 | Ключ Jina для эмбеддингов (опционально)                                                                                                                                                         |
| **KB_EMBEDDING_ENABLED**                                         | Включить RAG и чат-бота (true/false, по умолчанию false)                                                                                                                                        |
| **KB_EMBEDDING_API_URL**                                         | URL API для эмбеддингов (Ollama: http://localhost:11434)                                                                                                                                        |
| **KB_EMBEDDING_API_KEY**                                         | API key для эмбеддингов (Ollama: пустой)                                                                                                                                                        |
| **KB_EMBEDDING_MODEL**                                           | Модель для эмбеддингов (по умолчанию text-embedding-3-small)                                                                                                                                    |
| **KB_CHAT_MODEL**                                                | Модель для чат-бота (например llama3, mistral)                                                                                                                                                 |
| **KB_CHAT_API_URL**                                              | URL API для чата (если отличается от KB_EMBEDDING_API_URL)                                                                                                                                      |
| **KB_CHAT_API_KEY**                                              | API key для чата                                                                                                                                                                               |
| **KB_EMBEDDING_RATE_LIMIT**                                     | Rate limit между запросами к API эмбеддингов (по умолчанию 1s)                                                                                                                                |
| **LOG_LEVEL**                                                    | Уровень логирования: debug, info, warn, error (по умолчанию info)                                                                                                                             |
| **GIT_SYNC_INTERVAL**                                            | Интервал git sync (по умолчанию 5m)                                                                                                                                                           |
| **VITE_API_URL**                                                 | URL API для web (по умолчанию [http://localhost:8080](http://localhost:8080))                                                                                                                   |
| **ALLOWED_CORS_ORIGIN**                                          | CORS origin для dev (например [http://localhost:5173](http://localhost:5173))                                                                                                                   |

### MCP endpoint

- MCP доступен по `POST/GET /api/mcp` на том же сервере.
- Для доступа обязателен Bearer-токен из `KB_MCP_API_KEY`:
  - `Authorization: Bearer <KB_MCP_API_KEY>`
- При отсутствии/невалидном токене сервер возвращает `401 Unauthorized`.
- Если `KB_MCP_API_KEY` пустой или не задан, маршрут `/api/mcp` не обслуживается (MCP отключён).

### Debug MCP endpoint

- Debug MCP доступен по `POST/GET /api/mcp/debug`.
- Для доступа обязателен Bearer-токен из `KB_MCP_DEBUG_API_KEY`.
- Если `KB_MCP_DEBUG_API_KEY` пустой или не задан, debug endpoint не обслуживается.


## Режимы запуска

### Веб-авторизация

Способы входа **независимы**: каждый включается полным набором env для этого способа. `GET /api/auth/session` возвращает `auth_methods` — массив в порядке `password`, `google`, `yandex` (только настроенные). Поле `auth_mode` устарело: один способ → его имя; несколько → `multi`.

| Способ | Что задать |
| ------ | ---------- |
| **password** | `KB_LOGIN` и `KB_PASSWORD` |
| **google** | `KB_GOOGLE_OAUTH_*` + `KB_OAUTH_STATE_SECRET` + непустой `KB_AUTH_ALLOWED_EMAILS` + `KB_PUBLIC_WEB_BASE_URL` |
| **yandex** | `KB_YANDEX_OAUTH_*` + те же общие OAuth-переменные |

Частичный набор внутри одного способа или «висящие» `KB_OAUTH_STATE_SECRET` / `KB_AUTH_ALLOWED_EMAILS` без полного Google/Yandex — сервер **не стартует**.

**Открытый доступ** (по умолчанию): ни один способ не настроен.

```bash
KB_DATA_PATH=/path/to/data ./kb serve
```

**Только пароль:**

```bash
KB_DATA_PATH=/path/to/data KB_LOGIN=admin KB_PASSWORD=secret ./kb serve
```

**Общие OAuth-переменные** (при полном Google и/или Yandex): `KB_OAUTH_STATE_SECRET`, `KB_AUTH_ALLOWED_EMAILS`, `KB_PUBLIC_WEB_BASE_URL` (URL SPA без `/`, куда редиректить после callback).

**Google OAuth**

1. [Google Cloud Console](https://console.cloud.google.com/) → OAuth client **Web application**.
2. Redirect URI = `KB_GOOGLE_OAUTH_REDIRECT_URL` (например `https://api.example.com/api/auth/google/callback` или `http://localhost:8080/api/auth/google/callback` в test mode).
3. Вход только при `email_verified` и email из allowlist.

**Yandex OAuth**

1. [oauth.yandex.com](https://oauth.yandex.com/) → приложение, право **доступ к email**.
2. Redirect URI = `KB_YANDEX_OAUTH_REDIRECT_URL` → `/api/auth/yandex/callback`.
3. Allowlist по полю `default_email` из userinfo (нет `email_verified` как у Google).

**Production:** не оставляйте `KB_LOGIN`/`KB_PASSWORD` на публичном инстансе без необходимости; HTTPS и `X-Forwarded-Proto`; `ALLOWED_CORS_ORIGIN` = origin SPA; ротируйте `KB_OAUTH_STATE_SECRET`; минимальный allowlist.

**Dev:** пароль + OAuth на localhost — задайте `KB_PUBLIC_WEB_BASE_URL=http://localhost:5173` и callback на `:8080` для Google/Yandex.

**Безопасность:** при пароле и OAuth — два канала атаки; rate limit на `POST /api/auth/login`; OAuth только с allowlist.

**Миграция:** раньше пароль и Google были взаимоисключающими — теперь можно добавить `KB_LOGIN`/`KB_PASSWORD` к существующему Google без смены OAuth env. Клиентам UI: использовать `auth_methods`, не только `auth_mode`.

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

CORS: `ALLOWED_CORS_ORIGIN` для production. OAuth-кнопки ведут на `GET /api/auth/google` и `GET /api/auth/yandex` (тот же origin, что API, или прокси). TLS обязателен вне localhost (`Secure` cookie).

## RAG и чат-бот

Сервер поддерживает семантический поиск и чат-бот на основе RAG (Retrieval Augmented Generation). Работает с локальными LLM через Ollama и LM Studio.

**Полезные переменные окружения:**

- `LOG_LEVEL=debug` — подробное логирование синхронизации индекса
- `KB_EMBEDDING_RATE_LIMIT=500ms` — rate limit между запросами (по умолчанию 1s)

### Быстрый старт с локальными моделями

**1. Запустите Ollama** (для эмбеддингов):

```bash
ollama serve
ollama pull bge-m3
```

**2. Запустите LM Studio** (для чата):

- Скачайте LM Studio с https://lmstudio.ai
- Загрузите модель (llama3, mistral и т.п.)
- Запустите локальный сервер (кнопка в левом нижнем углу)
- По умолчанию: http://localhost:1234/v1

**3. Запустите kb**:

```bash
export KB_DATA_PATH=/path/to/data
export KB_EMBEDDING_ENABLED=true
export KB_EMBEDDING_API_URL=http://localhost:11434
export KB_EMBEDDING_API_KEY=""
export KB_EMBEDDING_MODEL=bge-m3
export KB_CHAT_MODEL=openai/gpt-oss-20b
export KB_CHAT_API_URL=http://localhost:1234/v1
export KB_CHAT_API_KEY="-"

./kb serve
```

### API эндпоинты

- `GET /api/chats` — список чат-сессий
- `POST /api/chats` — создать чат-сессию (`{ "title": "..." }`, title опционален)
- `GET /api/chats/{id}` — получить чат и сообщения
- `PATCH /api/chats/{id}` — переименовать чат (`{ "title": "..." }`)
- `DELETE /api/chats/{id}` — удалить чат
- `POST /api/chat` — отправить сообщение в чат-сессию (SSE stream)
  - body: `{ "session_id": "...", "message": "...", "source_paths": ["optional/path"] }`
  - summary-сообщения служебные и не возвращаются в пользовательской ленте
- `GET /api/search?q=запрос` — семантический поиск
- `GET /api/index/status` — статус индекса

### Web UI (чат)

- Чаты работают как список сессий: создать, выбрать, переименовать, удалить.
- На мобильных устройствах список чатов открывается через выезжающий сайдбар.
- Источники в ответах чата кликабельны и открываются в новой вкладке.

### Переиндексация

При старте сервера автоматически выполняется полная синхронизация индекса с файловой системой. После добавления новых записей можно запустить переиндексацию вручную:

```bash
# без запущенного сервера (нужны KB_EMBEDDING_* из .env)
KB_DATA_PATH=/path/to/data ./kb rebuild-index

# или через API при работающем kb serve
curl -X POST http://localhost:8080/api/index/rebuild
```

### Логирование синхронизации индекса

При уровне `LOG_LEVEL=debug` логируются все операции синхронизации:

- `sync: performing initial full reconcile` — начальный reconcile при старте
- `sync: event queued` — событие поставлено в очередь
- `sync: event received` — событие получено на обработку
- `sync: processing node` — обработка ноды
- `sync: node deleted from index` — нода удалена из индекса
- `sync: node unchanged, skipping` — хеши совпали, нода пропущена
- `sync: node indexed` — нода успешно проиндексирована
- `sync: embedding article chunks` — чанкинг article
- `sync: article chunks indexed` — чанки проиндексированы
- `sync: full reconcile complete` — финальная статистика (total_nodes, stale_deleted, duration_ms)

## Docker

Образ собирается в GitHub Actions при push в `main` и публикуется в [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).

```bash
# Сборка локально
docker build -t kb .

# Запуск (база — volume)
docker run -d -p 8080:8080 \
  -v /path/to/knowledge-base:/data \
  -e KB_DATA_PATH=/data \
  ghcr.io/OWNER/knowledge-db:latest
```

Образ устанавливает `cursor-agent` на этапе сборки (`curl https://cursor.com/install -fsS | bash`).

### Настройка Git в Docker

Сервер выполняет `git push` и `git fetch` для синхронизации базы с remote. Для доступа к репозиторию по SSH нужны учётные данные внутри контейнера.

**Вариант 1: Deploy key (рекомендуется)**

Создайте отдельный SSH-ключ для репозитория и смонтируйте его в контейнер:

```bash
# На хосте
ssh-keygen -t ed25519 -f /opt/kb-deploy/key -N "" -C "kb-deploy"
# Добавьте /opt/kb-deploy/key.pub как deploy key в GitLab/GitHub
```

```bash
docker run -d -p 8080:8080 \
  -v /path/to/knowledge-base:/data \
  -v /opt/kb-deploy:/opt/kb-deploy:ro \
  -e KB_DATA_PATH=/data \
  -e GIT_SSH_COMMAND="ssh -i /opt/kb-deploy/key -o StrictHostKeyChecking=accept-new" \
  ghcr.io/OWNER/knowledge-db:latest
```

**Вариант 2: Монтирование ~/.ssh**

Контейнер работает от пользователя `kb`. Монтируйте SSH в `/home/kb/.ssh`:

```bash
docker run -d -p 8080:8080 \
  -v /path/to/knowledge-base:/data \
  -v ~/.ssh:/home/kb/.ssh:ro \
  -e KB_DATA_PATH=/data \
  ghcr.io/OWNER/knowledge-db:latest
```

Права на хосте: `~/.ssh` — 700, ключи — 600.

**Вариант 3: Без git**

Если push/fetch не нужны (например, только локальная база):

```bash
docker run -d -p 8080:8080 \
  -v /path/to/knowledge-base:/data \
  -e KB_DATA_PATH=/data \
  -e KB_GIT_DISABLED=true \
  ghcr.io/OWNER/knowledge-db:latest
```

**Подготовка репозитория на хосте**

Перед первым запуском клонируйте базу на хост:

```bash
mkdir -p /path/to/knowledge-base
git clone git@gitlab.com:user/my-knowledge-base.git /path/to/knowledge-base
```

Убедитесь, что `git config user.name` и `user.email` заданы в репозитории — они нужны для коммитов.

## Лицензия

MIT © 2026 Igor Lazarev
