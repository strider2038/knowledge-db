## ADDED Requirements

### Requirement: Конфигурация KB_UPLOADS_DIR для импорта

API ДОЛЖЕН (SHALL) использовать путь из переменной окружения KB_UPLOADS_DIR для хранения данных импорта (в т.ч. telegram-import-sessions). При отсутствии KB_UPLOADS_DIR эндпоинты импорта MUST возвращать 503 или 400 с сообщением о неконфигурированном импорте.

#### Сценарий: Импорт без KB_UPLOADS_DIR

- **WHEN** KB_UPLOADS_DIR не задана и выполняется POST /api/import/telegram
- **THEN** возвращается ошибка (503 или 400) с сообщением о необходимости настройки

### Requirement: Импорт Telegram — создание сессии

API MUST предоставлять эндпоинт POST /api/import/telegram. Тело запроса — JSON экспорта одного чата Telegram. При успехе MUST создаваться сессия в {KB_UPLOADS_DIR}/telegram-import-sessions/{session_id}.json и возвращаться { session_id, total, current_index, current_item }. current_item — первая необработанная запись (текст, source_author, source_url) или null при пустом списке.

#### Сценарий: Создание сессии из валидного JSON

- **WHEN** POST /api/import/telegram с телом — JSON чата с messages[]
- **THEN** создаётся сессия, возвращается session_id, total, current_item

#### Сценарий: Невалидный JSON

- **WHEN** POST /api/import/telegram с невалидным JSON
- **THEN** возвращается 400 с сообщением об ошибке

### Requirement: Импорт Telegram — получение состояния сессии

API MUST предоставлять эндпоинт GET /api/import/telegram/session/:id. Ответ MUST содержать session_id, total, current_index, processed_count, rejected_count, current_item (или null если все обработаны).

#### Сценарий: Получение состояния

- **WHEN** GET /api/import/telegram/session/{valid_id}
- **THEN** возвращается текущее состояние сессии

#### Сценарий: Сессия не найдена

- **WHEN** GET /api/import/telegram/session/{unknown_id}
- **THEN** возвращается 404

### Requirement: Импорт Telegram — принять запись

API MUST предоставлять эндпоинт POST /api/import/telegram/session/:id/accept. Тело MAY содержать type_hint ("auto", "article", "link", "note"). При успехе MUST вызываться Ingester с text, source_url, source_author текущей записи и type_hint из тела; запись помечается processed; current_index увеличивается. Ответ MUST содержать созданный node и next_item (следующая запись или null).

#### Сценарий: Принять с type_hint

- **WHEN** POST /api/import/telegram/session/:id/accept с { type_hint: "article" }
- **THEN** текущая запись передаётся в Ingester, возвращается node и next_item

#### Сценарий: Сессия завершена

- **WHEN** все записи обработаны и вызывается accept
- **THEN** возвращается 400 или 409 (нет текущей записи)

### Requirement: Импорт Telegram — отклонить запись

API MUST предоставлять эндпоинт POST /api/import/telegram/session/:id/reject. При успехе запись помечается rejected, current_index увеличивается. Ответ MUST содержать next_item.

#### Сценарий: Отклонить запись

- **WHEN** POST /api/import/telegram/session/:id/reject
- **THEN** текущая запись помечается rejected, возвращается next_item
