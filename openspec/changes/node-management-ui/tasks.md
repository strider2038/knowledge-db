## 1. Backend: Store — удаление узла

- [ ] 1.1 Метод `DeleteNode` в `internal/kb/store.go`: удаление `.md` файла и директории вложений через afero.Fs
- [ ] 1.2 Обёрточная функция `DeleteNode` в `internal/kb/` (как существующие GetNode, PatchNodeManualProcessed)
- [ ] 1.3 Тесты `TestDeleteNode` в `internal/kb/`: успешное удаление, удаление с вложениями, не найден, пустой путь

## 2. Backend: Store — перемещение узла

- [ ] 2.1 Метод `MoveNode` в `internal/kb/store.go`: принимает `targetPath` (полный путь тема+slug), перемещает `.md` файл и директорию вложений, рекурсивно создаёт промежуточные директории, проверяет отсутствие path traversal и конфликта
- [ ] 2.2 Обёрточная функция `MoveNode` в `internal/kb/`
- [ ] 2.3 Тесты `TestMoveNode` в `internal/kb/`: перемещение в другую тему, с изменением slug, конфликт целевого пути, не найден, рекурсивное создание новых директорий, path traversal защита

## 3. Backend: API — хендлеры удаления и перемещения

- [ ] 3.1 Хендлер `DeleteNode` в `internal/api/handlers.go`: DELETE /api/nodes/{path}
- [ ] 3.2 Хендлер `MoveNode` в `internal/api/handlers.go`: POST /api/nodes/{path}/move, тело `{ target_path: string }`, разбор и валидация target_path
- [ ] 3.3 Роутинг в `internal/api/router.go`: регистрация новых маршрутов
- [ ] 3.4 API-тесты `TestDeleteNode` и `TestMoveNode` в `internal/api/`

## 4. Backend: Git-статус и коммит API

- [ ] 4.1 Метод `Status` в `internal/ingestion/git/committer.go`: выполнение `git status --porcelain`, подсчёт изменённых файлов
- [ ] 4.2 Метод `CommitAll` в `internal/ingestion/git/committer.go`: git add -A + commit + push с переданным сообщением
- [ ] 4.3 Обновление интерфейса `GitCommitter` и `NoopGitCommitter` / `SerializedGitCommitter`
- [ ] 4.4 Функция генерации commit message через LLM (OpenAI Responses API) на основе `git diff --stat`, с fallback на шаблонное сообщение
- [ ] 4.5 Хендлер `GetGitStatus` в `internal/api/handlers.go`: GET /api/git/status
- [ ] 4.6 Хендлер `PostGitCommit` в `internal/api/handlers.go`: POST /api/git/commit с интеграцией LLM
- [ ] 4.7 Роутинг: регистрация /api/git/status и /api/git/commit в router.go
- [ ] 4.8 API-тесты для git-эндпоинтов

## 5. Frontend: API-клиент

- [ ] 5.1 Добавить функции `deleteNode`, `moveNode`, `getGitStatus`, `postGitCommit` в `web/src/services/api.ts`
- [ ] 5.2 Обновить типы в `web/src/types/` при необходимости

## 6. Frontend: Панель действий (Action Bar) на NodePage

- [ ] 6.1 Создать компонент `NodeActionBar` в `web/src/components/` с группами кнопок: Проверено/Перевести и Переместить/Удалить
- [ ] 6.2 Интегрировать `NodeActionBar` в `NodePage.tsx` после breadcrumbs, перенести toggle «Проверено» и «Перевести» в панель
- [ ] 6.3 Стилизация: sticky позиционирование, разделитель между группами, destructive стиль для «Удалить»

## 7. Frontend: Модалка подтверждения удаления

- [ ] 7.1 Создать компонент `AlertDialog` (ui/alert-dialog.tsx) на базе Radix UI (shadcn/ui паттерн)
- [ ] 7.2 Создать `DeleteNodeDialog` в `web/src/components/` с показом заголовка, пути, кнопок «Отмена»/«Удалить»
- [ ] 7.3 Интегрировать в `NodeActionBar`: по клику «Удалить» → открыть модалку → подтвердить → API вызов → переход на обзор

## 8. Frontend: Модалка перемещения узла

- [ ] 8.1 Создать компонент `MoveNodeDialog` в `web/src/components/`: текстовое поле «Новый путь» (полный путь, редактируемое), дерево существующих тем (клик заполняет тему в поле), кнопка «Переместить»
- [ ] 8.2 Переиспользовать дерево тем из OverviewPage (или вынести в отдельный компонент TopicTree)
- [ ] 8.3 Интегрировать в `NodeActionBar`: по клику «Переместить» → открыть модалку → отредактировать путь (или выбрать тему из дерева) → API вызов с `target_path` → переход на новый путь

## 9. Frontend: Кнопка «Коммит» в Navbar

- [ ] 9.1 Создать хук `useGitStatus` в `web/src/hooks/`: polling GET /api/git/status каждые 30 секунд
- [ ] 9.2 Обновить `Navbar.tsx`: условное отображение кнопки «Коммит» с количеством файлов
- [ ] 9.3 Обработчик клика: вызов POST /api/git/commit, спиннер, уведомление результата
- [ ] 9.4 Стилизация кнопки «Коммит» в стиле Navbar

## 10. Проверка и сборка

- [ ] 10.1 Go: `go build ./...` и `go test ./... -race` без ошибок
- [ ] 10.2 Frontend: `cd web && npm run build` без ошибок
- [ ] 10.3 Go линтер: `golangci-lint run ./...` без новых ошибок
- [ ] 10.4 TypeScript линтер (при наличии): `cd web && npm run lint` без новых ошибок
