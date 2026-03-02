## 1. Setup

- [ ] 1.1 Создать go.mod с модулем github.com/strider2038/knowledge-db, добавить зависимости: caarlos0/env, cobra, muonsoft/clog, muonsoft/errors, muonsoft/api-testing, pior/runnable, google/uuid, godotenv
- [ ] 1.2 Создать директории cmd/kb-server, cmd/kb-cli, internal/kb, internal/api, internal/ingestion, internal/mcp, internal/ui, web
- [ ] 1.3 Создать Taskfile с задачами: build (web → copy → go build), build-server, build-cli, dev
- [ ] 1.4 Добавить .cursor/skills/knowledge-db/ для agent skill

## 2. internal/kb

- [ ] 2.1 Реализовать валидацию структуры базы (темы 2–3 уровня, узлы с annotation.md, content.md, metadata.json)
- [ ] 2.2 Реализовать чтение дерева тем и списка узлов по пути

## 3. internal/ingestion

- [ ] 3.1 Определить интерфейс Ingester с методами IngestText, IngestURL
- [ ] 3.2 Реализовать заглушку (ошибка "not implemented" или минимальный узел)

## 4. internal/api

- [ ] 4.1 Реализовать роутинг net/http.ServeMux (Go 1.22+)
- [ ] 4.2 Эндпоинт GET /api/nodes/{path} — чтение узла
- [ ] 4.3 Эндпоинт GET /api/tree — дерево тем
- [ ] 4.4 Эндпоинт GET /api/search?q=... — поиск (заглушка)
- [ ] 4.5 Эндпоинт POST /api/ingest — приём текста для ingestion
- [ ] 4.6 Раздача embedded статики для / и /index.html
- [ ] 4.7 API тесты для эндпоинтов

## 5. internal/ui

- [ ] 5.1 Создать internal/ui/embed.go с //go:embed static
- [ ] 5.2 Добавить static/.gitkeep (static заполняется при сборке)

## 6. cmd/kb-server

- [ ] 6.1 main.go: чтение KB_DATA_PATH, TELEGRAM_TOKEN из env
- [ ] 6.2 Запуск HTTP-сервера с API и раздачей статики
- [ ] 6.3 Интеграция Telegram bot (long polling, заглушка ingestion)
- [ ] 6.4 Endpoint /api/mcp (заглушка или каркас MCP)

## 7. cmd/kb-cli

- [ ] 7.1 Cobra: корневая команда kb-cli, подкоманды validate, init
- [ ] 7.2 validate: вызов internal/kb, вывод отчёта
- [ ] 7.3 init: создание .gitignore, копирование skill с подстановкой {{DATA_PATH}}

## 8. web (React + Vite)

- [ ] 8.1 Инициализировать Vite + React в web/
- [ ] 8.2 Настроить VITE_API_URL
- [ ] 8.3 Navbar с пунктами «Добавить», «Поиск»
- [ ] 8.4 Страница «Добавить»: textarea, кнопка «Добавить», вызов POST /api/ingest
- [ ] 8.5 Страница «Поиск»: дерево тем слева, таблица узлов справа
- [ ] 8.6 Просмотр узла: annotation, content, metadata

## 9. Agent skill

- [ ] 9.1 Создать .cursor/skills/knowledge-db/SKILL.md с инструкциями по структуре узла
- [ ] 9.2 Шаблон {{DATA_PATH}} для подстановки пути при init

## 10. Документация

- [ ] 10.1 Обновить README: структура, команды task build, kb-server, kb-cli
- [ ] 10.2 Обновить AGENTS.md под новую структуру (cmd/kb-server, cmd/kb-cli, web)
