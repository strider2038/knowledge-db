# Design: initial-project-scaffold

## Context

Проект knowledge-db — система управления персональной базой знаний. Принципы: оффлайн-first, git-first. Сейчас есть только документация (README, AGENTS.md, concept.md, proposal), исполняемого кода нет. Change создаёт scaffolding монолитного приложения на Go по [Golang Project Layout](https://github.com/golang-standards/project-layout).

## Goals / Non-Goals

**Goals:**

- Структура репозитория по golang-standards и каркасы компонентов
- Монолит: серверная часть (API + web UI + Telegram bot) и консольная (kb)
- Формат хранения с валидацией через kb-cli
- Agent skill + инициализация базы через kb-cli init

**Non-Goals:**

- Полноценная реализация ingestion (LLM, HTML→markdown) — заглушка/интерфейс
- Векторный поиск — опционально, не в scaffold
- Видео/транскрипция — будущее

## Decisions

### 1. Структура репозитория (golang-standards, monolith)

**Решение:** Один репозиторий, две части приложения:

- **Серверное (kb-server)** — API, web UI (embedded), Telegram bot (отдельный порт в том же процессе), MCP на `/api/mcp`. Бинарник `cmd/kb-server`.
- **Консольное (kb-cli)** — валидация структуры, инициализация базы. Бинарник `cmd/kb-cli`.

Сборка и задачи — через **Taskfile**.

Структура по [project-layout](https://github.com/golang-standards/project-layout):

```
/
├── cmd/
│   ├── kb-server/   # API + UI + bot + MCP (/api/mcp)
│   └── kb-cli/      # validate, init
├── internal/
│   ├── kb/          # работа с data/, валидация
│   ├── api/         # HTTP handlers
│   ├── ingestion/   # pipeline (интерфейс + заглушка)
│   ├── mcp/
│   └── ui/          # embed статики (см. ниже)
├── web/             # React исходники (Vite)
├── .cursor/skills/
└── go.mod
```

Go module: **`github.com/strider2038/knowledge-db`**

Директория базы знаний **вне проекта**, путь задаётся переменной окружения `KB_DATA_PATH`.

### 2. Go-библиотеки

**Решение:** Использовать проверенный стек (по аналогии с audio-quest):

| Библиотека | Назначение |
|------------|------------|
| `github.com/caarlos0/env/v10` | Конфигурация из env (KB_DATA_PATH, TELEGRAM_TOKEN) — struct-based |
| `github.com/joho/godotenv` | Загрузка .env в dev |
| `github.com/spf13/cobra` | CLI для kb-cli (validate, init) |
| `github.com/muonsoft/clog` | Структурированное логирование |
| `github.com/muonsoft/errors` | Обработка ошибок с контекстом |
| `github.com/muonsoft/api-testing` | API-тесты для эндпоинтов |
| `github.com/pior/runnable` | Graceful shutdown сервера |
| `github.com/google/uuid` | Генерация ID при необходимости |

Для ingestion (будущее): `github.com/openai/openai-go` или аналог. В scaffold не требуется.

При реализации: следовать skills `.cursor/skills/kb-backend-golang`, `golang-errors`, `golang-logging`, `golang-tests` и правилам `.cursor/rules/`.

### 3. HTTP-роутинг

**Решение:** Стандартный `net/http.ServeMux` (Go 1.22+): method matching (`GET /posts/{id}`), wildcards.

### 4. Embed статики (web UI)

Ограничение: `//go:embed` не допускает `..` в путях; embed задаётся относительно файла с директивой.

**Варианты:**

| Вариант | Описание | Плюсы | Минусы |
|---------|----------|-------|--------|
| **A. main.go в корне** | `main.go` и `web/dist` в корне, embed там же | Просто | Нарушает project layout (нет cmd/) |
| **B. Copy в build** | Taskfile копирует `web/dist` → `internal/ui/static/` перед `go build` | Соответствует layout, cmd/kb-server чистый | Два шага сборки |
| **C. Симлинк** | `internal/ui/static` — симлинк на `web/dist` | Один `go build` после сборки web | Симлинки в git, кроссплатформенность |
| **D. web внутри internal** | `internal/web/` — React-проект, `internal/web/dist` | Всё под рукой | Смешение Go и Node в internal |

**Выбор: B (copy в build).** Сохраняем layout, явный pipeline через Taskfile: `task build` (сборка web → копирование → go build).

Структура: `internal/ui/embed.go` с `//go:embed static`, `static/` заполняется при сборке. `web/` — исходники React.

### 5. Layout cmd/kb-cli

**Решение:** Cobra, подкоманды `validate`, `init`. Путь к data — флаг `--path` (по умолчанию текущая директория). Общая логика — `internal/kb`.

**`kb-cli init`** — инициализирует новую базу знаний в указанной (или текущей) директории: создаёт `.gitignore` (`**/.local/`, `**/.local/**`), устанавливает agent skills в `~/.cursor/skills/` с подстановкой пути к базе.

### 6. Хранение: путь к базе знаний

**Решение:** Директория базы может находиться вне проекта. Путь задаётся переменной окружения **`KB_DATA_PATH`**. В `data/` в репо — только пример структуры или `.gitkeep`; рабочая база — снаружи.

`.gitignore` в корне базы (если она под git): `**/.local/`, `**/.local/**`.

### 7. Agent skill: установка (в рамках init)

**Решение:** `kb-cli init --path <path>` создаёт `.gitignore` в корне базы и копирует skill из `.cursor/skills/knowledge-db/` в `~/.cursor/skills/` с подстановкой пути (шаблон `{{DATA_PATH}}`). Skill читает путь из встроенного параметра.

### 8. Ingestion pipeline (scaffold)

**Решение:** Интерфейс `Ingester` с методами `IngestText(text)`, `IngestURL(url)`. Реализация-заглушка возвращает ошибку "not implemented" или создаёт минимальный узел без LLM. LLM-интеграция — в последующих change.

### 9. Telegram bot: в том же процессе, отдельный порт

**Решение:** Бот слушает отдельный порт в рамках `cmd/kb-server` (отдельный http.Server или long-polling горутина). Конфиг через env: `TELEGRAM_TOKEN`, `KB_DATA_PATH`. Long polling — без публичного URL.

## Risks / Trade-offs

| Риск | Митигация |
|------|-----------|
| Дублирование логики kb между kb-server и kb-cli | Общий пакет `internal/kb`; cmd/kb-server и cmd/kb-cli импортируют |
| LLM API — затраты и латентность | В scaffold не используем; интерфейс позволяет подставить mock |
| CORS при не-localhost | В scaffold разрешаем только localhost |
| Skill перезаписывается при обновлении | init — идемпотентно; документировать, что локальные правки skill будут потеряны |

## Migration Plan

Новый проект — миграции нет. Шаги:

1. `go mod init github.com/strider2038/knowledge-db`, создать директории по project layout, Taskfile
2. `task build` — сборка web, копирование в internal/ui/static, go build
3. Задать `KB_DATA_PATH`, запустить kb-server
4. `kb-cli validate $KB_DATA_PATH`, `kb-cli init` (в директории базы) или `kb-cli init --path /path/to/base`

## MCP

MCP на том же сервере, что и API — отдельный endpoint **`/api/mcp`** (SSE/WebSocket или HTTP по протоколу MCP).
