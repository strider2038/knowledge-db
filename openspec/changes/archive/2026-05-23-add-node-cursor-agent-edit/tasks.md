## 1. Backend: общий cursor runner

- [x] 1.1 Вынести запуск `cursor-agent`, чтение stdout/stderr и `parseCursorLogEvents` в общий модуль (используется normalize и agent-edit)
- [x] 1.2 Обновить `cursorNodeNormalizer` на использование общего runner без изменения поведения нормализации

## 2. Backend: операция agent-edit

- [x] 2.1 Добавить job type `node_agent_edit`, структуры операции/логов и метаданные (`edit_ok`, `sync_done`)
- [x] 2.2 Реализовать сборку промта: ограничения + `instruction` + контекст узла
- [x] 2.3 Реализовать `POST /api/nodes/{path}/agent-edit` с валидацией `instruction` (непустая, max length)
- [x] 2.4 Реализовать `GET /api/node-agent-edit/{id}` и `GET /api/node-agent-edit/{id}/logs`
- [x] 2.5 Реализовать фоновый job: edit → sync, логирование стадий, блокировка параллельного запуска на path
- [x] 2.6 Зарегистрировать маршруты в router/bootstrap
- [x] 2.7 API-тесты: старт, 400/404, статус, логи `after`, конфликт running, недоступный cursor-agent (mock/skip по принятому в проекте паттерну)

## 3. Frontend: API client

- [x] 3.1 Добавить типы и функции `startNodeAgentEdit`, `getNodeAgentEditStatus`, `getNodeAgentEditLogs` в `web/src/services/api.ts`
- [x] 3.2 Unit-тесты API client для новых URL и query `after`

## 4. Frontend: UI на странице узла

- [x] 4.1 Добавить кнопку «Редактировать с агентом» в `NodeActionBar` (только оригинальные узлы)
- [x] 4.2 Добавить модалку с textarea, валидацией и запуском операции
- [x] 4.2a Сохранение/восстановление последней инструкции в `sessionStorage` per `node_path` (после успешного старта; подстановка при открытии модалки)
- [x] 4.3 Добавить polling статуса/логов и нижнюю панель логов (переиспользовать паттерн нормализации)
- [x] 4.4 После success обновить узел на странице (`getNode` / `onNodeChanged`)
- [x] 4.5 Тесты: модалка (пустая инструкция disabled), запуск, отображение логов, ошибка операции

## 5. Документация и проверка

- [x] 5.1 При необходимости обновить README/.env.example (если добавлены env; иначе пропустить)
- [x] 5.2 `go test ./... -race`, `golangci-lint run`, `npm run test` и `npm run build` в web
- [x] 5.3 `openspec validate add-node-cursor-agent-edit`
