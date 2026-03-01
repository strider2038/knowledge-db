# Agents Guide — knowledge-db

Руководство для AI-агентов (Cursor, Claude, и др.) при работе с проектом управления персональной базой знаний.

## Контекст проекта

**knowledge-db** — система управления персональной базой знаний с принципом **оффлайн-первым** и **git-first**. База хранится локально в отдельной директории под git, доступна без интернета и полностью под контролем пользователя.

## Архитектура

```
knowledge-db/
├── serverapp/      # Серверная часть (Go)
├── webapp/         # Web UI (React)
├── .cursor/skills/ # Agent skills для Cursor/Claude
├── data/           # База знаний (git subtree/submodule, локальная)
└── openspec/       # Спецификации, изменения (OpenSpec workflow)
```

### serverapp (Go)

- **REST API** — CRUD, поиск по ключевым словам, векторный поиск (RAG)
- **Telegram bot** — отдельный процесс/порт, доступ к базе через Telegram
- **MCP server** — Model Context Protocol для подключения чатботов (Claude, и др.)

### webapp (React)

- Упрощённый UI для работы с базой
- Работа с локально запущенным serverapp

### Agent skills

- Навыки для работы с базой напрямую из IDE (Cursor, VSCode)
- Локальный доступ к данным без веб-интерфейса

## Принципы для AI-агентов

1. **База — первый класс**: Хранится в `data/` (или аналогичной директории), под git. Не в БД по умолчанию — это markdown/JSON/YAML и т.п. файлы.

2. **Оффлайн-first**: Система должна работать без интернета. Векторные эмбеддинги — опционально; полнотекстовый/ключевой поиск — обязательно.

3. **Git как источник правды**: Версионирование, diff, merge — ключевые инструменты. Избегать форматов, которые сложно мержить.

4. **Локальность**: serverapp и webapp рассчитаны на localhost. Удалённый доступ — отдельная опция, не основной сценарий.

5. **Язык артефактов**: Proposal, design, tasks, specs — на русском. Код — по конвенции проекта (часто английский для идентификаторов).

## Расположение кода

| Компонент       | Путь                   | Технологии                    |
|-----------------|------------------------|-------------------------------|
| Сервер, API     | `serverapp/`           | Go, stdlib, возможно chi/fiber |
| Telegram bot    | `serverapp/cmd/bot/`   | Go, telegram API              |
| MCP server      | `serverapp/internal/mcp/` | Go, MCP SDK                |
| Web UI          | `webapp/`              | React, Vite                   |
| Agent skills    | `.cursor/skills/`      | Markdown, SKILL.md            |
| База знаний     | `data/` или отдельный репо | Markdown, frontmatter   |

## Когда агент работает с базой

- Читать/писать в `data/` вручную — допустимо, если это часть flow (например, skill)
- Структура записей — согласована со спецификацией (tags, frontmatter, связи)
- При добавлении новых полей или форматов — обновлять спеки и документацию

## OpenSpec

Проект использует OpenSpec для изменений. Спеки и артефакты в `openspec/`. Правила артефактов — в `openspec/config.yaml`.

## Полезные команды

```bash
# Запуск serverapp (пример)
go run ./serverapp/cmd/server

# Запуск Telegram bot
go run ./serverapp/cmd/bot

# Запуск webapp
cd webapp && npm run dev

# Открыть change
openspec status --change <name>
```
