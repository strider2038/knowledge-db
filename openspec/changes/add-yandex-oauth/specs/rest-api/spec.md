## Purpose

REST API: мульти-способная веб-авторизация и OAuth-провайдеры.

## ADDED Requirements

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

### Requirement: Список способов входа в GET /api/auth/session

`GET /api/auth/session` при `auth_enabled: true` MUST возвращать `auth_methods` — массив строк из `{ password, google, yandex }`, отражающий **все** настроенные способы. Поле `auth_mode` MAY дублировать единственный способ или значение `multi` при нескольких (deprecated, см. design).

#### Сценарий: Несколько способов

- **WHEN** настроены пароль и Google OAuth
- **THEN** `auth_methods` MUST быть `["password","google"]` (фиксированный порядок: `password`, `google`, `yandex` — только включённые)

#### Сценарий: Один способ

- **WHEN** настроен только Google OAuth
- **THEN** `auth_methods` MUST быть `["google"]`; `auth_mode` MAY быть `"google"`

## MODIFIED Requirements

### Requirement: Опциональная защита API сессией

При `auth_enabled` REST API SHALL требовать валидную cookie на `/api/*`, **кроме** `/api/auth/*` (все auth-маршруты, включая login, logout, session, google/*, yandex/*) и health/readiness. Не зависит от того, сколько способов входа настроено.

#### Сценарий: Доступ к API без сессии при включённой авторизации

- **WHEN** защищённый `/api/*` без cookie
- **THEN** `401 Unauthorized`

#### Сценарий: Доступ к API с валидной сессией

- **WHEN** защищённый `/api/*` с валидной cookie
- **THEN** обычная обработка

### Requirement: Эндпоинты Google OAuth

В Google-режиме заменяется на: если Google OAuth **настроен**, MUST быть `GET /api/auth/google` и `GET /api/auth/google/callback` с тем же поведением, что раньше. Если Google не настроен — 4xx на эти пути. Наличие Yandex или пароля MUST NOT отключать Google endpoints.

#### Сценарий: Старт OAuth

- **WHEN** Google настроен, `GET /api/auth/google`
- **THEN** редирект на Google authorize с `state`

#### Сценарий: OAuth callback

- **WHEN** Google настроен, успешный callback
- **THEN** обмен code, allowlist, сессия или отказ

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
