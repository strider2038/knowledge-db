# Agents Guide — knowledge-db

Руководство для AI-агентов (Cursor, Claude, и др.) при работе с проектом управления персональной базой знаний.

## Контекст проекта

**knowledge-db** — система управления персональной базой знаний с принципом **оффлайн-первым** и **git-first**. База хранится локально в отдельной директории под git, доступна без интернета и полностью под контролем пользователя.

## Архитектура

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
├── data/            # База знаний (git subtree/submodule, локальная)
└── openspec/        # Спецификации, изменения (OpenSpec workflow)
```

### serverapp (Go)

- **REST API** — CRUD, поиск по ключевым словам, векторный поиск (RAG)
- **Telegram bot** — в том же процессе, long polling
- **MCP server** — Model Context Protocol для подключения чатботов (Claude, и др.)

### web (React)

- Упрощённый UI для работы с базой
- Работа с локально запущенным kb-server

### Agent skills

- Навыки для работы с базой напрямую из IDE (Cursor, VSCode)
- Локальный доступ к данным без веб-интерфейса

## Принципы для AI-агентов

1. **База — первый класс**: Хранится в `data/` (или аналогичной директории), под git. Не в БД по умолчанию — это markdown/JSON/YAML и т.п. файлы.

2. **Оффлайн-first**: Система должна работать без интернета. Векторные эмбеддинги — опционально; полнотекстовый/ключевой поиск — обязательно.

3. **Git как источник правды**: Версионирование, diff, merge — ключевые инструменты. Избегать форматов, которые сложно мержить.

4. **Локальность**: kb-server и web рассчитаны на localhost. Удалённый доступ — отдельная опция, не основной сценарий.

5. **Язык артефактов**: Proposal, design, tasks, specs — на русском. Код — по конвенции проекта (часто английский для идентификаторов).

## Расположение кода

| Компонент | Путь | Технологии |
|-----------|------|------------|
| Сервер, API | `cmd/kb-server`, `internal/api` | Go, stdlib net/http |
| Telegram bot | `cmd/kb-server` (в том же процессе) | Go, long polling |
| MCP server | `internal/mcp` | Go, endpoint /api/mcp |
| Web UI | `web/` | React, Vite |
| Agent skills | `.cursor/skills/` | Markdown, SKILL.md |
| База знаний | `data/` или отдельный репо | Markdown, frontmatter |

## Когда агент работает с базой

- Читать/писать в `data/` вручную — допустимо, если это часть flow (например, skill)
- Структура записей — согласована со спецификацией (tags, frontmatter, связи)
- При добавлении новых полей или форматов — обновлять спеки и документацию

## OpenSpec

Проект использует OpenSpec для изменений. Спеки и артефакты в `openspec/`. Правила артефактов — в `openspec/config.yaml`.

## Полезные команды

```bash
# Запуск kb-server
go run ./cmd/kb-server
# или: KB_DATA_PATH=/path/to/data ./kb-server
# без git (коммиты и sync отключены): KB_GIT_DISABLED=true go run ./cmd/kb-server

# CLI: валидация
./kb-cli validate --path /path/to/data

# CLI: инициализация
./kb-cli init --path /path/to/data

# Сборка
task build

# Web UI (dev)
cd web && npm run dev

# Открыть change
openspec status --change <name>
```
