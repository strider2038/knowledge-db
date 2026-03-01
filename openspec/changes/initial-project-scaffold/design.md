# Design: initial-project-scaffold

## Context

Проект knowledge-db — система управления персональной базой знаний. Принципы: оффлайн-first, git-first. Сейчас есть только документация (README, AGENTS.md, concept.md, proposal), исполняемого кода нет. Change создаёт scaffolding монолитного приложения на Go по [Golang Project Layout](https://github.com/golang-standards/project-layout).

## Goals / Non-Goals

**Goals:**

- Структура репозитория по golang-standards и каркасы компонентов
- Монолит: серверная часть (API + web UI + Telegram bot) и консольная (kb)
- Формат хранения с валидацией через kb-cli
- Agent skill + установка через kb install-skills

**Non-Goals:**

- Полноценная реализация ingestion (LLM, HTML→markdown) — заглушка/интерфейс
- Векторный поиск — опционально, не в scaffold
- Видео/транскрипция — будущее

## Decisions

### 1. Структура репозитория (golang-standards, monolith)

**Решение:** Один репозиторий, две части приложения:

- **Серверное** — API, web UI (embedded), Telegram bot (отдельный порт в том же процессе). Один бинарник `cmd/server`.
- **Консольное** — валидация структуры, установка skills. Бинарник `cmd/kb`.

Структура по [project-layout](https://github.com/golang-standards/project-layout):

```
/
├── cmd/
│   ├── server/      # API + UI + bot + MCP
│   └── kb/          # validate, install-skills
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

Директория базы знаний **вне проекта**, путь задаётся переменной окружения `KB_DATA_PATH`.

### 2. HTTP-роутинг

**Решение:** Стандартный `net/http.ServeMux` (Go 1.22+): method matching (`GET /posts/{id}`), wildcards.

### 3. Embed статики (web UI)

Ограничение: `//go:embed` не допускает `..` в путях; embed задаётся относительно файла с директивой.

**Варианты:**

| Вариант | Описание | Плюсы | Минусы |
|---------|----------|-------|--------|
| **A. main.go в корне** | `main.go` и `web/dist` в корне, embed там же | Просто | Нарушает project layout (нет cmd/) |
| **B. Copy в build** | Makefile/скрипт копирует `web/dist` → `internal/ui/static/` перед `go build` | Соответствует layout, cmd/server чистый | Два шага сборки |
| **C. Симлинк** | `internal/ui/static` — симлинк на `web/dist` | Один `go build` после сборки web | Симлинки в git, кроссплатформенность |
| **D. web внутри internal** | `internal/web/` — React-проект, `internal/web/dist` | Всё под рукой | Смешение Go и Node в internal |

**Выбор: B (copy в build).** Сохраняем layout, явный pipeline: `make build` или `npm run build && cp -r web/dist internal/ui/static && go build ./cmd/server`.

Структура: `internal/ui/embed.go` с `//go:embed static`, `static/` заполняется при сборке. `web/` — исходники React.

### 4. Layout cmd/kb

**Решение:** Cobra, подкоманды `validate`, `install-skills`. Путь к data — флаг `--path` или `KB_DATA_PATH`. Общая логика — `internal/kb`.

### 5. Хранение: путь к базе знаний

**Решение:** Директория базы может находиться вне проекта. Путь задаётся переменной окружения **`KB_DATA_PATH`**. В `data/` в репо — только пример структуры или `.gitkeep`; рабочая база — снаружи.

`.gitignore` в корне базы (если она под git): `**/.local/`, `**/.local/**`.

### 6. Agent skill: установка

**Решение:** `kb install-skills --path <path>` копирует skill из `.cursor/skills/knowledge-db/` в `~/.cursor/skills/` и подставляет путь к data в SKILL.md (шаблон `{{DATA_PATH}}`). Skill читает путь из встроенного параметра.

### 7. Ingestion pipeline (scaffold)

**Решение:** Интерфейс `Ingester` с методами `IngestText(text)`, `IngestURL(url)`. Реализация-заглушка возвращает ошибку "not implemented" или создаёт минимальный узел без LLM. LLM-интеграция — в последующих change.

### 8. Telegram bot: в том же процессе, отдельный порт

**Решение:** Бот слушает отдельный порт в рамках `cmd/server` (отдельный http.Server или long-polling горутина). Конфиг через env: `TELEGRAM_TOKEN`, `KB_DATA_PATH`. Long polling — без публичного URL.

## Risks / Trade-offs

| Риск | Митигация |
|------|-----------|
| Дублирование логики kb между server и kb | Общий пакет `internal/kb`; cmd/server и cmd/kb импортируют |
| LLM API — затраты и латентность | В scaffold не используем; интерфейс позволяет подставить mock |
| CORS при не-localhost | В scaffold разрешаем только localhost |
| Skill перезаписывается при обновлении | install-skills — идемпотентно; документировать, что локальные правки skill будут потеряны |

## Migration Plan

Новый проект — миграции нет. Шаги:

1. `go mod init`, создать директории по project layout
2. `cd web && npm install && npm run build`
3. `cp -r web/dist internal/ui/static` (или через Makefile)
4. `go build ./cmd/server` и `go build ./cmd/kb`
5. Задать `KB_DATA_PATH`, запустить server
6. `kb validate $KB_DATA_PATH`, `kb install-skills --path $KB_DATA_PATH`

## Open Questions

- MCP в том же процессе, что и API, или отдельный бинарник — для scaffold достаточно встроенного в server.
