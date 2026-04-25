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


| Команда             | Описание                           |
| ------------------- | ---------------------------------- |
| `task build`        | Собрать web + kb-server + kb-cli   |
| `task build-server` | Собрать только kb-server           |
| `task build-cli`    | Собрать только kb-cli              |
| `task web:dev`      | Vite dev server (HMR, прокси /api) |
| `task server:dev`   | kb-server с hot reload (air)       |
| `task dev`          | Подсказка по запуску dev-окружения |
| `task test`         | Запустить тесты                    |
| `task lint`         | golangci-lint                      |
| `task lint:fix`     | golangci-lint с автоисправлением   |


## Разработка

Для разработки запустите в двух терминалах:

1. `task web:dev` — Vite dev server ([http://localhost:5173](http://localhost:5173))
2. `task server:dev` — kb-server с hot reload

Для `server:dev` нужен [air](https://github.com/air-verse/air): `task server:dev:install`.

## Конфигурация


| Переменная                                                       | Описание                                                                                                                                                                                        |
| ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **KB_DATA_PATH**                                                 | Путь к корню базы знаний (обязателен для kb-server)                                                                                                                                             |
| **KB_HTTP_ADDR**                                                 | Адрес HTTP-сервера (по умолчанию :8080)                                                                                                                                                         |
| **KB_GIT_DISABLED**                                              | Отключить git (коммиты и sync)                                                                                                                                                                  |
| **KB_LOGIN**, **KB_PASSWORD**                                    | Парольный режим: при задании **обоих** включается защита API и web UI (нельзя совмещать с Google OAuth)                                                                                         |
| **KB_GOOGLE_OAUTH_CLIENT_ID**, **KB_GOOGLE_OAUTH_CLIENT_SECRET** | Google OAuth: идентификатор и секрет OAuth 2.0-клиента (тип **Web application**) в Google Cloud Console                                                                                         |
| **KB_GOOGLE_OAUTH_REDIRECT_URL**                                 | Google OAuth: **точный** URL обратного вызова, как в Console — обычно `https://<хост API>/api/auth/google/callback`                                                                             |
| **KB_OAUTH_STATE_SECRET**                                        | Google OAuth: секрет для подписи параметра `state` (CSRF); сгенерируйте длинную случайную строку                                                                                                |
| **KB_AUTH_ALLOWED_EMAILS**                                       | Google OAuth: список разрешённых email через запятую; чужой Google-аккаунт не получит сессию                                                                                                    |
| **KB_SESSION_TTL**                                               | TTL сессии (по умолчанию 8h)                                                                                                                                                                    |
| **TELEGRAM_TOKEN**                                               | Токен Telegram-бота (опционально)                                                                                                                                                               |
| **TELEGRAM_OWNER_ID**                                            | Telegram user ID владельца (обязателен при TELEGRAM_TOKEN)                                                                                                                                      |
| **KB_PUBLIC_WEB_BASE_URL**                                       | Публичный URL веб-интерфейса без завершающего `/` (например `https://kb.example`); **обязателен в режиме Google OAuth** (редирект после входа в SPA), в ответе бота — ссылка «Открыть на сайте» |
| **LLM_API_URL**, **LLM_API_KEY**, **LLM_MODEL**                  | LLM для ingestion (OpenAI-совместимый API)                                                                                                                                                      |
| **JINA_API_KEY**                                                 | Ключ Jina для эмбеддингов (опционально)                                                                                                                                                         |
| **GIT_SYNC_INTERVAL**                                            | Интервал git sync (по умолчанию 5m)                                                                                                                                                             |
| **VITE_API_URL**                                                 | URL API для web (по умолчанию [http://localhost:8080](http://localhost:8080))                                                                                                                   |
| **ALLOWED_CORS_ORIGIN**                                          | CORS origin для dev (например [http://localhost:5173](http://localhost:5173))                                                                                                                   |


## Режимы запуска

**Открытый режим** (по умолчанию): без `KB_LOGIN`/`KB_PASSWORD` API и web UI доступны без авторизации.

```bash
KB_DATA_PATH=/path/to/data ./kb-server
```

**Парольный режим** (`KB_LOGIN` + `KB_PASSWORD`): вход через форму на `/login`. **Не задавайте** одновременно полный набор переменных Google OAuth — сервер не запустится.

```bash
KB_DATA_PATH=/path/to/data KB_LOGIN=admin KB_PASSWORD=secret ./kb-server
```

**Режим Google OAuth** — взаимоисключающий с паролем. Нужны **все** перечисленные в таблице переменные: клиент, redirect, секрет `state`, непустой allowlist, публичный URL SPA; `KB_LOGIN` и `KB_PASSWORD` должны быть **пустыми**.

1. В [Google Cloud Console](https://console.cloud.google.com/) создайте проект (или выберите существующий), настройте **OAuth consent screen** (для теста — External и тестовые пользователи), затем **APIs & Services → Credentials → Create Credentials → OAuth client ID** и тип **Web application**.
2. В **Authorized redirect URIs** укажите **ровно** тот же URL, что и в `KB_GOOGLE_OAUTH_REDIRECT_URL` — путь фиксирован: `/api/auth/google/callback` на том хосте и схеме, где снаружи доступен `kb-server`. Пример: `https://api.example.com/api/auth/google/callback` или для локальной проверки: `http://localhost:8080/api/auth/google/callback` (в test mode Google допускает localhost).
3. Скопируйте **Client ID** и **Client secret** в `KB_GOOGLE_OAUTH_CLIENT_ID` и `KB_GOOGLE_OAUTH_CLIENT_SECRET`. Сгенерируйте криптостойкую строку для `KB_OAUTH_STATE_SECRET` (например `openssl rand -hex 32`). В `KB_AUTH_ALLOWED_EMAILS` перечислите email пользователей; вход возможен только для **подтверждённого** в Google email.

```bash
export KB_DATA_PATH=/path/to/data
export KB_PUBLIC_WEB_BASE_URL=https://kb.example
export KB_GOOGLE_OAUTH_CLIENT_ID=....apps.googleusercontent.com
export KB_GOOGLE_OAUTH_CLIENT_SECRET=...
export KB_GOOGLE_OAUTH_REDIRECT_URL=https://api.example.com/api/auth/google/callback
export KB_OAUTH_STATE_SECRET=$(openssl rand -hex 32)
export KB_AUTH_ALLOWED_EMAILS=you@example.com,colleague@example.com
./kb-server
```

Если задана **хотя бы одна** переменная Google OAuth, пустой набор остальных не допускается: либо полная конфигурация, либо очистите все `KB_GOOGLE_`*, `KB_OAUTH_STATE_SECRET` и `KB_AUTH_ALLOWED_EMAILS`.

UI и API должны согласовываться по CORS: для production укажите `ALLOWED_CORS_ORIGIN` (origin веб-интерфейса, без пути). Вход: кнопка «Войти через Google» ведёт на `GET /api/auth/google` на том же origin, что и API, либо настраивайте прокси так, чтобы этот путь попадал на `kb-server`.

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