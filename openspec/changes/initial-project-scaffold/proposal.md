## Why

Нужна рабочая основа проекта knowledge-db — scaffolding всех компонентов и единый формат хранения, чтобы можно было развивать систему поэтапно. Сейчас есть только описания (README, AGENTS.md), но нет исполняемой структуры кода.

Подробная концепция — в [concept.md](./concept.md).

## What Changes

- Создание директорий по golang-standards: `cmd/kb-server`, `cmd/kb-cli`, `internal/`, `web/`, `.cursor/skills/`
- Формат хранения: темы/подтемы (2–3 уровня), узлы = папки с `annotation.md`, `content.md`, `notes/`, `images/`, `metadata.json`, `.local/` (gitignore)
- Каркас REST API (Go) с минимальными эндпоинтами, stdlib net/http
- Telegram бот в том же процессе, что и kb-server — приём сообщений и URL, pipeline создания узлов
- MCP на `/api/mcp` в том же сервере
- Веб-приложение (React + Vite) embedded в kb-server
- Agent skill + инициализация через `kb-cli init`
- Консольная утилита `kb-cli`: валидация (`kb-cli validate`), инициализация базы (`kb-cli init`)
- Pipeline заполнения: текст/URL → LLM → создание файлов узла (в scaffold — заглушка)
- Taskfile, `go.mod`, `package.json`

## Capabilities

### New Capabilities

- `knowledge-storage`: структура хранения (темы 2–3 уровня, узел = папка с annotation, content, notes, images, metadata, .local)
- `ingestion-pipeline`: добавление записей (текст, URL) — LLM, парсинг, создание узлов
- `rest-api`: REST API для CRUD, поиска (полнотекст, ключевые слова, векторный — опционально)
- `telegram-bot`: приём сообщений/ссылок, вызов ingestion pipeline
- `mcp-server`: MCP-сервер для подключения чатботов
- `webapp`: веб-интерфейс для просмотра, поиска, создания и редактирования
- `agent-skills`: skill для локальной работы с базой из IDE
- `kb-cli`: консольная утилита — валидация структуры базы, инициализация (init: .gitignore, agent skills)

### Modified Capabilities

(пусто — существующих спеки отсутствуют)

## Impact

- Новый код: `cmd/kb-server`, `cmd/kb-cli`, `internal/`, `web/`, `.cursor/skills/`
- Зависимости: Go-модули (роутер, MCP SDK, Telegram API), npm (React, Vite), LLM-API для ingestion
- Затрагиваемые системы: только новый проект, внешних интеграций нет
