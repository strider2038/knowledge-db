# Personal knowledge database

Система управления персональной базой знаний.

## Концепция

- **Запись** — онлайн: web UI, Telegram, API, MCP. Добавлять заметки удобно из любого места.
- **Чтение** — offline-first + git-first: база хранится локально в отдельной директории под git. Знания всегда доступны без интернета, версионируются, удобно мержатся. **Ничего не потеряется** — надёжная версионируемая база под вашим контролем.

## Структура проекта

```
knowledge-db/
├── cmd/
│   ├── kb-server/   # API + UI + Telegram bot + MCP
│   └── kb-cli/      # validate, init
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
KB_DATA_PATH=/path/to/data ./kb-server

# Без git (коммиты и sync отключены)
KB_DATA_PATH=/path/to/data KB_GIT_DISABLED=true ./kb-server

# CLI: валидация структуры базы
./kb-cli validate --path /path/to/data

# CLI: инициализация новой базы
./kb-cli init --path /path/to/data

# CLI: инициализация с примером узла (формат Obsidian)
./kb-cli init --path /path/to/data --example
```

## Команды Taskfile

| Команда | Описание |
|---------|----------|
| `task build` | Собрать web + kb-server + kb-cli |
| `task build-server` | Собрать только kb-server |
| `task build-cli` | Собрать только kb-cli |
| `task web:dev` | Vite dev server (HMR, прокси /api) |
| `task server:dev` | kb-server с hot reload (air) |
| `task dev` | Подсказка по запуску dev-окружения |
| `task test` | Запустить тесты |
| `task lint` | golangci-lint |
| `task lint:fix` | golangci-lint с автоисправлением |

## Разработка

Для разработки запустите в двух терминалах:

1. `task web:dev` — Vite dev server (http://localhost:5173)
2. `task server:dev` — kb-server с hot reload

Для `server:dev` нужен [air](https://github.com/air-verse/air): `task server:dev:install`.

## Конфигурация

| Переменная | Описание |
|------------|----------|
| **KB_DATA_PATH** | Путь к корню базы знаний (обязателен для kb-server) |
| **KB_HTTP_ADDR** | Адрес HTTP-сервера (по умолчанию :8080) |
| **KB_GIT_DISABLED** | Отключить git (коммиты и sync) |
| **KB_LOGIN**, **KB_PASSWORD** | Опциональная авторизация: при задании обоих включается защита API и web UI |
| **KB_SESSION_TTL** | TTL сессии (по умолчанию 8h) |
| **TELEGRAM_TOKEN** | Токен Telegram-бота (опционально) |
| **TELEGRAM_OWNER_ID** | Telegram user ID владельца (обязателен при TELEGRAM_TOKEN) |
| **LLM_API_URL**, **LLM_API_KEY**, **LLM_MODEL** | LLM для ingestion (OpenAI-совместимый API) |
| **JINA_API_KEY** | Ключ Jina для эмбеддингов (опционально) |
| **GIT_SYNC_INTERVAL** | Интервал git sync (по умолчанию 5m) |
| **VITE_API_URL** | URL API для web (по умолчанию http://localhost:8080) |
| **ALLOWED_CORS_ORIGIN** | CORS origin для dev (например http://localhost:5173) |

## Режимы запуска

**Открытый режим** (по умолчанию): без `KB_LOGIN`/`KB_PASSWORD` API и web UI доступны без авторизации.

```bash
KB_DATA_PATH=/path/to/data ./kb-server
```

**Режим с авторизацией**: при задании `KB_LOGIN` и `KB_PASSWORD` требуется вход через форму на `/login`.

```bash
KB_DATA_PATH=/path/to/data KB_LOGIN=admin KB_PASSWORD=secret ./kb-server
```

При публичном доступе (вне localhost) рекомендуется использовать TLS: cookie `Secure` требует HTTPS. Настройте reverse proxy (nginx, Caddy) с TLS и корректные заголовки `X-Forwarded-Proto`, `X-Forwarded-For`.

При включённой авторизации для production задайте `ALLOWED_CORS_ORIGIN` (origin вашего web UI) — это усиливает проверку Origin/Referer для state-changing auth-запросов.

## Docker

Образ собирается в GitHub Actions при push в `main` и публикуется в [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).

```bash
# Сборка локально
docker build -t kb-server .

# Запуск (база — volume)
docker run -d -p 8080:8080 \
  -v /path/to/knowledge-base:/data \
  -e KB_DATA_PATH=/data \
  ghcr.io/OWNER/knowledge-db:latest
```

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
