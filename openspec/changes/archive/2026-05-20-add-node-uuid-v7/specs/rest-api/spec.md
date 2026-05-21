## Purpose

Дельта: поле `id` в API, lookup узла по id.

## ADDED Requirements

### Requirement: Получение узла по id

REST API MUST предоставлять эндпоинт `GET /api/nodes/by-id/{id}` для чтения узла по стабильному UUID. Ответ MUST совпадать по форме с `GET /api/nodes/{path}` (тот же JSON-объект узла, включая актуальный `path` и `id`).

#### Scenario: Успешное получение по id

- **WHEN** `GET /api/nodes/by-id/{valid-uuid}`
- **THEN** возвращается 200 и объект узла с полями `id` и `path`

#### Scenario: Узел не найден по id

- **WHEN** `GET /api/nodes/by-id/{unknown-uuid}`
- **THEN** возвращается 404

#### Scenario: Невалидный id

- **WHEN** `GET /api/nodes/by-id/not-a-uuid`
- **THEN** возвращается 400

### Requirement: Список узлов без фильтра по id

`GET /api/nodes` MUST NOT принимать query-параметр `id` для поиска узла. Чтение по стабильному идентификатору MUST выполняться только через `GET /api/nodes/by-id/{id}`.

#### Scenario: Запрос списка с id

- **WHEN** клиент вызывает `GET /api/nodes?id={uuid}`
- **THEN** возвращается 400 с указанием использовать `GET /api/nodes/by-id/{id}`

## MODIFIED Requirements

### Requirement: CRUD узлов

API MUST предоставлять эндпоинты для создания, чтения, обновления и удаления узлов (в scaffold — каркас/заглушки). Добавляется поддержка DELETE для удаления узла и POST /move для перемещения. Каждый ответ с объектом узла MUST включать поле `id` (string, UUID). Эндпоинт `GET /api/nodes/by-id/{id}` MUST быть доступен для чтения по стабильному идентификатору.

#### Сценарий: Получение узла по пути

- **WHEN** GET /api/nodes/{path}
- **THEN** возвращается узел с полями `id` и `path` или 404

#### Сценарий: Получение дерева тем

- **WHEN** GET /api/tree
- **THEN** возвращается иерархическое дерево тем и подтем

#### Сценарий: Удаление узла

- **WHEN** DELETE /api/nodes/{path}
- **THEN** узел (файл .md и директория вложений) удаляется, возвращается `{ path, id, deleted: true }` или 404

#### Сценарий: Перемещение узла

- **WHEN** POST /api/nodes/{path}/move с `{ target_path: "new/topic/slug" }`
- **THEN** узел перемещается по указанному пути, `id` в ответе совпадает с id до move, `path` обновлён, 409 при конфликте

#### Сценарий: Список узлов содержит id

- **WHEN** GET /api/nodes с фильтрами
- **THEN** каждый элемент списка содержит поле `id`
