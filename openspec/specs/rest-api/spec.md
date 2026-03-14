## Purpose

REST API для CRUD операций с узлами базы знаний, полнотекстового и ключевого поиска. В scaffold — минимальный набор эндпоинтов.

## Requirements

### Requirement: Конфигурация через KB_DATA_PATH

API ДОЛЖЕН (SHALL) использовать путь к базе из переменной окружения KB_DATA_PATH.

#### Сценарий: Запуск без KB_DATA_PATH

- **WHEN** kb-server запущен без KB_DATA_PATH
- **THEN** сервер возвращает ошибку конфигурации или не стартует

### Requirement: CRUD узлов

API MUST предоставлять эндпоинты для создания, чтения, обновления и удаления узлов (в scaffold — каркас/заглушки).

#### Сценарий: Получение узла по пути

- **WHEN** GET /api/nodes/{path}
- **THEN** возвращается узел или 404

#### Сценарий: Получение дерева тем

- **WHEN** GET /api/tree
- **THEN** возвращается иерархическое дерево тем и подтем

### Requirement: Ingestion

API MUST предоставлять эндпоинт POST /api/ingest для приёма текста и передачи в ingestion pipeline. Тело запроса MUST поддерживать поля: text (обязательно), source_url (опционально), source_author (опционально), type_hint (опционально). Допустимые значения type_hint: "auto", "article", "link", "note". При отсутствии или неизвестном значении type_hint MUST трактовать как "auto".

#### Сценарий: Отправка текста

- **WHEN** POST /api/ingest с телом { "text": "..." }
- **THEN** текст передаётся в Ingester, возвращается результат

#### Сценарий: Отправка с type_hint

- **WHEN** POST /api/ingest с телом { "text": "...", "type_hint": "article" }
- **THEN** текст и type_hint передаются в Ingester, возвращается результат

### Requirement: Список узлов с фильтрами

API MUST поддерживать GET /api/nodes с query-параметрами: path (путь темы, пустой = вся база), recursive (bool, по умолчанию false), q (подстрока поиска в title, keywords, annotation), type (article, link, note — через запятую), limit, offset (пагинация). При recursive=true возвращаются узлы всего поддерева. Ответ MUST содержать nodes (массив с path, title, type, created, source_url, translations) и total (общее количество до пагинации). Переводы (slug.lang.md) не включаются как отдельные узлы.

#### Сценарий: Рекурсивный список

- **WHEN** GET /api/nodes?path=programming&recursive=true
- **THEN** возвращаются узлы из programming и всех подпапок

#### Сценарий: Поиск по тексту

- **WHEN** GET /api/nodes?path=ai&recursive=true&q=go
- **THEN** возвращаются только узлы, где «go» входит в title, keywords или annotation

#### Сценарий: Фильтр по типу

- **WHEN** GET /api/nodes?path=&recursive=true&type=article,link
- **THEN** возвращаются только узлы типа article или link

#### Сценарий: Пагинация

- **WHEN** GET /api/nodes?path=ai&recursive=true&limit=20&offset=40
- **THEN** возвращаются узлы 41–60 и total для расчёта страниц

### Requirement: Поиск

Поиск по ключевым словам и подстроке в title/annotation MUST осуществляться через GET /api/nodes с параметром q. Полнотекстовый поиск по content — опционально (в scaffold — каркас).

#### Сценарий: Поиск по запросу

- **WHEN** GET /api/nodes?q=... (с path и recursive при необходимости)
- **THEN** возвращается список подходящих узлов с метаданными (nodes, total)

### Requirement: Раздача статики UI

API ДОЛЖЕН (SHALL) раздавать встроенную статику веб-интерфейса (embedded из internal/ui).

#### Сценарий: Запрос корня

- **WHEN** GET / (или /index.html)
- **THEN** возвращается index.html веб-приложения

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

### Requirement: Опциональная защита API сессией

При включённой авторизации (`KB_LOGIN` и `KB_PASSWORD` заданы) REST API ДОЛЖЕН (SHALL) требовать валидную сессионную cookie для доступа к защищённым endpoint `/api/*`, включая `/api/assets/*`. При отсутствии или невалидности сессии сервер MUST возвращать `401 Unauthorized`.

#### Сценарий: Доступ к API без сессии при включённой авторизации

- **WHEN** клиент вызывает защищённый endpoint `/api/*` без валидной cookie-сессии
- **THEN** сервер возвращает `401 Unauthorized`

#### Сценарий: Доступ к API с валидной сессией

- **WHEN** клиент вызывает защищённый endpoint `/api/*` с валидной cookie-сессией
- **THEN** сервер обрабатывает запрос по обычной бизнес-логике endpoint

### Requirement: Auth endpoints для login/logout/session

REST API MUST предоставлять endpoints `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/session` для работы web-клиента с сессионной авторизацией.

#### Сценарий: Login endpoint выдаёт сессию

- **WHEN** клиент отправляет корректные credentials на `POST /api/auth/login`
- **THEN** сервер создаёт сессию и устанавливает cookie в ответе

#### Сценарий: Session endpoint отражает текущий статус

- **WHEN** клиент вызывает `GET /api/auth/session`
- **THEN** сервер возвращает статус аутентификации текущей сессии

#### Сценарий: Logout endpoint завершает сессию

- **WHEN** клиент вызывает `POST /api/auth/logout`
- **THEN** сервер инвалидирует текущую сессию и очищает auth-cookie
