## ADDED Requirements

### Requirement: API запуска редактирования узла агентом

REST API SHALL предоставлять endpoint `POST /api/nodes/{path}/agent-edit` для запуска операции редактирования узла через Cursor Agent. Тело запроса MUST быть JSON с полем `instruction` (string). Ответ MUST содержать идентификатор операции, `node_path`, `status`, `stage`, временные метки и флаги `edit_ok`, `sync_done` (по мере готовности). При отсутствии узла API MUST возвращать 404; при пустой инструкции — 400; при недоступном `cursor-agent` — диагностируемую ошибку runtime (503 или согласованный код проекта).

#### Scenario: Успешный старт

- **WHEN** клиент вызывает `POST /api/nodes/{path}/agent-edit` с валидной `instruction` для существующего узла
- **THEN** API возвращает 200/202 с объектом операции и `status: running`

#### Scenario: Валидация instruction

- **WHEN** тело запроса не содержит `instruction` или оно пустое после trim
- **THEN** API возвращает 400 с сообщением об ошибке

### Requirement: API статуса операции agent-edit

REST API SHALL предоставлять endpoint `GET /api/node-agent-edit/{id}` для получения текущего статуса операции agent-edit. Ответ MUST использовать snake_case и поля, совместимые с UI polling (как у node-normalization).

#### Scenario: Получение статуса running

- **WHEN** клиент запрашивает статус существующей running-операции
- **THEN** API возвращает `status: running`, актуальную `stage` и `node_path`

#### Scenario: Операция не найдена

- **WHEN** клиент запрашивает статус неизвестного `id`
- **THEN** API возвращает 404

### Requirement: API логов операции agent-edit

REST API SHALL предоставлять endpoint `GET /api/node-agent-edit/{id}/logs` для получения логов операции. Endpoint MUST поддерживать query-параметр `after` для инкрементального чтения и MUST возвращать только записи с offset больше `after`, а также `next_offset`. Формат записей MUST быть совместим с логами нормализации (`offset`, `stream`, `text`, `timestamp`).

#### Scenario: Инкрементальное чтение логов

- **WHEN** клиент вызывает `/api/node-agent-edit/{id}/logs?after=10`
- **THEN** API возвращает только записи с offset > 10 и `next_offset`

#### Scenario: Логи неизвестной операции

- **WHEN** `id` не существует
- **THEN** API возвращает 404
