## 1. Setup

- [x] 1.1 Создать go.mod с модулем github.com/strider2038/knowledge-db, добавить Go-стек из design (Decision 2): caarlos0/env, cobra, muonsoft/clog, muonsoft/errors, muonsoft/api-testing, pior/runnable, google/uuid, godotenv
- [x] 1.2 Создать директории cmd/kb-server, cmd/kb-cli, internal/kb, internal/api, internal/ingestion, internal/mcp, internal/ui, web
- [x] 1.3 Создать Taskfile с задачами: build (web → copy → go build), build-server, build-cli, dev
- [x] 1.4 Добавить .cursor/skills/knowledge-db/ для agent skill
- [x] 1.5 При реализации Go-кода следовать skills: kb-backend-golang, golang-errors, golang-logging, golang-tests; правила: agent-finish-tests-lint, golang-errors-wrap

## 2. internal/kb

- [x] 2.1 Реализовать валидацию структуры базы (темы 2–3 уровня, узлы с annotation.md, content.md, metadata.json)
- [x] 2.2 Реализовать чтение дерева тем и списка узлов по пути

## 3. internal/ingestion

- [x] 3.1 Определить интерфейс Ingester с методами IngestText, IngestURL
- [x] 3.2 Реализовать заглушку (ошибка "not implemented" или минимальный узел)

## 4. internal/api

- [x] 4.1 Реализовать роутинг net/http.ServeMux (Go 1.22+), конфиг через caarlos0/env
- [x] 4.2 Эндпоинт GET /api/nodes/{path} — чтение узла
- [x] 4.3 Эндпоинт GET /api/tree — дерево тем
- [x] 4.4 Эндпоинт GET /api/search?q=... — поиск (заглушка)
- [x] 4.5 Эндпоинт POST /api/ingest — приём текста для ingestion
- [x] 4.6 Раздача embedded статики для / и /index.html
- [x] 4.7 API тесты для эндпоинтов (muonsoft/api-testing, skill golang-tests)

## 5. internal/ui

- [x] 5.1 Создать internal/ui/embed.go с //go:embed static
- [x] 5.2 Добавить static/.gitkeep (static заполняется при сборке)

## 6. cmd/kb-server

- [x] 6.1 main.go: чтение KB_DATA_PATH, TELEGRAM_TOKEN через caarlos0/env
- [x] 6.2 Запуск HTTP-сервера с API и раздачей статики (pior/runnable для graceful shutdown)
- [x] 6.3 Интеграция Telegram bot (long polling, runnable, skill runnable-background-processes)
- [x] 6.4 Endpoint /api/mcp (заглушка или каркас MCP)

## 7. cmd/kb-cli

- [x] 7.1 Cobra (spf13/cobra): корневая команда kb-cli, подкоманды validate, init
- [x] 7.2 validate: вызов internal/kb, вывод отчёта
- [x] 7.3 init: создание .gitignore, копирование skill с подстановкой {{DATA_PATH}}

## 8. web (React + Vite)

- [x] 8.1 Инициализировать Vite + React в web/ (skill web-frontend)
- [x] 8.2 Настроить VITE_API_URL
- [x] 8.3 Navbar с пунктами «Добавить», «Поиск»
- [x] 8.4 Страница «Добавить»: textarea, кнопка «Добавить», вызов POST /api/ingest
- [x] 8.5 Страница «Поиск»: дерево тем слева, таблица узлов справа
- [x] 8.6 Просмотр узла: annotation, content, metadata

## 9. Agent skill

- [x] 9.1 Создать .cursor/skills/knowledge-db/SKILL.md с инструкциями по структуре узла
- [x] 9.2 Шаблон {{DATA_PATH}} для подстановки пути при init

## 10. Документация

- [x] 10.1 Обновить README: структура, команды task build, kb-server, kb-cli
- [x] 10.2 Обновить AGENTS.md под новую структуру (cmd/kb-server, cmd/kb-cli, web)
