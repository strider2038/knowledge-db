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
| **TELEGRAM_TOKEN** | Токен Telegram-бота (опционально) |
| **TELEGRAM_OWNER_ID** | Telegram user ID владельца (обязателен при TELEGRAM_TOKEN) |
| **LLM_API_URL**, **LLM_API_KEY**, **LLM_MODEL** | LLM для ingestion (OpenAI-совместимый API) |
| **JINA_API_KEY** | Ключ Jina для эмбеддингов (опционально) |
| **GIT_SYNC_INTERVAL** | Интервал git sync (по умолчанию 5m) |
| **VITE_API_URL** | URL API для web (по умолчанию http://localhost:8080) |
| **ALLOWED_CORS_ORIGIN** | CORS origin для dev (например http://localhost:5173) |

## Лицензия

MIT © 2026 Igor Lazarev
