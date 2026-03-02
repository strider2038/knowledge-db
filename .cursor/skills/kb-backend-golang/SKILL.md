---
name: kb-backend-golang
description: Правила и лучшие практики для Go-кода backend (cmd/kb-server, cmd/kb-cli, internal/). Используй при работе с internal/**/*.go, cmd/**/*.go.
---

# Go в knowledge-db — стиль и практики

Этот skill применяется при работе с Go-кодом в `internal/`, `cmd/kb-server`, `cmd/kb-cli`.  
Цель — идиоматичный, модульный, тестируемый код.

## Роль

- Идиоматичный Go-код
- Чёткое разделение слоёв (API handlers, internal/kb, internal/ingestion)
- Тестируемость через интерфейсы

## Архитектура

- **internal/api** — HTTP handlers, роутинг
- **internal/kb** — работа с data/, валидация структуры
- **internal/ingestion** — интерфейс Ingester, pipeline
- **internal/mcp** — MCP endpoint
- **internal/ui** — embed статики

Предпочитать **интерфейсы** вместо жёстких зависимостей. Публичные функции принимают интерфейсы, где уместно.

## Порядок объявлений в файле

- Публичные типы и методы — в начале
- Приватные методы и функции — в конце

## Импорты

- Группы: stdlib → внешние → internal (github.com/strider2038/knowledge-db/...)
- Сортировка по алфавиту внутри группы

## Линтинг

- `golangci-lint`, конфиг `.golangci.yml` в корне

## Стиль кода

- Именование: функции — глаголы, переменные — существительные
- Ранние возвраты, глубина вложенности не более 3
- Ошибки: `github.com/muonsoft/errors`, оборачивать через `errors.Errorf("action: %w", err)`
- Контекст: I/O-операции принимают `context.Context` первым аргументом

## Работа с ошибками

- Sentinel-ошибки: `var ErrNodeNotFound = errors.New("node not found")`
- Каждая возвращаемая ошибка оборачивается через `errors.Errorf` или `errors.Wrap`
- Не подавлять ошибки (`_ = err` запрещён)

## Логирование (muonsoft/clog)

- Логгер из контекста: `clog.FromContext(ctx)`
- Не создавать `slog.Logger` в бизнес-коде
- Error-уровень: `clog.Errorf(ctx, "msg: %w", err)` — не `slog.String("error", ...)`

## Хранение данных

- База знаний — файловая система (markdown, JSON)
- Путь к базе — `KB_DATA_PATH` (env)
- Нет СУБД, нет миграций

## Чек-лист перед коммитом

- [ ] gofmt / goimports
- [ ] golangci-lint проходит
- [ ] Тесты для затронутой логики
- [ ] Нет временных логов
