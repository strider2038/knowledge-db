# Personal knowledge database

Система управления персональной базой знаний с принципом **оффлайн-first** и **git-first**.

База хранится локально в отдельной директории под git — знания всегда доступны на текущей машине без интернета, версионируются и удобно мержатся.

## Компоненты

### serverapp (Go)

Серверная часть приложения:

- **REST API** — CRUD, поиск по ключевым словам, векторный поиск для RAG
- **Telegram bot** — отдельный порт/процесс, доступ к базе через Telegram
- **MCP server** — Model Context Protocol для подключения чатботов (Claude, Cursor и др.)

### webapp (React)

Веб-интерфейс для работы с базой: просмотр, поиск, редактирование записей. Работает с локально запущенным serverapp.

### Agent skills

Навыки для Cursor, Claude и других IDE — локальная работа с базой напрямую из редактора, без веб-интерфейса.

### База знаний (data/)

Хранится в отдельной директории под git (subtree/submodule или отдельный репозиторий). Формат — markdown с frontmatter или аналогичный, удобный для версионирования.

## Ключевые принципы

1. **Git-first** — база под контролем git, diff и merge как основные инструменты
2. **Оффлайн** — полнотекстовый поиск работает без сети; векторный поиск — опционально
3. **Локальность** — serverapp и webapp рассчитаны на localhost; удалённый доступ — опция
4. **Ручной режим** — можно править файлы вручную в IDE или через веб-интерфейс

## Структура проекта

```
knowledge-db/
├── serverapp/         # Go: API, бот, MCP
│   ├── cmd/server/    # REST API
│   ├── cmd/bot/       # Telegram bot
│   └── internal/      # Бизнес-логика, MCP
├── webapp/            # React UI
├── data/              # База знаний (git)
├── .cursor/skills/    # Agent skills
├── openspec/          # Спецификации (OpenSpec)
├── AGENTS.md          # Руководство для AI-агентов
└── README.md
```

## Быстрый старт

```bash
# API
go run ./serverapp/cmd/server

# Telegram bot
go run ./serverapp/cmd/bot

# Web UI
cd webapp && npm run dev
```

## Лицензия

MIT © 2026 Igor Lazarev
