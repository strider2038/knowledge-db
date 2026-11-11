# QA Plan

## Risk and Priority

| Risk / behavior | Evidence required | P0/P1 |
|-----------------|-------------------|-------|
| По умолчанию остаётся Cursor, текст 503 прежний | API-тест без `cursor-agent` в PATH | P0 |
| `opencode` без бинаря даёт 503 и не создаёт джобу | API-тест с раннером OpenCode | P0 |
| Невалидный `KB_NODE_AGENT`, пустая модель, Go без ключа, Go вместе с URL | unit-тесты `ValidateNodeAgent` | P0 |
| JSONL `error` не считается успехом; `tool_use` логируется как имя | unit-тест парсера | P0 |
| Конфиг OpenCode запрещает shell и не пишет файл в базу | unit-тест собранного argv/env/JSON | P0 |
| Модалка не содержит «Cursor» | Vitest `NodeAgentEditDialog` | P0 |
| Образ ставит оба CLI | ревью Dockerfile, без сборки образа в CI этой задачи | P1 |
| Живой прогон модели | не входит в гейт: нет ключа в CI | — |

## Automated Checks

| Spec / contract ref | Test level | Command / suite | Expected |
|---------------------|------------|-----------------|----------|
| Выбор агента, 503 | API | `go test ./internal/api/ ./internal/bootstrap/config/ -count=1` | pass |
| Гонки раннера и джоб | API | `go test ./internal/api/ ./internal/bootstrap/... -race -count=1` | pass |
| Модалка | web unit | `cd web && npm test -- src/components/NodeAgentEditDialog.test.tsx` | pass, текст без Cursor |
| Линтер Go | lint | `golangci-lint run ./internal/api/... ./internal/bootstrap/...` | pass |
| OpenSpec | schema | `openspec validate add-opencode-node-agent` | pass |

Полный `go test ./... -race` — на приёмке родителя, если пакетный прогон зелёный.

## Roles and Fixtures

| Role / tenant | Sign-in/setup | Fixture | What it proves |
|---------------|---------------|---------|----------------|
| Анонимный handler-тест | auth выключен, как в текущих тестах agent-edit | временная база afero/директория | 400/404/409/503 |
| Нет роли UI | Vitest без сервера | мок `startNodeAgentEdit` | текст и submit |

## Browser Scenarios

| ID | Spec / UI ref | Steps | Expected | P0/P1 |
|----|---------------|-------|----------|-------|
| W1 | Модалка без имени поставщика | Открыть страницу узла, нажать «Редактировать с агентом» | В пояснении нет «Cursor», есть «Агент отредактирует файл на сервере», пустой «Запустить» disabled | P1 |

P0 этой строки закрывает Vitest. W1 выполняется, если поднят локальный `kb serve` и браузер доступен. Без сервера W1 пропускается с записью в Notes: покрытие — Vitest.

## Responsive / Accessibility / Failure States

- Viewports and input methods: отдельный мобильный проход не нужен, композиция диалога не менялась.
- Keyboard/screen-reader checks: заголовок диалога прежний.
- Slow/offline/error/retry behavior: ошибка старта остаётся в модалке; проверяется существующим тестом, если он есть, иначе не расширяется.

## Environment

- Stack/start commands from project instructions: `KB_GIT_DISABLED=true KB_DATA_PATH=<tmp> go run ./cmd/kb serve` только для W1.
- Seed/reset method: временный каталог базы, без git.
- External dependency probes: `opencode` и `cursor-agent` в unit-тестах не вызываются. Сборка Docker-образа не является гейтом.

## Evidence

- Report path: вывод команд в ответе приёмки.
- Screenshots/logs/traces required: для W1 — снимок модалки, если браузерный прогон выполнен. Иначе явный skip со ссылкой на Vitest.

## Browser Skip

Не полный skip: поверхность модалки меняется. Если браузер или сервер недоступны, W1 пропускается, P0 остаётся на Vitest.
