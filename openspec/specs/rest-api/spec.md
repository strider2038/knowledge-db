## Purpose

REST API для CRUD операций с узлами базы знаний, полнотекстового и ключевого поиска. В scaffold — минимальный набор эндпоинтов.
## Requirements
### Requirement: Конфигурация через KB_DATA_PATH

API ДОЛЖЕН (SHALL) использовать путь к базе из переменной окружения KB_DATA_PATH.

#### Сценарий: Запуск без KB_DATA_PATH

- **WHEN** `kb serve` запущен без KB_DATA_PATH
- **THEN** сервер возвращает ошибку конфигурации или не стартует

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

### Requirement: Обновление описания узла из источника

REST API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/nodes/{path}/refresh-description` для обновления описания существующего узла на основе его `source_url`. Endpoint MUST загружать текущий узел, требовать наличие `source_url`, запускать тот же алгоритм классификации и генерации digest, что используется при ingestion внешних источников, и сохранять обновлённый markdown-файл узла. Ответ MUST содержать обновлённый объект узла. Если `source_url` отсутствует, endpoint MUST возвращать 400. Если узел не найден, endpoint MUST возвращать 404.

Endpoint MUST обновлять описательные поля: `annotation`, `keywords`, `source_kind`, `content_profile` и markdown-тело digest. Endpoint MAY обновить `type`, если классификация показывает, что текущий тип был ошибочным, например новость `type=link` должна стать `type=note` с `content_profile=brief_digest`. Endpoint MUST сохранять `created`, `source_url`, `manual_processed` и пользовательские поля, не относящиеся к описанию источника. Endpoint SHOULD сохранять существующие `source_author` и `source_date`, если новый источник не даёт более точных значений.

#### Scenario: Обновление repository link

- **WHEN** клиент вызывает `POST /api/nodes/programming/golang/packages/go-libraries-runnable-manager/refresh-description` для узла с `source_url` на GitHub-репозиторий
- **THEN** API обновляет узел как `type=link`, `source_kind=repository`, `content_profile=repository_profile` и возвращает обновлённый объект узла

#### Scenario: Обновление длинной статьи как conceptual digest

- **WHEN** клиент вызывает refresh-description для узла с `source_url` на длинную статью, которая не хранится полной копией
- **THEN** API обновляет узел как `type=note`, `source_kind=article`, `content_profile=conceptual_digest` и сохраняет markdown-тело digest

#### Scenario: Исправление новости, ошибочно сохранённой как link

- **WHEN** клиент вызывает refresh-description для узла `type=link` с `source_url` на новостную публикацию
- **THEN** API MAY изменить тип на `note`, установить `source_kind=news`, `content_profile=brief_digest` и сохранить краткое markdown-тело digest

#### Scenario: Узел без source_url

- **WHEN** клиент вызывает refresh-description для узла без `source_url`
- **THEN** API возвращает 400 с сообщением, что обновление из источника невозможно

#### Scenario: Узел не найден

- **WHEN** клиент вызывает refresh-description для неизвестного пути
- **THEN** API возвращает 404

#### Scenario: Ошибка LLM или fetch источника

- **WHEN** источник недоступен или LLM-конфигурация отсутствует
- **THEN** API возвращает ошибку 503 или 502 с диагностируемым сообщением и не изменяет markdown-файл узла

#### Scenario: Переиндексация обновлённого узла

- **WHEN** refresh-description успешно сохраняет узел
- **THEN** API инициирует переиндексацию этого узла тем же механизмом, который используется после изменения узлов

### Requirement: Единый API фоновых jobs

REST API ДОЛЖЕН (SHALL) предоставлять единый контракт для запуска и наблюдения долгих операций через:
- `POST /api/jobs`
- `GET /api/jobs/{id}`
- `GET /api/jobs/{id}/logs?after=N`

`POST /api/jobs` MUST принимать `{ "type": string, "target": string, "options"?: object }` и возвращать snapshot job с полями:
`id`, `type`, `target`, `status` (`queued|running|success|error`), `stage`, `error`, `started_at`, `finished_at`, `meta`, `next_offset`.

`GET /api/jobs/{id}/logs` MUST возвращать инкрементальные записи логов с полями:
`offset`, `stream` (`system|stdout|stderr`), `text`, `timestamp`, а также `next_offset`.
Параметр `after` MUST ограничивать выдачу только новыми записями после указанного offset.

V1 MAY хранить jobs только в памяти процесса (без персистентности между рестартами).

#### Сценарий: Запуск job через generic API

- **WHEN** клиент вызывает `POST /api/jobs` с валидными `type` и `target`
- **THEN** API создаёт job, возвращает snapshot со статусом `queued` или `running`

#### Сценарий: Неизвестный тип job

- **WHEN** клиент вызывает `POST /api/jobs` с неподдерживаемым `type`
- **THEN** API возвращает `400 Bad Request` с диагностируемым сообщением

#### Сценарий: Инкрементальные логи

- **WHEN** клиент вызывает `GET /api/jobs/{id}/logs?after=N`
- **THEN** API возвращает только записи с `offset > N` и корректный `next_offset`

#### Сценарий: Переходы статуса

- **WHEN** job выполняется в фоне
- **THEN** `GET /api/jobs/{id}` отражает переходы `queued -> running -> success|error`, включая `stage` и `error` при сбое

### Requirement: Job type для refresh-description

REST API ДОЛЖЕН (SHALL) поддерживать запуск `refresh-description` в job-модели через `POST /api/jobs` с `type=refresh_description`.
Выполнение job SHOULD публиковать системные этапы как минимум: `start`, `classify`, `fetch/meta`, `llm`, `retry_digest_if_needed`, `save`, `sync`, `done` (или `error`).

Для профильных `link`-узлов с digest-профилями (`repository_profile`, `learning_resource_profile` и т.п.) job MUST предотвращать сохранение пустого markdown-тела: при пустом результате MUST выполнить один retry с усиленной инструкцией к LLM; при повторном пустом результате MUST завершиться `error` без мутации узла.

#### Сценарий: Async refresh через jobs

- **WHEN** клиент запускает `POST /api/jobs` с `type=refresh_description`
- **THEN** обновление выполняется в фоне, прогресс доступен через `GET /api/jobs/{id}` и `GET /api/jobs/{id}/logs`

#### Сценарий: Пустой digest для профильного link

- **WHEN** refresh job получил пустой `content` для профильного `type=link`
- **THEN** выполняется один retry; при повторной пустоте job завершается `error`, а файл узла не изменяется

### Requirement: Legacy-совместимость async endpoint-ов

Существующие endpoint-ы асинхронных операций (`node-normalization`, `node-dump-images`, текущий async-поток перевода и др.) ДОЛЖНЫ (SHALL) сохраняться как compatibility layer минимум на переходный релиз и работать поверх общего job-слоя.
Legacy-ответы MUST сохранять прежнюю форму для старых клиентов.

#### Сценарий: Старый клиент использует legacy endpoint

- **WHEN** клиент вызывает legacy async endpoint
- **THEN** операция выполняется через общий `JobManager`, а ответ возвращается в прежнем формате endpoint-а

### Requirement: Ingestion

API MUST предоставлять эндпоинт POST /api/ingest для приёма текста и передачи в ingestion pipeline. Тело запроса MUST поддерживать поля: text (обязательно), source_url (опционально), source_author (опционально), type_hint (опционально). Допустимые значения type_hint: "auto", "article", "link", "note". При отсутствии или неизвестном значении type_hint MUST трактовать как "auto".

#### Сценарий: Отправка текста

- **WHEN** POST /api/ingest с телом { "text": "..." }
- **THEN** текст передаётся в Ingester, возвращается результат

#### Сценарий: Отправка с type_hint

- **WHEN** POST /api/ingest с телом { "text": "...", "type_hint": "article" }
- **THEN** текст и type_hint передаются в Ingester, возвращается результат

### Requirement: Список узлов с фильтрами

API MUST поддерживать GET /api/nodes с query-параметрами: path (путь темы, пустой = вся база), recursive (bool, по умолчанию false), q (подстрока поиска в title, keywords, annotation), type (article, link, note — через запятую), manual_processed (опционально: true или false — только узлы с соответствующим флагом; при отсутствии параметра возвращаются все узлы независимо от флага), labels (опционально: список меток через запятую — только узлы, содержащие **все** указанные метки; сравнение без учёта регистра после нормализации), limit, offset (пагинация). При recursive=true возвращаются узлы всего поддерева. Ответ MUST содержать nodes (массив с path, title, type, created, source_url, translations, manual_processed, labels) и total (общее количество до пагинации). Узлы без поля manual_processed в хранилище MUST трактоваться как manual_processed=false в JSON. Узлы без поля labels MUST возвращать labels как пустой массив `[]`. Переводы (slug.lang.md) не включаются как отдельные узлы. Параметр q MUST NOT искать по полю labels.

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

#### Сценарий: Фильтр по одной метке

- **WHEN** GET /api/nodes?path=&recursive=true&labels=favorite
- **THEN** возвращаются только узлы, у которых в labels есть метка favorite (без учёта регистра)

#### Сценарий: Фильтр по нескольким меткам (AND)

- **WHEN** GET /api/nodes?path=&recursive=true&labels=favorite,review
- **THEN** возвращаются только узлы, содержащие и favorite, и review

#### Сценарий: Пустые сегменты labels игнорируются

- **WHEN** GET /api/nodes?labels=favorite,,review
- **THEN** API трактует запрос как labels=favorite,review

### Requirement: Метаданные узла содержат manual_processed

Ответ GET узла по пути (и любые ответы с полным телом метаданных узла, используемые веб-клиентом) MUST содержать boolean поле manual_processed (false, если в файле поле отсутствует).

#### Сценарий: Чтение узла без поля в файле

- **WHEN** GET узла для .md без ключа manual_processed
- **THEN** в JSON manual_processed равен false

### Requirement: Метаданные узла содержат labels

Ответ GET узла по пути (и любые ответы с полным телом метаданных узла, используемые веб-клиентом) MUST содержать поле labels (массив строк). Если в файле поле отсутствует, JSON MUST содержать `labels: []`.

#### Сценарий: Чтение узла без labels в файле

- **WHEN** GET узла для .md без ключа labels
- **THEN** в JSON labels равен пустому массиву

### Requirement: Подсказки существующих labels

API MUST предоставлять GET /api/label-suggestions, возвращающий JSON `{ "labels": ["...", ...] }` — уникальные метки, встречающиеся хотя бы у одного узла базы, отсортированные для UI (например, по алфавиту). Список MUST NOT включать keywords. Ответ MAY ограничиваться разумным лимитом (например, 500 записей).

#### Сценарий: Получение подсказок

- **WHEN** клиент вызывает GET /api/label-suggestions
- **THEN** возвращается массив уникальных строк labels из frontmatter узлов

### Requirement: Обновление метаданных узла (PATCH)

API MUST поддерживать частичное обновление метаданных узла через `PATCH /api/nodes/{path...}` и принимать одно или несколько полей из набора: `manual_processed`, `title`, `keywords`, `labels`. Для неподдерживаемых полей API MUST возвращать `400 Bad Request`. Для некорректных типов значений API MUST возвращать `400 Bad Request`.

Сервер MUST нормализовать значения перед сохранением:
- `title`: trim; пустая строка удаляет поле `title` из frontmatter.
- `keywords`: trim каждого элемента, удаление пустых значений, дедупликация с сохранением порядка.
- `manual_processed`: boolean, при `false` допускается снятие флага согласно принятому представлению optional bool.
- `labels`: массив строк; нормализация по правилам knowledge-storage (trim, dedupe case-insensitive, лимиты); пустой массив удаляет `labels` из frontmatter.

#### Сценарий: Установка флага manual_processed

- **WHEN** клиент отправляет PATCH с `{ "manual_processed": true }`
- **THEN** в frontmatter сохраняется `manual_processed: true`, ответ содержит обновлённый узел

#### Сценарий: Снятие флага manual_processed

- **WHEN** клиент отправляет PATCH с `{ "manual_processed": false }`
- **THEN** флаг manual_processed снимается или сохраняется как false согласно реализации, ответ содержит обновлённый узел

#### Сценарий: Обновление title

- **WHEN** клиент отправляет PATCH с `{ "title": "  New title  " }`
- **THEN** сервер сохраняет `title: "New title"` и возвращает обновлённый узел

#### Сценарий: Очистка title

- **WHEN** клиент отправляет PATCH с `{ "title": "   " }`
- **THEN** поле `title` удаляется из frontmatter

#### Сценарий: Обновление keywords с повторами и пустыми значениями

- **WHEN** клиент отправляет PATCH с `{ "keywords": ["go", "  kubernetes ", "go", ""] }`
- **THEN** сервер сохраняет `keywords: ["go", "kubernetes"]` и возвращает обновлённый узел

#### Сценарий: Обновление labels

- **WHEN** клиент отправляет PATCH с `{ "labels": ["  favorite ", "Favorite", "review"] }`
- **THEN** сервер сохраняет нормализованный список (например `["favorite", "review"]`) и возвращает узел с полем labels в JSON

#### Сценарий: Очистка labels

- **WHEN** клиент отправляет PATCH с `{ "labels": [] }`
- **THEN** ключ labels удаляется из frontmatter, в JSON labels равен `[]`

#### Сценарий: Неподдерживаемое поле

- **WHEN** клиент отправляет PATCH с неизвестным полем, например `{ "unexpected": "x" }`
- **THEN** API возвращает `400 Bad Request` и не изменяет файл узла

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

При `auth_enabled` (см. `web-session-auth`) REST API SHALL требовать валидную cookie на `/api/*`, **кроме** `/api/auth/*` (все auth-маршруты, включая login, logout, session, google/*, yandex/*) и health/readiness. Не зависит от того, сколько способов входа настроено.

#### Сценарий: Доступ к API без сессии при включённой авторизации

- **WHEN** защищённый `/api/*` без cookie
- **THEN** `401 Unauthorized`

#### Сценарий: Доступ к API с валидной сессией

- **WHEN** защищённый `/api/*` с валидной cookie
- **THEN** обычная обработка

### Requirement: Эндпоинты Yandex OAuth

Если Yandex OAuth настроен (см. `web-session-auth`), REST API MUST предоставлять `GET /api/auth/yandex` и `GET /api/auth/yandex/callback` под `/api/auth/`, exempt от проверки сессии до установки cookie. Если Yandex **не** настроен, запросы к этим путям SHALL отвечать ошибкой (4xx), указывающей, что провайдер не сконфигурирован.

#### Сценарий: Старт Yandex OAuth

- **WHEN** Yandex настроен, клиент вызывает `GET /api/auth/yandex`
- **THEN** редирект 3xx на authorize Yandex с подписанным `state`

#### Сценарий: Yandex OAuth callback

- **WHEN** Yandex настроен, callback с валидными `code` и `state`
- **THEN** обмен на токен, userinfo, allowlist, установка `kb_session` или отказ

#### Сценарий: Yandex не настроен

- **WHEN** Yandex env неполный/отсутствует, клиент вызывает `GET /api/auth/yandex`
- **THEN** сервер SHALL NOT инициировать OAuth у Yandex (4xx)

### Requirement: Эндпоинты Google OAuth

Если Google OAuth **настроен**, MUST быть `GET /api/auth/google` и `GET /api/auth/google/callback` с тем же поведением, что раньше. Если Google не настроен — 4xx на эти пути. Наличие Yandex или пароля MUST NOT отключать Google endpoints.

#### Сценарий: Старт OAuth

- **WHEN** Google настроен, `GET /api/auth/google`
- **THEN** редирект на Google authorize с `state`

#### Сценарий: OAuth callback

- **WHEN** Google настроен, успешный callback
- **THEN** обмен code, allowlist, сессия или отказ

### Requirement: Список способов входа в GET /api/auth/session

`GET /api/auth/session` при `auth_enabled: true` MUST возвращать `auth_methods` — массив строк из `{ password, google, yandex }`, отражающий **все** настроенные способы. Поле `auth_mode` MAY дублировать единственный способ или значение `multi` при нескольких (deprecated).

#### Сценарий: Несколько способов

- **WHEN** настроены пароль и Google OAuth
- **THEN** `auth_methods` MUST быть `["password","google"]` (фиксированный порядок: `password`, `google`, `yandex` — только включённые)

#### Сценарий: Один способ

- **WHEN** настроен только Google OAuth
- **THEN** `auth_methods` MUST быть `["google"]`; `auth_mode` MAY быть `"google"`

### Requirement: Режим в ответе GET /api/auth/session

`GET /api/auth/session` SHALL возвращать при `auth_enabled: true` массив `auth_methods` — основной контракт для UI. Поле `auth_mode` MAY сохраняться для обратной совместимости (единственный способ или `multi`). Клиенты MUST предпочитать `auth_methods` при его наличии.

#### Сценарий: Веб-интерфейс определяет доступные способы

- **WHEN** `auth_enabled` и `GET /api/auth/session`
- **THEN** ответ MUST содержать `auth_methods`; UI SHALL строить экран входа по этому массиву

### Requirement: Auth endpoints для login/logout/session

REST API MUST предоставлять `POST /api/auth/logout`, `GET /api/auth/session` при `auth_enabled`. `POST /api/auth/login` MUST работать (создавать сессию), если пароль **настроен**, независимо от OAuth. OAuth-маршруты каждого провайдера MUST регистрироваться и работать только при настройке провайдера. `POST /api/auth/login` при **не**настроенном пароле SHALL отвечать 400 «password auth not configured». При настроенном OAuth и отключённом пароле login SHALL NOT создавать сессию.

#### Сценарий: Эндпоинт login выдаёт сессию (пароль)

- **WHEN** пароль настроен, верные credentials
- **THEN** cookie сессии

#### Сценарий: Login при только OAuth

- **WHEN** пароль не настроен, `POST /api/auth/login`
- **THEN** сессия SHALL NOT создаваться; ошибка конфигурации

#### Сценарий: Эндпоинт session отражает текущий статус

- **WHEN** `GET /api/auth/session`
- **THEN** `authenticated`, `auth_enabled`, `auth_methods`

#### Сценарий: Эндпоинт logout завершает сессию

- **WHEN** `POST /api/auth/logout`
- **THEN** инвалидация и очистка cookie

#### Сценарий: OAuth-режим без парольного login

- **WHEN** только Google/Yandex, без пароля, `POST /api/auth/login`
- **THEN** сессия SHALL NOT создаваться

### Requirement: Git-статус API

API MUST предоставлять эндпоинт `GET /api/git/status`, возвращающий информацию о незакоммиченных изменениях в git-репозитории базы. Ответ MUST содержать `has_changes` (boolean) и `changed_files` (число). При отключённом git MUST возвращаться 503.

#### Сценарий: Есть незакоммиченные изменения

- **WHEN** GET /api/git/status и git имеет modified/untracked/deleted файлы
- **THEN** возвращается `{ "has_changes": true, "changed_files": N }`

#### Сценарий: Нет изменений

- **WHEN** GET /api/git/status и рабочий каталог чист
- **THEN** возвращается `{ "has_changes": false, "changed_files": 0 }`

#### Сценарий: Git отключён

- **WHEN** GET /api/git/status и KB_GIT_DISABLED=true
- **THEN** возвращается 503

### Requirement: Git-коммит API

API MUST предоставлять эндпоинт `POST /api/git/commit` для коммита всех изменений с автогенерацией commit message через LLM. Тело MAY содержать `{ "message"?: string }`. При отсутствии message MUST вызываться LLM (OpenAI Responses API) для генерации conventional commit message на основе `git diff --stat`. При недоступности LLM MUST использоваться fallback `chore: manual commit via UI`. Выполняет git add -A, commit, push. При отключённом git — 503.

#### Сценарий: Автогенерация commit message

- **WHEN** POST /api/git/commit без message и есть изменения
- **THEN** LLM генерирует conventional commit message, выполняется commit+push, возвращается `{ message, committed: true }`

#### Сценарий: Ручной commit message

- **WHEN** POST /api/git/commit с `{ "message": "fix: typo" }`
- **THEN** используется указанное сообщение, LLM не вызывается

#### Сценарий: Нет изменений

- **WHEN** POST /api/git/commit и нет незакоммиченных изменений
- **THEN** возвращается `{ "committed": false, "message": "no changes to commit" }`

### Requirement: Git-sync API

API MUST предоставлять эндпоинт `POST /api/git/sync` для ручной синхронизации рабочей копии базы с удалённым репозиторием: выполняется та же операция, что и у фонового git-sync (fetch и merge с `origin/main`, см. `GitCommitter.Sync`). Тело запроса MAY быть пустым JSON `{}`. При успехе ответ MUST содержать `synced: true` и поле `message` с кратким описанием для UI. При отключённом git — 503. При ошибке git (сеть, merge и т.п.) — 5xx с диагностическим текстом. Если на сервере включён embedding-индекс, после успешного sync система SHOULD поставить в очередь индексатору полную сверку с файловой системой (как при событии после pull).

#### Сценарий: Успешный sync

- **WHEN** POST /api/git/sync и git включён, операция fetch/merge завершается без ошибки
- **THEN** возвращается 200 и JSON с `synced: true` и непустым `message`

#### Сценарий: Git отключён

- **WHEN** POST /api/git/sync и KB_GIT_DISABLED=true
- **THEN** возвращается 503

### Requirement: API управления чат-сессиями

Система SHALL предоставлять REST API для создания, получения списка, чтения, продолжения, удаления и переименования чат-сессий.

#### Scenario: Получение списка чатов

- **WHEN** клиент запрашивает список чат-сессий
- **THEN** система MUST вернуть упорядоченный список сессий с id, title, updatedAt и краткими метаданными

#### Scenario: Открытие чата

- **WHEN** клиент запрашивает конкретную сессию по id
- **THEN** система SHALL вернуть сообщения сессии в пользовательском представлении без служебных summary-сегментов

#### Scenario: Удаление чата

- **WHEN** клиент удаляет чат-сессию по id
- **THEN** система MUST удалить сессию и связанные данные из SQLite и вернуть успешный статус операции

#### Scenario: Переименование чата

- **WHEN** клиент отправляет новое название чата
- **THEN** система MUST обновить title сессии и вернуть обновлённые метаданные

### Requirement: API отправки сообщения в сессию

Система MUST принимать новое сообщение в выбранную сессию и возвращать ответ ассистента, сформированный с учётом истории и ограничений контекста.

#### Scenario: Сообщение в существующий чат

- **WHEN** клиент отправляет сообщение в существующую сессию
- **THEN** система SHALL сохранить сообщение, выполнить генерацию ответа и вернуть обновлённое состояние чата

#### Scenario: Ошибка неизвестной сессии

- **WHEN** клиент отправляет сообщение в несуществующую сессию
- **THEN** система MUST вернуть 404 с диагностируемой ошибкой

### Requirement: Endpoint чатбота POST /api/chat

API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/chat` для RAG-чатбота с поддержкой чат-сессий. Запрос MUST содержать `message` (string) и `session_id` активной сессии. Создание новой сессии выполняется через `POST /api/chats`. Запрос MAY содержать `source_paths` для ограничения ответа выбранными источниками. Ответ MUST быть streaming (SSE) и MUST использовать гибридный retrieval pipeline для поиска контекста (кроме режима `chat_memory`, см. capability rag-chat). Генерация LLM ответа SHOULD использовать OpenAI-compatible Chat Completions streaming. SSE response MUST не сжиматься gzip middleware. При `KB_EMBEDDING_ENABLED=false` MUST возвращать 503. При пустом `message` или отсутствии `session_id` MUST возвращать 400.

#### Scenario: Успешный запрос

- **WHEN** `POST /api/chat` с `{ "session_id": "...", "message": "..." }`
- **THEN** выполняется гибридный retrieval (в соответствующем режиме), возвращается SSE stream с источниками и токенами ответа

#### Scenario: Запрос по выбранным источникам

- **WHEN** `POST /api/chat` содержит `source_paths`
- **THEN** контекст ответа ограничивается указанными источниками

#### Scenario: SSE не сжимается

- **WHEN** клиент запрашивает `/api/chat` с `Accept-Encoding: gzip`
- **THEN** response не содержит `Content-Encoding: gzip`, имеет `Content-Type: text/event-stream` и может отдавать токены без gzip buffering

#### Scenario: Сервис недоступен

- **WHEN** `POST /api/chat` при `KB_EMBEDDING_ENABLED=false`
- **THEN** возвращается 503

### Requirement: Управление индексом

API ДОЛЖЕН (SHALL) предоставлять endpoints для управления индексом: `POST /api/index/rebuild` — полная перестройка индекса (запускает SyncWorker ManualRebuild асинхронно); `GET /api/index/status` — состояние индекса (total_nodes, total_chunks, embedding_model, keyword_index, last_indexed_at, status). Оба endpoint MUST возвращать 503 при `KB_EMBEDDING_ENABLED=false`. Эквивалентная синхронная офлайн-операция MUST быть доступна через CLI `kb rebuild-index` (см. `kb-cli`).

#### Scenario: Запуск перестройки индекса

- **WHEN** `POST /api/index/rebuild` при работающем сервере и включённых эмбеддингах
- **THEN** в очередь SyncWorker ставится ManualRebuild, возвращается 202 Accepted

#### Scenario: Проверка статуса индекса

- **WHEN** `GET /api/index/status`
- **THEN** возвращается JSON с метриками индекса, включая режим keyword_index

### Requirement: Endpoint гибридного поиска POST /api/search

API ДОЛЖЕН (SHALL) предоставлять endpoint `POST /api/search` для гибридного поиска по базе знаний. Запрос MUST содержать `query` (string). Запрос MAY содержать `type`, `path`, `recursive`, `manual_processed`, `limit`, `offset` и `mode`. Ответ MUST содержать `results`, `total`, `query`, `mode` и метаданные retrieval. Метаданные MUST содержать `keyword_index` и MAY содержать `query_rewrite`, если поиск использовал LLM-normalized query. Endpoint MUST возвращать 503, если индекс недоступен для гибридного поиска.

#### Scenario: Успешный гибридный поиск

- **WHEN** клиент отправляет `POST /api/search` с `{ "query": "sqlite vector search" }`
- **THEN** API возвращает JSON со списком ранжированных карточек нод и релевантных фрагментов

#### Scenario: Пустой запрос

- **WHEN** клиент отправляет `POST /api/search` с пустым `query`
- **THEN** API возвращает 400 с ошибкой валидации

#### Scenario: Фильтр по типу

- **WHEN** клиент отправляет `POST /api/search` с `type=["article"]`
- **THEN** API возвращает только article-ноды

#### Scenario: Индекс недоступен

- **WHEN** `KB_EMBEDDING_ENABLED=false` или индекс не инициализирован
- **THEN** `POST /api/search` возвращает 503

#### Scenario: Ответ содержит query rewrite metadata

- **WHEN** поиск использует LLM rewrite исходного запроса
- **THEN** JSON response содержит `meta.query_rewrite` с фактически использованным rewrite query

### Requirement: API получения логов нормализации

REST API SHALL предоставлять endpoint `GET /api/node-normalization/{id}/logs` для получения логов операции нормализации. Endpoint MUST поддерживать query-параметр `after` для инкрементального чтения и MUST возвращать только записи с offset больше `after`, а также `next_offset`.

#### Сценарий: Чтение логов с начала

- **WHEN** клиент вызывает `/api/node-normalization/{id}/logs` без `after`
- **THEN** API возвращает доступные записи начиная с минимального offset и `next_offset`

#### Сценарий: Инкрементальное чтение

- **WHEN** клиент вызывает `/api/node-normalization/{id}/logs?after=42`
- **THEN** API возвращает только записи с offset > 42

#### Сценарий: Операция не найдена

- **WHEN** клиент запрашивает логи для неизвестного `id`
- **THEN** API возвращает 404

### Requirement: API нормализации узла через Cursor Agent

REST API SHALL предоставлять endpoint запуска нормализации узла через Cursor Agent (например `POST /api/nodes/{path}/normalize`) и endpoint получения статуса операции (или эквивалентный контракт в ответе запуска). API MUST принимать path текущего узла, формировать промт нормализации на сервере, запускать Cursor Agent в серверном окружении и возвращать диагностируемый статус выполнения.

#### Сценарий: Успешный старт операции

- **WHEN** клиент вызывает endpoint нормализации для существующего узла при валидной конфигурации Cursor Agent
- **THEN** API возвращает статус запуска операции (running/accepted) и идентификатор для отслеживания

#### Сценарий: Cursor Agent не настроен

- **WHEN** endpoint нормализации вызван без доступного Cursor Agent
- **THEN** API возвращает ошибку конфигурации с понятным сообщением и MUST NOT запускать операцию

### Requirement: Серверный post-step sync после нормализации

После успешного завершения Cursor Agent API MUST запускать `sync` как обязательный post-step и SHALL возвращать клиенту итоговый статус с учётом шага sync.

#### Сценарий: Успешный sync после нормализации

- **WHEN** нормализация завершилась успешно и `sync` завершился успешно
- **THEN** API возвращает итог success

#### Сценарий: Ошибка sync после нормализации

- **WHEN** нормализация завершилась успешно, но `sync` завершился ошибкой
- **THEN** API возвращает ошибку шага sync с деталями

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

