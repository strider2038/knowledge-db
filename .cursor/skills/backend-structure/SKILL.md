---
name: backend-structure
description: Структура backend knowledge-db (cmd/, internal/). Используй при добавлении handlers, пакетов internal/, работе с kb и ingestion.
---

# Структура backend knowledge-db

## Расположение кода

```
/
├── cmd/
│   ├── kb-server/   # API + UI + Telegram bot + MCP
│   └── kb-cli/      # validate, init
├── internal/
│   ├── kb/          # работа с data/, валидация, дерево тем
│   ├── api/         # HTTP handlers, роутинг
│   ├── ingestion/   # интерфейс Ingester, pipeline
│   ├── mcp/          # MCP endpoint /api/mcp
│   └── ui/          # embed статики (embed.go, static/)
├── web/             # React исходники
└── .cursor/skills/  # agent skills
```

## internal/kb

- Валидация структуры базы (темы 2–3 уровня, узлы с {dirname}.md и frontmatter)
- Чтение дерева тем, списка узлов
- Путь к базе — `KB_DATA_PATH` (env)

## internal/api

- Роутинг: `net/http.ServeMux` (Go 1.22+)
- Эндпоинты: GET /api/nodes/{path}, GET /api/tree, GET /api/search, POST /api/ingest
- Раздача embedded статики для /
- Маппинг ошибок в HTTP-статусы

## internal/ingestion

- Интерфейс `Ingester`: `IngestText(text)`, `IngestURL(url)`
- LLM-оркестратор в `internal/ingestion/llm` — использует **OpenAI Responses API** (не Chat Completions)

## internal/mcp

- Endpoint `/api/mcp` на том же сервере
- Протокол MCP для чатботов

## cmd/kb-server

- main: чтение env (KB_DATA_PATH, TELEGRAM_TOKEN)
- HTTP-сервер, Telegram bot (runnable), MCP

## cmd/kb-cli

- Cobra: `validate`, `init`
- validate: вызов internal/kb
- init: .gitignore, копирование skills с подстановкой {{DATA_PATH}}

## Хранение

- Нет СУБД — файловая система (markdown, JSON в data/)
- Репозитории — интерфейсы для доступа к данным, реализации работают с файлами
