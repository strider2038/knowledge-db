## 1. OpenSpec artifacts

- [x] 1.1 Подготовить proposal с мотивацией и scope изменения.
- [x] 1.2 Подготовить design с решениями и рисками.
- [x] 1.3 Подготовить delta specs для `rest-api` и `webapp`.

## 2. Backend: PATCH metadata

- [x] 2.1 Расширить `PATCH /api/nodes/{path...}` для полей `title` и `keywords` (с whitelist и валидацией типов).
- [x] 2.2 Реализовать санитизацию `title`/`keywords` в `kb.Store`.
- [x] 2.3 Добавить/обновить тесты API и Store на новые сценарии.

## 3. Frontend: UX ручного редактирования

- [x] 3.1 Добавить иконки-карандаши рядом с заголовком и ключевыми словами.
- [x] 3.2 Добавить модалку редактирования заголовка.
- [x] 3.3 Добавить модалку редактирования тегов (chips + keyboard interactions).
- [x] 3.4 Реализовать suggestions ключевиков на основе существующих keywords.
- [x] 3.5 Добавить/обновить frontend тесты.

## 4. Verification

- [x] 4.1 Прогнать backend проверки: `go build ./... && go test ./... -race`.
- [x] 4.2 Прогнать Go lint для новых изменений.
- [x] 4.3 Прогнать frontend проверки: `npm run build`, `npm run test`, `npm run lint`.
- [x] 4.4 Прогнать `openspec validate edit-node-title-keywords`.
- [x] 4.5 Прогнать `openspec status --change edit-node-title-keywords`.
