## ADDED Requirements

### Requirement: API запуска dump images для узла

REST API SHALL предоставлять endpoint запуска `dump images` для узла (например `POST /api/nodes/{path}/dump-images`) и endpoint получения статуса операции. API MUST принимать path текущего узла, запускать асинхронную операцию и возвращать диагностируемый статус выполнения.

#### Сценарий: Успешный старт операции
- **WHEN** клиент вызывает endpoint `dump images` для существующего article-узла
- **THEN** API возвращает статус запуска (`running/accepted`) и идентификатор операции

#### Сценарий: Невалидный тип узла
- **WHEN** клиент вызывает endpoint `dump images` для узла не типа article
- **THEN** API возвращает ошибку валидации и MUST NOT запускать операцию

### Requirement: API получения логов dump images

REST API SHALL предоставлять endpoint `GET /api/node-dump-images/{id}/logs` для получения логов операции `dump images`. Endpoint MUST поддерживать query-параметр `after` для инкрементального чтения и MUST возвращать только записи с offset больше `after`, а также `next_offset`.

#### Сценарий: Чтение логов с начала
- **WHEN** клиент вызывает `/api/node-dump-images/{id}/logs` без `after`
- **THEN** API возвращает доступные записи начиная с минимального offset и `next_offset`

#### Сценарий: Инкрементальное чтение
- **WHEN** клиент вызывает `/api/node-dump-images/{id}/logs?after=42`
- **THEN** API возвращает только записи с offset > 42

#### Сценарий: Операция не найдена
- **WHEN** клиент запрашивает логи для неизвестного `id`
- **THEN** API возвращает 404

### Requirement: Серверный post-step sync после dump images

После успешного завершения шага `dump images` API MUST запускать `sync` как обязательный post-step и SHALL возвращать клиенту итоговый статус с учётом шага sync.

#### Сценарий: Успешный sync после dump images
- **WHEN** `dump images` завершился успешно и `sync` завершился успешно
- **THEN** API возвращает итог success

#### Сценарий: Ошибка sync после dump images
- **WHEN** `dump images` завершился успешно, но `sync` завершился ошибкой
- **THEN** API возвращает ошибку шага sync с деталями
