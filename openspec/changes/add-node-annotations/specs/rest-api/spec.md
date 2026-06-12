## Purpose

Дельта: REST API для CRUD личных аннотаций узла.

## ADDED Requirements

### Requirement: API чтения аннотаций узла

REST API MUST предоставлять `GET /api/nodes/{basePath}/annotations`, где `basePath` — базовый путь узла без суффикса языка. Ответ MUST быть JSON-объектом с полем `notes` (массив аннотаций). Каждая аннотация MUST включать `id`, `created`, `updated`, `body`, `anchor` (null или объект) и для привязанных — `resolved` (boolean). Если узел не найден — 404.

#### Сценарий: Успешное чтение

- **WHEN** `GET /api/nodes/topic/article/annotations` и узел существует
- **THEN** возвращается 200 и список аннотаций (возможно пустой)

#### Сценарий: Чтение с пути перевода

- **WHEN** `GET /api/nodes/topic/article.ru/annotations`
- **THEN** возвращается тот же список, что для `topic/article`

#### Сценарий: Узел не найден

- **WHEN** `GET /api/nodes/nonexistent/annotations`
- **THEN** возвращается 404

### Requirement: API создания аннотации

REST API MUST предоставлять `POST /api/nodes/{basePath}/annotations`. Тело MUST содержать `body` (string, обязательное) и опционально `anchor` (объект `text_quote`). При успехе MUST возвращаться созданная аннотация со статусом 201. Сервер MUST присвоить `id`, `created`, `updated`.

#### Сценарий: Общая аннотация

- **WHEN** `POST` с `{ "body": "Перечитать позже" }`
- **THEN** создаётся запись с `anchor: null`, статус 201

#### Сценарий: Привязанная аннотация

- **WHEN** `POST` с `body` и `anchor` типа `text_quote`
- **THEN** создаётся запись с якорем, `resolved` вычисляется при ответе

### Requirement: API обновления и удаления аннотации

REST API MUST предоставлять `PATCH /api/nodes/{basePath}/annotations/{id}` (поля `body` и/или `anchor`) и `DELETE /api/nodes/{basePath}/annotations/{id}`. При успешном PATCH MUST обновляться `updated`. Если аннотация или узел не найдены — 404.

#### Сценарий: Редактирование текста

- **WHEN** `PATCH` с `{ "body": "новый текст" }`
- **THEN** возвращается обновлённая аннотация, `updated` изменён

#### Сценарий: Перепривязка якоря

- **WHEN** `PATCH` с новым объектом `anchor`
- **THEN** якорь заменяется, `resolved` пересчитывается

#### Сценарий: Удаление

- **WHEN** `DELETE /api/nodes/topic/article/annotations/{id}` и id существует
- **THEN** запись удаляется из sidecar, статус 204 или 200
