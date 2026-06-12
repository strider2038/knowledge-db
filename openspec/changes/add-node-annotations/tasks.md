## 1. Backend: хранение аннотаций

- [ ] 1.1 Добавить типы и парсинг `annotations.yaml` в `internal/kb` (`version`, `notes`, anchor `text_quote`)
- [ ] 1.2 Реализовать CRUD sidecar: чтение, создание, обновление, удаление записей с лимитами (200 заметок, body 4000, exact 500)
- [ ] 1.3 Нормализация `basePath` (без суффикса перевода) для всех операций
- [ ] 1.4 Resolve якорей при GET: `resolved` по поиску `exact` в теле `content_path`
- [ ] 1.5 Unit-тесты store: пустой sidecar, CRUD, лимиты, resolve true/false

## 2. Backend: move и delete

- [ ] 2.1 Перенос `annotations.yaml` при `MoveNode` вместе с директорией вложений
- [ ] 2.2 Удаление sidecar при `DeleteNode`
- [ ] 2.3 Тесты move/delete с аннотациями

## 3. Backend: API

- [ ] 3.1 Маршруты `GET/POST /api/nodes/{path}/annotations`, `PATCH/DELETE .../annotations/{id}`
- [ ] 3.2 Handlers: 404 узел/заметка, 422 валидация, JSON snake_case
- [ ] 3.3 API-тесты: CRUD, basePath перевода, resolved в ответе

## 4. Backend: индекс

- [ ] 4.1 Убедиться, что sync не читает `annotations.yaml` и смена только sidecar не меняет hash
- [ ] 4.2 Тест: добавление заметки не триггерит re-embed

## 5. Frontend: API-клиент

- [ ] 5.1 Типы и функции в `api.ts`: `getNodeAnnotations`, `createNodeAnnotation`, `updateNodeAnnotation`, `deleteNodeAnnotation`
- [ ] 5.2 Unit-тесты api client

## 6. Frontend: панель заметок

- [ ] 6.1 Компонент `NodeAnnotationsPanel` — список, пустое состояние, `+ Заметка`, textarea с debounced autosave
- [ ] 6.2 Интеграция в `NodePage`: layout справа (lg+), Sheet на мобиле, `basePath` для API
- [ ] 6.3 Удаление заметки, отображение дат, orphan-бейдж

## 7. Frontend: привязка к тексту

- [ ] 7.1 Обёртка выделения в блоке «Содержание»: toolbar «Добавить заметку», формирование anchor
- [ ] 7.2 Маркеры ● у абзацев с `resolved: true`; подсветка цитаты
- [ ] 7.3 Jump: клик заметка ↔ фрагмент; «Перепривязать» для orphan
- [ ] 7.4 Тесты NodePage: панель, создание общей и привязанной заметки

## 8. Завершение

- [ ] 8.1 `go build ./... && go test ./... -race`, `golangci-lint run ./...`
- [ ] 8.2 `cd web && npm run build && npm run test && npm run lint`
- [ ] 8.3 `openspec validate add-node-annotations`
