## ADDED Requirements

### Requirement: API получения логов нормализации
REST API SHALL предоставлять endpoint `GET /api/node-normalization/{id}/logs` для получения логов операции нормализации. Endpoint MUST поддерживать query-параметр `after` для инкрементального чтения и MUST возвращать только записи с offset больше `after`, а также `next_offset`.

#### Scenario: Чтение логов с начала
- **WHEN** клиент вызывает `/api/node-normalization/{id}/logs` без `after`
- **THEN** API возвращает доступные записи начиная с минимального offset и `next_offset`

#### Scenario: Инкрементальное чтение
- **WHEN** клиент вызывает `/api/node-normalization/{id}/logs?after=42`
- **THEN** API возвращает только записи с offset > 42

#### Scenario: Операция не найдена
- **WHEN** клиент запрашивает логи для неизвестного `id`
- **THEN** API возвращает 404
