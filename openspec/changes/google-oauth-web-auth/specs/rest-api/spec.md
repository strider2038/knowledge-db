## Purpose

Delta для capability `rest-api`: Google OAuth маршруты, уточнение условий «включённой авторизации» и auth endpoints.

## Requirements

## ADDED Requirements

### Requirement: Google OAuth endpoints

В Google-режиме (см. `web-session-auth`) REST API MUST предоставлять `GET /api/auth/google` (инициация редиректа на Google) и `GET /api/auth/google/callback` (приём `code`/`state`). Эти маршруты SHALL располагаться под `/api/auth/` и SHALL быть исключены из mandatory session check так же, как `POST /api/auth/login` в парольном режиме (до установки сессии).

#### Сценарий: Старт OAuth

- **WHEN** Google-режим и клиент запрашивает `GET /api/auth/google` с валидным `Origin` (при настроенном CORS)
- **THEN** сервер SHALL отвечать редиректом (3xx) на Google authorization URL с корректным `state`

#### Сценарий: Callback OAuth

- **WHEN** Google-режим и внешний HTTP-клиент (браузер) вызывает `GET /api/auth/google/callback` с параметрами `code` и `state` при успешной авторизации у Google
- **THEN** сервер SHALL обменять `code` на токен(ы), получить userinfo, применить allowlist и либо установить `kb_session` с редиректом в веб-интерфейс, либо отказать без сессии

### Requirement: Режим в ответе GET /api/auth/session

`GET /api/auth/session` SHALL возвращать для включённой веб-авторизации поле `auth_mode` со значением `password` или `google` (а также `auth_enabled: true`), чтобы Web UI могла выбрать форму входа без отдельного build-time API URL.

#### Сценарий: Web UI определяет тип входа

- **WHEN** `auth_enabled` истинен и `GET /api/auth/session` вызывается с клиента
- **THEN** ответ MUST содержать `auth_mode` в `{ password, google }` согласно фактическому конфигу сервера

## MODIFIED Requirements

### Requirement: Опциональная защита API сессией

При включённой авторизации (парольный режим: `KB_LOGIN` и `KB_PASSWORD` заданы; **или** Google-режим: полный набор Google OAuth и `KB_AUTH_ALLOWED_EMAILS` при пустых `KB_LOGIN`/`KB_PASSWORD`, см. `web-session-auth`) REST API ДОЛЖЕН (SHALL) требовать валидную сессионную cookie для доступа к защищённым endpoint `/api/*`, **кроме** явно разрешённых путей аутентификации: `POST /api/auth/login` (в парольном режиме), `GET /api/auth/google` и `GET /api/auth/google/callback` (в Google-режиме), а также health/readiness. Маршруты `/api/*`, включая `/api/assets/*`, SHALL требовать сессию, если не относятся к перечисленным исключениям. При отсутствии или невалидности сессии на защищённом пути сервер MUST возвращать `401 Unauthorized`.

#### Сценарий: Доступ к API без сессии при включённой авторизации

- **WHEN** клиент вызывает защищённый endpoint `/api/*` (не из исключений) без валидной cookie-сессии
- **THEN** сервер возвращает `401 Unauthorized`

#### Сценарий: Доступ к API с валидной сессией

- **WHEN** клиент вызывает защищённый endpoint `/api/*` с валидной cookie-сессией
- **THEN** сервер обрабатывает запрос по обычной бизнес-логике endpoint

### Requirement: Auth endpoints для login/logout/session

REST API MUST предоставлять для web-клиента: `POST /api/auth/logout`, `GET /api/auth/session`. В **парольном** режиме MUST дополнительно предоставлять `POST /api/auth/login` (создание сессии по credentials). В **Google-режиме** MUST предоставлять `GET /api/auth/google` и `GET /api/auth/google/callback` вместо выдачи сессии через `POST /api/auth/login`, при этом `POST /api/auth/login` SHALL NOT создавать сессию (SHALL отвечать ошибкой, указывающей, что используется Google-вход).

#### Сценарий: Login endpoint выдаёт сессию (пароль)

- **WHEN** парольный режим и клиент отправляет корректные credentials на `POST /api/auth/login`
- **THEN** сервер создаёт сессию и устанавливает cookie в ответе

#### Сценарий: Session endpoint отражает текущий статус

- **WHEN** клиент вызывает `GET /api/auth/session`
- **THEN** сервер возвращает статус аутентификации текущей сессии (и флаг, что web-auth включён, если применимо)

#### Сценарий: Logout endpoint завершает сессию

- **WHEN** клиент вызывает `POST /api/auth/logout`
- **THEN** сервер инвалидирует текущую сессию и очищает auth-cookie

#### Сценарий: Google-режим без парольного login

- **WHEN** Google-режим и клиент вызывает `POST /api/auth/login` с JSON credentials
- **THEN** сессия SHALL NOT создаваться
