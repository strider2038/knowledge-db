## Purpose

REST API для CRUD операций с узлами базы знаний, полнотекстового и ключевого поиска. В scaffold — минимальный набор эндпоинтов.
## Requirements
### Requirement: Конфигурация через KB_DATA_PATH

API ДОЛЖЕН (SHALL) использовать путь к базе из переменной окружения KB_DATA_PATH.

#### Сценарий: Запуск без KB_DATA_PATH

- **WHEN** `kb serve` запущен без KB_DATA_PATH
- **THEN** сервер возвращает ошибку конфигурации или не стартует

### Requirement: CRUD узлов

API MUST предоставлять эндпоинты для создания, чтения, обновления и удаления узлов (в scaffold — каркас/заглушки). Добавляется поддержка DELETE для удаления узла и POST /move для перемещения.

#### Сценарий: Получение узла по пути

- **WHEN** GET /api/nodes/{path}
- **THEN** возвращается узел или 404

#### Сценарий: Получение дерева тем

- **WHEN** GET /api/tree
- **THEN** возвращается иерархическое дерево тем и подтем

#### Сценарий: Удаление узла

- **WHEN** DELETE /api/nodes/{path}
- **THEN** узел (файл .md и директория вложений) удаляется, возвращается `{ path, deleted: true }` или 404

#### Сценарий: Перемещение узла

- **WHEN** POST /api/nodes/{path}/move с `{ target_path: "new/topic/slug" }`
- **THEN** узел перемещается по указанному пути, промежуточные директории создаются рекурсивно, возвращается обновлённый объект узла, 409 при конфликте

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

API MUST поддерживать частичное обновление метаданных узла через `PATCH /api/nodes/{path...}` и принимать одно или несколько полей из набора: `manual_processed`, `title`, `keywords`. Для неподдерживаемых полей API MUST возвращать `400 Bad Request`. Для некорректных типов значений API MUST возвращать `400 Bad Request`.

Сервер MUST нормализовать значения перед сохранением:
- `title`: trim; пустая строка удаляет поле `title` из frontmatter.
- `keywords`: trim каждого элемента, удаление пустых значений, дедупликация с сохранением порядка.
- `manual_processed`: boolean, при `false` допускается снятие флага согласно принятому представлению optional bool.

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

API ДОЛЖЕН (SHALL) предоставлять endpoints для управления индексом: `POST /api/index/rebuild` — полная перестройка индекса (запускает SyncWorker ManualRebuild); `GET /api/index/status` — состояние индекса (total_nodes, total_chunks, embedding_model, keyword_index, last_indexed_at, status). Оба endpoint MUST возвращать 503 при `KB_EMBEDDING_ENABLED=false`.

#### Scenario: Запуск перестройки индекса

- **WHEN** `POST /api/index/rebuild`
- **THEN** запускается полная переиндексация, возвращается 202 Accepted

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
