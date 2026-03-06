# Personal knowledge database

Система управления персональной базой знаний с принципом **оффлайн-first** и **git-first**.

База хранится локально в отдельной директории под git — знания всегда доступны на текущей машине без интернета, версионируются и удобно мержатся.

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
├── openspec/        # Спецификации (OpenSpec)
├── AGENTS.md        # Руководство для AI-агентов
└── README.md
```

## Быстрый старт

```bash
# Сборка
task build

# Запуск сервера (KB_DATA_PATH обязателен)
KB_DATA_PATH=/path/to/data ./kb-server

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
| `task test` | Запустить тесты |
| `task lint` | golangci-lint |

## Конфигурация

- **KB_DATA_PATH** — путь к корню базы знаний (обязателен для kb-server)
- **TELEGRAM_TOKEN** — токен Telegram-бота (опционально)
- **KB_HTTP_ADDR** — адрес HTTP-сервера (по умолчанию :8080)
- **VITE_API_URL** — URL API для web (по умолчанию http://localhost:8080)

## Лицензия

MIT © 2026 Igor Lazarev
