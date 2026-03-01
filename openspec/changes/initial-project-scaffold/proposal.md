## Why

Нужна рабочая основа проекта knowledge-db — scaffolding всех компонентов и единый формат хранения, чтобы можно было развивать систему поэтапно. Сейчас есть только описания (README, AGENTS.md), но нет исполняемой структуры кода.

Подробная концепция — в [concept.md](./concept.md).

## What Changes

- Создание директорий и базовой структуры `serverapp/`, `webapp/`, `data/`, agent skills
- Формат хранения: темы/подтемы (2–3 уровня), узлы = папки с `annotation.md`, `content.md`, `notes/`, `images/`, `metadata.json`, `.local/` (gitignore)
- Каркас REST API (Go) с минимальными эндпоинтами
- Каркас Telegram бота (отдельный бинарник) — приём сообщений и URL, pipeline создания узлов
- Каркас MCP-сервера для интеграции с чатботами
- Каркас веб-приложения (React + Vite) — создание и редактирование записей
- Agent skill + установка через консольную утилиту
- Консольная утилита `kb` (Go): валидация структуры базы (`kb validate`), установка agent skills (`kb install-skills`)
- Pipeline заполнения: текст/URL → LLM → создание файлов узла (URL: fetch, HTML→markdown, метаданные)
- `go.mod`, `package.json`, скрипты запуска

## Capabilities

### New Capabilities

- `knowledge-storage`: структура хранения (темы 2–3 уровня, узел = папка с annotation, content, notes, images, metadata, .local)
- `ingestion-pipeline`: добавление записей (текст, URL) — LLM, парсинг, создание узлов
- `rest-api`: REST API для CRUD, поиска (полнотекст, ключевые слова, векторный — опционально)
- `telegram-bot`: приём сообщений/ссылок, вызов ingestion pipeline
- `mcp-server`: MCP-сервер для подключения чатботов
- `webapp`: веб-интерфейс для просмотра, поиска, создания и редактирования
- `agent-skills`: skill для локальной работы с базой из IDE
- `kb-cli`: консольная утилита — валидация структуры базы, установка agent skills

### Modified Capabilities

(пусто — существующих спеки отсутствуют)

## Impact

- Новый код: `serverapp/`, `webapp/`, `data/`, `.cursor/skills/`
- Зависимости: Go-модули (роутер, MCP SDK, Telegram API), npm (React, Vite), LLM-API для ingestion
- Затрагиваемые системы: только новый проект, внешних интеграций нет
