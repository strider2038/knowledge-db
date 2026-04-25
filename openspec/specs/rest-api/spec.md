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

API MUST поддерживать GET /api/nodes с query-параметрами: path (путь темы, пустой = вся база), recursive (bool, по умолчанию false), q (подстрока поиска в title, keywords, annotation), type (article, link, note — через запятую), manual_processed (опционально: true или false — только узлы с соответствующим флагом; при отсутствии параметра возвращаются все узлы независимо от флага), limit, offset (пагинация). При recursive=true возвращаются узлы всего поддерева. Ответ MUST содержать nodes (массив с path, title, type, created, source_url, translations, manual_processed) и total (общее количество до пагинации). Узлы без поля manual_processed в хранилище MUST трактоваться как manual_processed=false в JSON. Переводы (slug.lang.md) не включаются как отдельные узлы.

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

#### Сценарий: Фильтр только проверенных вручную

- **WHEN** GET /api/nodes?path=&recursive=true&manual_processed=true
- **THEN** возвращаются только узлы с manual_processed=true

#### Сценарий: Фильтр только непроверенных

- **WHEN** GET /api/nodes?path=&recursive=true&manual_processed=false
- **THEN** возвращаются только узлы без отметки или с manual_processed=false

### Requirement: Метаданные узла содержат manual_processed

Ответ GET узла по пути (и любые ответы с полным телом метаданных узла, используемые веб-клиентом) MUST содержать boolean поле manual_processed (false, если в файле поле отсутствует).

#### Сценарий: Чтение узла без поля в файле

- **WHEN** GET узла для .md без ключа manual_processed
- **THEN** в JSON manual_processed равен false

### Requirement: Обновление manual_processed

API MUST позволять установить или снять флаг manual_processed при сохранении метаданных узла тем же способом, как обновляются прочие редактируемые поля frontmatter (один запрос на сохранение метаданных узла). Некорректный тип значения MUST приводить к 400.

#### Сценарий: Установка флага

- **WHEN** клиент отправляет сохранение метаданных с manual_processed=true
- **THEN** в файле узла в frontmatter записывается manual_processed: true (или эквивалентный YAML), updated обновляется по правилам Store

#### Сценарий: Снятие флага

- **WHEN** клиент отправляет manual_processed=false
- **THEN** в frontmatter флаг снят или записан как false согласно принятому в реализации представлению опциональных булевых полей

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

При включённой авторизации (парольный режим: заданы `KB_LOGIN` и `KB_PASSWORD`; **или** Google-режим: полный набор Google OAuth и `KB_AUTH_ALLOWED_EMAILS` при пустых `KB_LOGIN` / `KB_PASSWORD`, см. `web-session-auth`) REST API ДОЛЖЕН (SHALL) требовать валидную сессионную cookie для доступа к защищённым эндпоинтам `/api/*`, **кроме** явно разрешённых путей аутентификации: `POST /api/auth/login` (в парольном режиме), `GET /api/auth/google` и `GET /api/auth/google/callback` (в Google-режиме), а также health/readiness. Маршруты `/api/*`, включая `/api/assets/*`, SHALL требовать сессию, если не относятся к перечисленным исключениям. При отсутствии или невалидности сессии на защищённом пути сервер MUST возвращать `401 Unauthorized`.

#### Сценарий: Доступ к API без сессии при включённой авторизации

- **WHEN** клиент вызывает защищённый эндпоинт `/api/*` (не из исключений) без валидной cookie-сессии
- **THEN** сервер возвращает `401 Unauthorized`

#### Сценарий: Доступ к API с валидной сессией

- **WHEN** клиент вызывает защищённый эндпоинт `/api/*` с валидной cookie-сессией
- **THEN** сервер обрабатывает запрос по обычной бизнес-логике эндпоинта

### Requirement: Эндпоинты Google OAuth

В Google-режиме (см. `web-session-auth`) REST API MUST предоставлять `GET /api/auth/google` (инициация редиректа на Google) и `GET /api/auth/google/callback` (приём `code`/`state`). Эти маршруты SHALL располагаться под `/api/auth/` и SHALL быть исключены из обязательной проверки сессии так же, как `POST /api/auth/login` в парольном режиме (до установки сессии).

#### Сценарий: Старт OAuth

- **WHEN** Google-режим и клиент запрашивает `GET /api/auth/google` с корректным `Origin` (при настроенном CORS)
- **THEN** сервер SHALL отвечать редиректом (3xx) на URL авторизации Google с корректным `state`

#### Сценарий: OAuth callback

- **WHEN** Google-режим и внешний HTTP-клиент (браузер) вызывает `GET /api/auth/google/callback` с параметрами `code` и `state` при успешной авторизации у Google
- **THEN** сервер SHALL обменять `code` на токен(ы), получить userinfo, применить allowlist и либо установить `kb_session` с редиректом в веб-интерфейс, либо отказать без сессии

### Requirement: Режим в ответе GET /api/auth/session

`GET /api/auth/session` SHALL возвращать при включённой веб-авторизации поле `auth_mode` со значением `password` или `google` (а также `auth_enabled: true`), чтобы веб-интерфейс мог выбрать форму входа без отдельного build-time API URL.

#### Сценарий: Веб-интерфейс определяет тип входа

- **WHEN** `auth_enabled` истинен и `GET /api/auth/session` вызывается с клиента
- **THEN** ответ MUST содержать `auth_mode` в `{ password, google }` согласно фактическому конфигу сервера

### Requirement: Auth endpoints для login/logout/session

REST API MUST предоставлять для веб-клиента: `POST /api/auth/logout`, `GET /api/auth/session`. В **парольном** режиме MUST дополнительно предоставлять `POST /api/auth/login` (создание сессии по учётным данным). В **Google-режиме** MUST предоставлять `GET /api/auth/google` и `GET /api/auth/google/callback` вместо выдачи сессии через `POST /api/auth/login`, при этом `POST /api/auth/login` SHALL NOT создавать сессию (SHALL отвечать ошибкой, указывающей, что используется вход через Google).

#### Сценарий: Эндпоинт login выдаёт сессию (пароль)

- **WHEN** парольный режим и клиент отправляет корректные credentials на `POST /api/auth/login`
- **THEN** сервер создаёт сессию и устанавливает cookie в ответе

#### Сценарий: Эндпоинт session отражает текущий статус

- **WHEN** клиент вызывает `GET /api/auth/session`
- **THEN** сервер возвращает статус аутентификации текущей сессии (и флаг, что веб-авторизация включена, если применимо)

#### Сценарий: Эндпоинт logout завершает сессию

- **WHEN** клиент вызывает `POST /api/auth/logout`
- **THEN** сервер инвалидирует текущую сессию и очищает auth-cookie

#### Сценарий: Google-режим без парольного login

- **WHEN** Google-режим и клиент вызывает `POST /api/auth/login` с JSON credentials
- **THEN** сессия SHALL NOT создаваться
