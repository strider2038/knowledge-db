## MODIFIED Requirements

### Requirement: CRUD узлов

API MUST предоставлять эндпоинты для создания, чтения, обновления и удаления узлов. **Чтения**
адресуются по пути: `GET /api/nodes/{path...}` и `GET /api/nodes/by-id/{id}` (санкционированное
REST-исключение для shareable deep-links). **Мутации** используют POST-action контракт с
идентификатором в теле: обновление — `POST /api/nodes/update`, удаление — `POST /api/nodes/delete`,
перемещение — `POST /api/nodes/move`. REST-глаголы `PATCH`/`DELETE` под `/api/` не используются.
Каждый ответ с объектом узла MUST включать поле `id` (string, UUID).

#### Сценарий: Получение узла по пути

- **WHEN** `GET /api/nodes/{path}`
- **THEN** возвращается узел с полями `id` и `path` или 404

#### Сценарий: Обновление frontmatter узла

- **WHEN** `POST /api/nodes/update` с телом `{ path, ...поддерживаемые поля }`
  (`manual_processed`, `title`, `keywords`, `labels`)
- **THEN** обновляются только переданные поля; при отсутствии `path` — 400,
  при неподдерживаемом поле — 400, при отсутствии узла — 404

#### Сценарий: Удаление узла

- **WHEN** `POST /api/nodes/delete` с телом `{ path }`
- **THEN** узел (файл .md и директория вложений) удаляется, возвращается `{ path, id, deleted: true }`
  или 404; при отсутствии `path` — 400

#### Сценарий: Перемещение узла

- **WHEN** `POST /api/nodes/move` с телом `{ path, target_path }`
- **THEN** узел перемещается по указанному пути, `id` в ответе совпадает с id до move,
  `path` обновлён, 409 при конфликте, 400 при отсутствии `path`/`target_path`

#### Сценарий: Список узлов содержит id

- **WHEN** `GET /api/nodes` с фильтрами
- **THEN** каждый элемент списка содержит поле `id`

### Requirement: Обновление описания узла из источника

REST API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/nodes/refresh-description` для
обновления описания существующего узла на основе его `source_url`. Путь узла передаётся в теле
запроса (`{ path }`). Endpoint MUST загружать текущий узел, требовать наличие `source_url`,
запускать refresh mode inference из ingestion-pipeline spec, и сохранять обновлённый markdown-файл
узла. Ответ MUST содержать обновлённый объект узла. Если `source_url` отсутствует — 400, если узел
не найден — 404, при отсутствии `path` — 400. Для обычных note без digest profile refresh MUST NOT
переписывать markdown-тело в digest.

Endpoint MUST обновлять описательные поля: `annotation`, `keywords`, `source_kind`,
`content_profile` и markdown-тело digest. Endpoint MAY обновить `type`. Endpoint MUST сохранять
`created`, `source_url`, `manual_processed` и пользовательские поля. Endpoint SHOULD сохранять
существующие `source_author` и `source_date`, если новый источник не даёт более точных значений.

#### Scenario: Обновление repository link

- **WHEN** клиент вызывает `POST /api/nodes/refresh-description` с телом `{ path }` для узла с
  `source_url` на GitHub-репозиторий
- **THEN** API обновляет узел как `type=link`, `source_kind=repository`,
  `content_profile=repository_profile` и возвращает обновлённый объект узла

#### Scenario: Узел без source_url

- **WHEN** клиент вызывает `POST /api/nodes/refresh-description` для узла без `source_url`
- **THEN** API возвращает 400

#### Scenario: Узел не найден

- **WHEN** клиент вызывает `POST /api/nodes/refresh-description` для неизвестного пути
- **THEN** API возвращает 404

## ADDED Requirements

### Requirement: POST-action контракт мутаций

Все мутации HTTP API MUST использовать форму `POST /api/<resource>/<action>` с идентификатором
или путём в JSON-теле (snake_case: `path`, `id`, `target_path`). REST-глаголы `PUT`/`DELETE`/`PATCH`
под `/api/` MUST NOT использоваться. Адресация по пути (`{path...}`, `{id}`) допускается ТОЛЬКО для
`GET`-чтений (deep-links, скачивание ассетов, поллинг статуса/логов). Диспетчеризация по суффиксу
пути (единый wildcard-обработчик, разбирающий действие из `path`) MUST NOT применяться — каждое
действие регистрируется отдельным маршрутом и обработчиком.

Соответствие MUST быть закреплено source-scanning тестом (`internal/api/router_guard_test.go`),
который сканирует `router.go` и падает, если любой маршрут использует `PUT`/`DELETE`/`PATCH`.

#### Сценарий: Guard отклоняет REST-глагол

- **WHEN** в `router.go` регистрируется маршрут с методом `PUT`, `DELETE` или `PATCH`
- **THEN** `router_guard_test.go` падает с указанием нарушающего маршрута

#### Сценарий: Guard допускает GET deep-link

- **WHEN** в `router.go` регистрируется `GET /api/nodes/{path...}` или `GET /api/jobs/{id}/logs`
- **THEN** `router_guard_test.go` проходит (адресация по пути разрешена для чтений)

### Requirement: POST-action мутации чатов, issues, telegram-импорта и перевода

API MUST предоставлять мутации для чат-сессий, debug-issues, telegram-импорта и перевода статей в
POST-action форме с идентификатором в теле:

- `POST /api/chats/update` — `{ id, title }` (переименование), 404 при отсутствии
- `POST /api/chats/delete` — `{ id }`, 404 при отсутствии
- `POST /api/debug/issues/update` — `{ id, ...обновляемые поля статуса }`
- `POST /api/import/telegram/session/accept` — `{ id, ... }`
- `POST /api/import/telegram/session/reject` — `{ id }`
- `POST /api/articles/translate` — `{ path, ... }`

`GET`-чтения этих ресурсов (`GET /api/chats/{id}`, `GET /api/import/telegram/session/{id}`,
`GET /api/articles/translate/{path...}`) остаются без изменений.

#### Сценарий: Переименование чат-сессии

- **WHEN** `POST /api/chats/update` с телом `{ id, title }`
- **THEN** сессия переименовывается, возвращается `{ id, title }`; при пустом `title` — 400,
  при отсутствии сессии — 404

#### Сценарий: Приём telegram-сессии

- **WHEN** `POST /api/import/telegram/session/accept` с телом `{ id, ... }`
- **THEN** сессия принимается так же, как ранее при `POST .../session/{id}/accept`
