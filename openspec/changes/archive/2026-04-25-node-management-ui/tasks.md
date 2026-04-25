## 1. Backend: Store — удаление узла

- [x] 1.1 Метод `DeleteNode` в `internal/kb/store.go`: удаление `.md` файла и директории вложений через afero.Fs
- [x] 1.2 Обёрточная функция `DeleteNode` в `internal/kb/` (как существующие GetNode, PatchNodeManualProcessed)
- [x] 1.3 Тесты `TestDeleteNode` в `internal/kb/`: успешное удаление, удаление с вложениями, не найден, пустой путь

## 2. Backend: Store — перемещение узла

- [x] 2.1 Метод `MoveNode` в `internal/kb/store.go`: принимает `targetPath` (полный путь тема+slug), перемещает `.md` файл и директорию вложений, рекурсивно создаёт промежуточные директории, проверяет отсутствие path traversal и конфликта
- [x] 2.2 Обёрточная функция `MoveNode` в `internal/kb/`
- [x] 2.3 Тесты `TestMoveNode` в `internal/kb/`: перемещение в другую тему, с изменением slug, конфликт целевого пути, не найден, рекурсивное создание новых директорий, path traversal защита

## 3. Backend: API — хендлеры удаления и перемещения

- [x] 3.1 Хендлер `DeleteNode` в `internal/api/handlers.go`: DELETE /api/nodes/{path}
- [x] 3.2 Хендлер `MoveNode` в `internal/api/handlers.go`: POST /api/nodes/{path}/move, тело `{ target_path: string }`, разбор и валидация target_path
- [x] 3.3 Роутинг в `internal/api/router.go`: регистрация новых маршрутов
- [x] 3.4 API-тесты `TestDeleteNode` и `TestMoveNode` в `internal/api/`

## 4. Backend: Git-статус и коммит API

- [x] 4.1 Метод `Status` в `internal/ingestion/git/committer.go`: выполнение `git status --porcelain`, подсчёт изменённых файлов
- [x] 4.2 Метод `CommitAll` в `internal/ingestion/git/committer.go`: git add -A + commit + push с переданным сообщением
- [x] 4.3 Обновление интерфейса `GitCommitter` и `NoopGitCommitter` / `SerializedGitCommitter`
- [x] 4.4 Функция генерации commit message через LLM (OpenAI Responses API) на основе `git diff --stat`, с fallback на шаблонное сообщение
- [x] 4.5 Хендлер `GetGitStatus` в `internal/api/handlers.go`: GET /api/git/status
- [x] 4.6 Хендлер `PostGitCommit` в `internal/api/handlers.go`: POST /api/git/commit с интеграцией LLM
- [x] 4.7 Роутинг: регистрация /api/git/status и /api/git/commit в router.go
- [x] 4.8 API-тесты для git-эндпоинтов

## 5. Frontend: API-клиент

- [x] 5.1 Добавить функции `deleteNode`, `moveNode`, `getGitStatus`, `postGitCommit` в `web/src/services/api.ts`
- [x] 5.2 Обновить типы в `web/src/types/` при необходимости

## 6. Frontend: Панель действий (Action Bar) на NodePage

- [x] 6.1 Создать компонент `NodeActionBar` в `web/src/components/` с группами кнопок: Проверено/Перевести и Переместить/Удалить
- [x] 6.2 Интегрировать `NodeActionBar` в `NodePage.tsx` после breadcrumbs, перенести toggle «Проверено» и «Перевести» в панель
- [x] 6.3 Стилизация: sticky позиционирование, разделитель между группами, destructive стиль для «Удалить»

## 7. Frontend: Модалка подтверждения удаления

- [x] 7.1 Создать компонент `AlertDialog` (ui/alert-dialog.tsx) на базе Radix UI (shadcn/ui паттерн)
- [x] 7.2 Создать `DeleteNodeDialog` в `web/src/components/` с показом заголовка, пути, кнопок «Отмена»/«Удалить»
- [x] 7.3 Интегрировать в `NodeActionBar`: по клику «Удалить» → открыть модалку → подтвердить → API вызов → переход на обзор

## 8. Frontend: Модалка перемещения узла

- [x] 8.1 Создать компонент `MoveNodeDialog` в `web/src/components/`: текстовое поле «Новый путь» (полный путь, редактируемое), дерево существующих тем (клик заполняет тему в поле), кнопка «Переместить»
- [x] 8.2 Переиспользовать дерево тем из OverviewPage (или вынести в отдельный компонент TopicTree)
- [x] 8.3 Интегрировать в `NodeActionBar`: по клику «Переместить» → открыть модалку → отредактировать путь (или выбрать тему из дерева) → API вызов с `target_path` → переход на новый путь

## 9. Frontend: «Сохранить» в Navbar (git)

- [x] 9.1 Создать хук `useGitStatus` в `web/src/hooks/`: polling GET /api/git/status каждые 30 секунд
- [x] 9.2 Обновить `Navbar.tsx`: кнопка «Сохранить» (активна при изменениях; неактивна при чистом дереве; ⚠️ и неактивна при git off), подсказки по наведению
- [x] 9.3 Обработчик: POST /api/git/commit, состояние «Сохранение...», результат через toast снизу (Radix Toast)
- [x] 9.4 Стилизация кнопки в стиле Navbar; `ToasterViewport` + `Toast.Provider` в корне приложения

## 10. Проверка и сборка

- [x] 10.1 Go: `go build ./...` и `go test ./... -race` без ошибок
- [x] 10.2 Frontend: `cd web && npm run build` без ошибок
- [x] 10.3 Go линтер: `golangci-lint run ./...` без новых ошибок
- [x] 10.4 TypeScript линтер (при наличии): `cd web && npm run lint` без новых ошибок
